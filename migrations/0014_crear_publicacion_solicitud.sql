-- 0014_crear_publicacion_solicitud.sql
-- Requisito: RF-06, RN-08, RN-02. Decisión D-20. Módulo 6 de docs/PLAN_CONSTRUCCION.md.
-- Reversible: sí (DROP de las 3 columnas, el CHECK, los 3 triggers y sus funciones).
-- Afecta: solicitud (fecha_servicio_deseada, franja_horaria_inicio, franja_horaria_fin; 1 trigger),
--   solicitud_objeto (2 triggers). Sin cambios de policies.
--
-- 1. Fecha y franja del servicio (D-20). La franja es opcional (null = "lo antes posible"); si
--    se carga, van las dos horas y el inicio es anterior al fin. dbFletway no tiene solicitudes
--    (verificado el 2026-10-03), así que la fecha nace NOT NULL.
-- 2. Sin edición (D-20): solicitud_update deja al Cliente cambiar cualquier columna, incluido el
--    estado. Un usuario sólo puede cancelar una solicitud publicada; para cambiar algo cancela y
--    publica otra. El alta nace publicada y sin la cotización deprecada.
-- 3. Objetos de la solicitud: el precio de cada oferta se calcula sobre ellos (RN-01, RN-02), así
--    que un usuario no los modifica ni los borra, y sólo agrega objetos mientras la solicitud
--    está publicada y sin ofertas (el alta inserta solicitud y objetos en una transacción).
-- 4. Un objeto del catálogo copia siempre peso, medidas y restricciones de objeto (RN-08), aunque
--    el request mande otros valores.
--
-- Las restricciones 2 y 3 aplican cuando el que escribe es el usuario (current_user =
-- 'authenticated') y no es Administrador. Las funciones SECURITY DEFINER que vengan después
-- (fn_aceptar_oferta, módulo 9) corren como su dueño y no quedan bloqueadas. Por eso estas
-- funciones de trigger NO son SECURITY DEFINER: necesitan ver el current_user real.

BEGIN;

-- 1. Fecha y franja -----------------------------------------------------------------------

ALTER TABLE solicitud
  ADD COLUMN fecha_servicio_deseada date NOT NULL,
  ADD COLUMN franja_horaria_inicio  time,
  ADD COLUMN franja_horaria_fin     time,
  ADD CONSTRAINT chk_solicitud_franja CHECK (
    (franja_horaria_inicio IS NULL AND franja_horaria_fin IS NULL)
    OR (franja_horaria_inicio IS NOT NULL AND franja_horaria_fin IS NOT NULL
        AND franja_horaria_inicio < franja_horaria_fin)
  );

-- 2. Solicitud sin edición ----------------------------------------------------------------

CREATE FUNCTION public.fn_proteger_solicitud()
RETURNS trigger
LANGUAGE plpgsql
SET search_path TO 'public'
AS $function$
DECLARE
  v_sin_estado solicitud;
BEGIN
  IF current_user <> 'authenticated' OR fn_es_administrador() THEN
    RETURN NEW;
  END IF;

  IF TG_OP = 'INSERT' THEN
    NEW.estado_codigo := 'publicada';
    NEW.cotizacion_estimada_monto := NULL;
    NEW.requiere_escalera := NULL;
    NEW.pisos_escalera := NULL;
    NEW.creado_en := now();
    RETURN NEW;
  END IF;

  v_sin_estado := NEW;
  v_sin_estado.estado_codigo := OLD.estado_codigo;
  IF v_sin_estado IS DISTINCT FROM OLD THEN
    RAISE EXCEPTION 'solicitud: no se puede editar; cancelala y publicá otra'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF NEW.estado_codigo IS DISTINCT FROM OLD.estado_codigo
     AND NOT (OLD.estado_codigo = 'publicada' AND NEW.estado_codigo = 'cancelada') THEN
    RAISE EXCEPTION 'solicitud: sólo se puede cancelar una solicitud publicada'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  RETURN NEW;
END;
$function$;

CREATE TRIGGER trg_proteger_solicitud
  BEFORE INSERT OR UPDATE ON solicitud
  FOR EACH ROW EXECUTE FUNCTION fn_proteger_solicitud();

-- 3. Objetos fijos una vez publicada ------------------------------------------------------

CREATE FUNCTION public.fn_proteger_solicitud_objeto()
RETURNS trigger
LANGUAGE plpgsql
SET search_path TO 'public'
AS $function$
BEGIN
  IF current_user <> 'authenticated' OR fn_es_administrador() THEN
    RETURN COALESCE(NEW, OLD);
  END IF;

  IF TG_OP <> 'INSERT' THEN
    RAISE EXCEPTION 'solicitud_objeto: los objetos de una solicitud no se modifican'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM solicitud s WHERE s.id = NEW.solicitud_id AND s.estado_codigo = 'publicada')
     OR EXISTS (SELECT 1 FROM oferta o WHERE o.solicitud_id = NEW.solicitud_id) THEN
    RAISE EXCEPTION 'solicitud_objeto: sólo se agregan objetos a una solicitud publicada y sin ofertas'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  RETURN NEW;
END;
$function$;

CREATE TRIGGER trg_proteger_solicitud_objeto
  BEFORE INSERT OR UPDATE OR DELETE ON solicitud_objeto
  FOR EACH ROW EXECUTE FUNCTION fn_proteger_solicitud_objeto();

-- 4. Copia del catálogo -------------------------------------------------------------------

CREATE FUNCTION public.fn_copiar_objeto_catalogo()
RETURNS trigger
LANGUAGE plpgsql
SET search_path TO 'public'
AS $function$
DECLARE
  v_objeto objeto;
BEGIN
  IF NEW.objeto_id IS NOT NULL THEN
    -- Si el objeto no existe no se toca nada: la FK rechaza la fila con un error claro.
    SELECT * INTO v_objeto FROM objeto o WHERE o.id = NEW.objeto_id;
    IF FOUND THEN
      NEW.peso_unitario_kg := v_objeto.peso_estimado_kg;
      NEW.largo_m := v_objeto.largo_m;
      NEW.ancho_m := v_objeto.ancho_m;
      NEW.alto_m := v_objeto.alto_m;
      NEW.rotacion_horizontal := v_objeto.rotacion_horizontal;
      NEW.rotacion_vertical := v_objeto.rotacion_vertical;
      NEW.apilable := v_objeto.apilable;
    END IF;
  END IF;
  NEW.volumen_unitario_m3 := NULL; -- deprecada (0001)
  RETURN NEW;
END;
$function$;

-- Se llama trg_a_... para correr antes que trg_proteger_solicitud_objeto (los BEFORE corren
-- en orden alfabético); el orden no cambia el resultado, pero así la copia se ve primero.
CREATE TRIGGER trg_a_copiar_objeto_catalogo
  BEFORE INSERT ON solicitud_objeto
  FOR EACH ROW EXECUTE FUNCTION fn_copiar_objeto_catalogo();

COMMIT;

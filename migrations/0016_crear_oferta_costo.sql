-- 0016_crear_oferta_costo.sql
-- Requisito: RF-17, RN-01, RN-02. Decisiones D-15, D-23, D-24. Módulo 8 de
--   docs/PLAN_CONSTRUCCION.md.
-- Reversible: sí, mientras no haya ofertas (DROP de config_margen, oferta_costo, los triggers y
--   sus funciones; volver a agregar las columnas de costo a oferta y el UNIQUE original).
-- Afecta: config_margen (tabla nueva, 4 policies, seed); oferta_costo (tabla nueva, 2 policies);
--   oferta (se van 11 columnas de costo; el UNIQUE pasa a índice parcial; 3 triggers);
--   COMMENT de config_operacion.tiempo_espera_min.
--
-- APLICADA el 2026-10-03 vía apply_migration (confirmación humana explícita), después de
-- probarla en Supabase local (make db-local). dbFletway tenía 0 ofertas y 0 viajes.
--
-- 1. config_margen (D-15): margen de plataforma versionado como las demás config_*. La seed es
--    0 % porque el equipo todavía no definió otro valor (PLAN_CONSTRUCCION, módulo 8); cambiarlo
--    es cerrar la fila vigente e insertar otra.
-- 2. oferta_costo (D-23): el desglose de costo sale de oferta, que el Cliente lee, a una tabla
--    1 a 1 que sólo ven el Transportista dueño y el Administrador. Las columnas se mueven con el
--    mismo tipo y CHECK que tenían en 0005; oferta está vacía en dbFletway (verificar antes de
--    aplicar). Nadie la modifica ni la borra: no hay policies de UPDATE ni DELETE.
-- 3. Una oferta nace con su desglose: un trigger diferido rechaza, al hacer COMMIT, la oferta
--    que no tenga su fila en oferta_costo. Una llamada de PostgREST es una sola sentencia sobre
--    una tabla, así que no puede crear las dos filas: las ofertas se crean sólo por el backend,
--    que calcula el precio (RN-01). Al insertar, la oferta nace pendiente, con un vehículo del
--    Transportista y sobre una solicitud publicada.
-- 4. trg_proteger_campos_oferta (D-23): un usuario no cambia precio, viajes, ayudantes ni FK.
--    Los únicos cambios de estado permitidos: el Transportista dueño retira una oferta pendiente,
--    y el Cliente que cancela su solicitud pasa las pendientes a no_seleccionada (D-20).
--    Aceptar una oferta (módulo 9) lo hace fn_aceptar_oferta, SECURITY DEFINER, que no queda
--    bloqueada. Igual que en 0014, las funciones de trigger no son SECURITY DEFINER porque
--    necesitan ver el current_user real.
-- 5. "Retirala y creá otra" (D-23): el UNIQUE (solicitud, transportista, vehículo) impedía volver
--    a ofertar con el mismo vehículo después de retirar. Pasa a un índice único parcial que
--    ignora las ofertas retiradas.

BEGIN;

-- 1. config_margen ------------------------------------------------------------------------

CREATE TABLE config_margen (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  margen_pct     numeric(5,2) NOT NULL CHECK (margen_pct BETWEEN 0 AND 100),
  vigente_desde  timestamptz  NOT NULL DEFAULT now(),
  vigente_hasta  timestamptz,
  CONSTRAINT chk_vigencia_margen CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde)
);
ALTER TABLE config_margen ENABLE ROW LEVEL SECURITY;
CREATE UNIQUE INDEX idx_config_margen_vigente
  ON config_margen (vigente_hasta) WHERE vigente_hasta IS NULL;

-- Lectura abierta: el cálculo de la oferta corre con el rol del Transportista (D-02).
CREATE POLICY config_margen_select ON config_margen FOR SELECT USING (true);
CREATE POLICY config_margen_insert ON config_margen FOR INSERT WITH CHECK (fn_es_administrador());
CREATE POLICY config_margen_update ON config_margen FOR UPDATE USING (fn_es_administrador()) WITH CHECK (fn_es_administrador());
CREATE POLICY config_margen_delete ON config_margen FOR DELETE USING (fn_es_administrador());

INSERT INTO config_margen (margen_pct) VALUES (0);

-- El comentario de 0004 decía "reservado"; D-24 lo suma una vez por viaje.
COMMENT ON COLUMN config_operacion.tiempo_espera_min IS
  'Espera y acceso en el lugar; se suma una vez por viaje (D-24).';

-- 2. oferta_costo -------------------------------------------------------------------------

CREATE TABLE oferta_costo (
  oferta_id             uuid PRIMARY KEY REFERENCES oferta(id) ON DELETE CASCADE,
  distancia_km          numeric(8,2)  NOT NULL CHECK (distancia_km >= 0),
  duracion_ruta_h       numeric(6,2)  NOT NULL CHECK (duracion_ruta_h >= 0),
  duracion_operacion_h  numeric(6,2)  NOT NULL CHECK (duracion_operacion_h >= 0),
  costo_laboral         numeric(12,2) NOT NULL CHECK (costo_laboral >= 0),
  costo_vehiculo        numeric(12,2) NOT NULL CHECK (costo_vehiculo >= 0),
  costos_adicionales    numeric(12,2) NOT NULL DEFAULT 0 CHECK (costos_adicionales >= 0),
  costo_operativo       numeric(12,2) NOT NULL CHECK (costo_operativo >= 0),
  margen_pct            numeric(5,2)  NOT NULL CHECK (margen_pct >= 0),
  precio_neto           numeric(12,2) NOT NULL CHECK (precio_neto >= 0),
  porcentaje_comision   numeric(5,2)  NOT NULL CHECK (porcentaje_comision BETWEEN 0 AND 100),
  iva_pct               numeric(5,2)  NOT NULL CHECK (iva_pct BETWEEN 0 AND 100)
);
ALTER TABLE oferta_costo ENABLE ROW LEVEL SECURITY;

CREATE POLICY oferta_costo_select ON oferta_costo FOR SELECT USING (
  fn_es_administrador() OR EXISTS (
    SELECT 1 FROM oferta o
    WHERE o.id = oferta_costo.oferta_id AND o.transportista_id = (select auth.uid())
  )
);
CREATE POLICY oferta_costo_insert ON oferta_costo FOR INSERT WITH CHECK (
  EXISTS (
    SELECT 1 FROM oferta o
    WHERE o.id = oferta_costo.oferta_id AND o.transportista_id = (select auth.uid())
  )
);

ALTER TABLE oferta
  DROP COLUMN distancia_km,
  DROP COLUMN duracion_ruta_h,
  DROP COLUMN duracion_operacion_h,
  DROP COLUMN costo_laboral,
  DROP COLUMN costo_vehiculo,
  DROP COLUMN costos_adicionales,
  DROP COLUMN costo_operativo,
  DROP COLUMN margen_pct,
  DROP COLUMN precio_neto,
  DROP COLUMN porcentaje_comision,
  DROP COLUMN iva_pct;

-- 3. Alta de la oferta --------------------------------------------------------------------

CREATE FUNCTION public.fn_proteger_alta_oferta()
RETURNS trigger
LANGUAGE plpgsql
SET search_path TO 'public'
AS $function$
BEGIN
  IF current_user <> 'authenticated' OR fn_es_administrador() THEN
    RETURN NEW;
  END IF;

  NEW.estado_codigo := 'pendiente';
  NEW.creado_en := now();
  IF NOT EXISTS (SELECT 1 FROM vehiculo v WHERE v.id = NEW.vehiculo_id AND v.transportista_id = NEW.transportista_id) THEN
    RAISE EXCEPTION 'oferta: el vehículo no es del Transportista'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM solicitud s WHERE s.id = NEW.solicitud_id AND s.estado_codigo = 'publicada') THEN
    RAISE EXCEPTION 'oferta: la solicitud no está publicada'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  RETURN NEW;
END;
$function$;

CREATE TRIGGER trg_proteger_alta_oferta
  BEFORE INSERT ON oferta
  FOR EACH ROW EXECUTE FUNCTION fn_proteger_alta_oferta();

-- Corre como su dueño para ver oferta_costo aunque quien inserta no tenga SELECT sobre esa fila;
-- sólo lee, y no se puede llamar desde la API (devuelve trigger).
CREATE FUNCTION public.fn_exigir_costo_oferta()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
BEGIN
  IF EXISTS (SELECT 1 FROM oferta o WHERE o.id = NEW.id)
     AND NOT EXISTS (SELECT 1 FROM oferta_costo oc WHERE oc.oferta_id = NEW.id) THEN
    RAISE EXCEPTION 'oferta: falta el desglose de costo (oferta_costo)'
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NULL;
END;
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_exigir_costo_oferta() FROM PUBLIC, anon, authenticated;

CREATE CONSTRAINT TRIGGER trg_exigir_costo_oferta
  AFTER INSERT ON oferta
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW EXECUTE FUNCTION fn_exigir_costo_oferta();

-- 4. Oferta sin edición -------------------------------------------------------------------

CREATE FUNCTION public.fn_proteger_campos_oferta()
RETURNS trigger
LANGUAGE plpgsql
SET search_path TO 'public'
AS $function$
DECLARE
  v_sin_estado oferta;
BEGIN
  IF current_user <> 'authenticated' OR fn_es_administrador() THEN
    RETURN NEW;
  END IF;

  v_sin_estado := NEW;
  v_sin_estado.estado_codigo := OLD.estado_codigo;
  IF v_sin_estado IS DISTINCT FROM OLD THEN
    RAISE EXCEPTION 'oferta: no se puede editar; retirala y creá otra'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF NEW.estado_codigo IS DISTINCT FROM OLD.estado_codigo
     AND NOT (OLD.estado_codigo = 'pendiente' AND NEW.estado_codigo = 'retirada'
              AND OLD.transportista_id = (select auth.uid()))
     AND NOT (OLD.estado_codigo = 'pendiente' AND NEW.estado_codigo = 'no_seleccionada'
              AND EXISTS (SELECT 1 FROM solicitud s
                          WHERE s.id = OLD.solicitud_id AND s.cliente_id = (select auth.uid())
                            AND s.estado_codigo = 'cancelada')) THEN
    RAISE EXCEPTION 'oferta: cambio de estado no permitido'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  RETURN NEW;
END;
$function$;

CREATE TRIGGER trg_proteger_campos_oferta
  BEFORE UPDATE ON oferta
  FOR EACH ROW EXECUTE FUNCTION fn_proteger_campos_oferta();

-- 5. Una oferta vigente por vehículo ------------------------------------------------------

ALTER TABLE oferta DROP CONSTRAINT oferta_solicitud_id_transportista_id_vehiculo_id_key;
CREATE UNIQUE INDEX uq_oferta_vigente_por_vehiculo
  ON oferta (solicitud_id, transportista_id, vehiculo_id)
  WHERE estado_codigo <> 'retirada';

COMMIT;

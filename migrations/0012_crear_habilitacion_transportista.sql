-- 0012_crear_habilitacion_transportista.sql
-- Requisito: RF-01, RF-16 (documentación), D-19, D-33. Módulo 3 de docs/PLAN_CONSTRUCCION.md.
-- Reversible: sí (DROP de triggers, funciones y policies de Storage; DELETE del bucket si
--   está vacío; restaurar documento_transportista_insert con WITH CHECK
--   (transportista_id = (select auth.uid())) y fn_proteger_campos_transportista sin la
--   condición de pg_trigger_depth).
-- Afecta: storage.buckets (bucket nuevo), storage.objects (3 policies nuevas),
--   documento_transportista (policy de insert reemplazada, 2 triggers nuevos),
--   transportista (fn_proteger_campos_transportista modificada).
--
-- APLICADA el 2026-10-03 vía apply_migration (confirmación humana explícita), después de
-- probarla en Supabase local (make db-local).
--
-- 1. Bucket privado documentos-transportista (D-19): jpg, png y pdf de hasta 10 MB. Cada
--    Transportista sube y lee sólo bajo transportista/{su usuario_id}/; el Administrador lee
--    todo. Sin UPDATE ni DELETE: cada carga es un objeto nuevo (el historial queda en
--    documento_transportista).
-- 2. documento_transportista: el alta de un usuario no Administrador nace pendiente y sin
--    datos de revisión, y su path tiene que estar bajo su propio prefijo.
-- 3. Estado de habilitación derivado de los documentos (D-33), en la base para que ningún
--    camino (API, PostgREST) lo saltee. Se mira el último documento de cada tipo de
--    tipo_documento (dni, seguro, registro, vtv):
--      - alguno rechazado                 -> rechazado (rechazo inmediato)
--      - los cuatro aprobados             -> habilitado
--      - si no, quien estaba habilitado sigue habilitado (renovación en revisión);
--        el resto queda pendiente (incluido quien estaba rechazado y volvió a cargar).
--    trg_proteger_campos_transportista deja pasar este cambio porque viene de un trigger
--    (pg_trigger_depth() > 1); un UPDATE directo del Transportista sigue protegido.

BEGIN;

-- 1. Storage ------------------------------------------------------------------------------

INSERT INTO storage.buckets (id, name, public, file_size_limit, allowed_mime_types)
VALUES ('documentos-transportista', 'documentos-transportista', false, 10485760,
        ARRAY['image/jpeg', 'image/png', 'application/pdf']);

CREATE POLICY documentos_transportista_insert ON storage.objects FOR INSERT TO authenticated
  WITH CHECK (
    bucket_id = 'documentos-transportista'
    AND (storage.foldername(name))[1] = 'transportista'
    AND (storage.foldername(name))[2] = (select auth.uid())::text
    AND EXISTS (SELECT 1 FROM public.transportista t WHERE t.usuario_id = (select auth.uid()))
  );

CREATE POLICY documentos_transportista_select ON storage.objects FOR SELECT TO authenticated
  USING (
    bucket_id = 'documentos-transportista'
    AND (
      ((storage.foldername(name))[1] = 'transportista'
        AND (storage.foldername(name))[2] = (select auth.uid())::text)
      OR public.fn_es_administrador()
    )
  );

-- 2. Alta de documentos -------------------------------------------------------------------

CREATE FUNCTION public.fn_proteger_alta_documento()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
BEGIN
  IF (select auth.role()) IN ('anon', 'authenticated') AND NOT fn_es_administrador() THEN
    NEW.estado := 'pendiente';
    NEW.motivo_rechazo := NULL;
    NEW.revisado_por_admin_id := NULL;
    NEW.revisado_en := NULL;
    NEW.cargado_en := now();
  END IF;
  RETURN NEW;
END;
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_proteger_alta_documento() FROM PUBLIC, anon, authenticated;

CREATE TRIGGER trg_proteger_alta_documento
  BEFORE INSERT ON documento_transportista
  FOR EACH ROW EXECUTE FUNCTION fn_proteger_alta_documento();

DROP POLICY documento_transportista_insert ON documento_transportista;
CREATE POLICY documento_transportista_insert ON documento_transportista FOR INSERT TO authenticated
  WITH CHECK (
    transportista_id = (select auth.uid())
    AND url_archivo LIKE 'transportista/' || (select auth.uid())::text || '/%'
  );

-- 3. Estado de habilitación ---------------------------------------------------------------

CREATE FUNCTION public.fn_recalcular_habilitacion()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
DECLARE
  v_actual     text;
  v_nuevo      text;
  v_rechazados int;
  v_aprobados  int;
  v_requeridos int;
BEGIN
  SELECT estado_habilitacion_codigo INTO v_actual
  FROM transportista WHERE usuario_id = NEW.transportista_id;

  SELECT count(*) FILTER (WHERE ultimo.estado = 'rechazado'),
         count(*) FILTER (WHERE ultimo.estado = 'aprobado')
    INTO v_rechazados, v_aprobados
  FROM (
    SELECT DISTINCT ON (d.tipo_documento_codigo) d.estado
    FROM documento_transportista d
    WHERE d.transportista_id = NEW.transportista_id
    ORDER BY d.tipo_documento_codigo, d.cargado_en DESC, d.id DESC
  ) ultimo;

  SELECT count(*) INTO v_requeridos FROM tipo_documento;

  v_nuevo := CASE
    WHEN v_rechazados > 0 THEN 'rechazado'
    WHEN v_aprobados = v_requeridos THEN 'habilitado'
    WHEN v_actual = 'habilitado' THEN 'habilitado'
    ELSE 'pendiente'
  END;

  IF v_nuevo IS DISTINCT FROM v_actual THEN
    UPDATE transportista SET estado_habilitacion_codigo = v_nuevo
    WHERE usuario_id = NEW.transportista_id;
  END IF;
  RETURN NULL;
END;
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_recalcular_habilitacion() FROM PUBLIC, anon, authenticated;

CREATE TRIGGER trg_recalcular_habilitacion
  AFTER INSERT OR UPDATE OF estado ON documento_transportista
  FOR EACH ROW EXECUTE FUNCTION fn_recalcular_habilitacion();

CREATE OR REPLACE FUNCTION public.fn_proteger_campos_transportista()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
BEGIN
  -- pg_trigger_depth() > 1: el UPDATE viene de fn_recalcular_habilitacion, no del usuario.
  IF NOT fn_es_administrador() AND pg_trigger_depth() <= 1 THEN
    NEW.estado_habilitacion_codigo := OLD.estado_habilitacion_codigo;
    NEW.calificacion_promedio := OLD.calificacion_promedio;
    NEW.tasa_cumplimiento := OLD.tasa_cumplimiento;
  END IF;
  RETURN NEW;
END;
$function$;

COMMIT;

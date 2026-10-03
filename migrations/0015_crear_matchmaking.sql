-- 0015_crear_matchmaking.sql
-- Requisito: RN-04, RN-05 (aviso in-app). Decisiones D-21, D-22. Módulo 7 de
--   docs/PLAN_CONSTRUCCION.md.
-- Reversible: sí (DROP de las 4 funciones y del índice único).
-- Afecta: funciones nuevas fn_veto_vigente, fn_es_compatible, fn_solicitudes_compatibles y
--   fn_notificar_solicitud_compatible; índice único parcial en notificacion. Sin cambios de
--   tablas ni de policies.
--
-- D-21 en un solo lugar: una solicitud es compatible con un Transportista si
--   - está publicada y no vencida (fecha >= hoy en Argentina, D-20);
--   - el Transportista está habilitado, disponible y sin veto vigente;
--   - la zona de origen o la de destino está entre sus zonas (RN-04);
--   - al menos un vehículo activo pasa la cota rápida de peso y volumen: la carga entra en
--     20 viajes como máximo (ALGORITMO_VIAJES_EMPAQUETADO.md §4, paso 1). No se corre el
--     empaquetado completo; eso pasa al ofertar (módulo 8).
-- El listado del Transportista y el aviso al publicar usan la misma función, así que no hay
-- dos versiones de la regla.
--
-- Las funciones son SECURITY DEFINER porque necesitan datos que el usuario no ve por RLS (los
-- vetos de otros, y el Cliente no puede insertar notificaciones). Sólo se exponen las dos que
-- operan sobre el usuario que llama (auth.uid()); las de consulta interna no se pueden
-- ejecutar desde la API.

BEGIN;

-- Veto vigente: activo y, si es temporal, todavía sin vencer (D-30 lo completa).
CREATE FUNCTION public.fn_veto_vigente(p_usuario_id uuid)
RETURNS boolean
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
  SELECT EXISTS (
    SELECT 1 FROM veto v
    WHERE v.usuario_id = p_usuario_id AND v.estado = 'activo'
      AND (v.tipo = 'definitivo' OR v.fecha_fin > now())
  );
$function$;

CREATE FUNCTION public.fn_es_compatible(p_transportista_id uuid, p_solicitud_id uuid)
RETURNS boolean
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
  WITH carga AS (
    SELECT COALESCE(sum(so.peso_unitario_kg * so.cantidad), 0) AS peso,
           COALESCE(sum(so.largo_m * so.ancho_m * so.alto_m * so.cantidad), 0) AS volumen
    FROM solicitud_objeto so
    WHERE so.solicitud_id = p_solicitud_id
  )
  SELECT EXISTS (
    SELECT 1
    FROM solicitud s
    JOIN transportista t ON t.usuario_id = p_transportista_id
    CROSS JOIN carga c
    WHERE s.id = p_solicitud_id
      AND s.estado_codigo = 'publicada'
      AND s.fecha_servicio_deseada >= (now() AT TIME ZONE 'America/Argentina/Buenos_Aires')::date
      AND t.estado_habilitacion_codigo = 'habilitado'
      AND t.disponible
      AND NOT fn_veto_vigente(t.usuario_id)
      AND EXISTS (
        SELECT 1 FROM transportista_zona tz
        WHERE tz.transportista_id = t.usuario_id
          AND tz.zona_id IN (s.origen_zona_id, s.destino_zona_id)
      )
      AND EXISTS (
        SELECT 1 FROM vehiculo v
        WHERE v.transportista_id = t.usuario_id AND v.activo
          AND ceil(c.peso / v.peso_maximo_kg) <= 20
          AND ceil(c.volumen / (v.largo_util_m * v.ancho_util_m * v.alto_util_m)) <= 20
      )
  );
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_veto_vigente(uuid) FROM PUBLIC, anon, authenticated;
REVOKE EXECUTE ON FUNCTION public.fn_es_compatible(uuid, uuid) FROM PUBLIC, anon, authenticated;

-- Solicitudes compatibles con el Transportista que llama (listado, RN-04).
CREATE FUNCTION public.fn_solicitudes_compatibles()
RETURNS SETOF uuid
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
  SELECT s.id
  FROM solicitud s
  WHERE s.estado_codigo = 'publicada'
    AND fn_es_compatible((select auth.uid()), s.id);
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_solicitudes_compatibles() FROM PUBLIC, anon;
GRANT EXECUTE ON FUNCTION public.fn_solicitudes_compatibles() TO authenticated;

-- Una sola notificación por Transportista y solicitud: el job que avisa es idempotente
-- (RNF-02) y republicar crea otra solicitud, que sí se avisa de nuevo.
CREATE UNIQUE INDEX uq_notificacion_solicitud_compatible
  ON notificacion (destinatario_usuario_id, entidad_referencia_id)
  WHERE tipo_notificacion_codigo = 'solicitud_compatible';

-- Aviso in-app a los Transportistas compatibles (RN-05, D-22). Sólo la puede disparar el
-- Cliente dueño de la solicitud (o un proceso sin JWT). Devuelve cuántas notificaciones creó.
CREATE FUNCTION public.fn_notificar_solicitud_compatible(p_solicitud_id uuid)
RETURNS integer
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
DECLARE
  v_creadas integer;
BEGIN
  IF (select auth.uid()) IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM solicitud WHERE id = p_solicitud_id AND cliente_id = (select auth.uid())
  ) THEN
    RAISE EXCEPTION 'notificar: la solicitud no es del usuario' USING ERRCODE = 'insufficient_privilege';
  END IF;

  INSERT INTO notificacion (destinatario_usuario_id, tipo_notificacion_codigo, contenido,
                            entidad_referencia_tipo, entidad_referencia_id)
  SELECT t.usuario_id, 'solicitud_compatible',
         'Hay una nueva solicitud de flete en tu zona. Revisala y postulate si te interesa.',
         'solicitud', p_solicitud_id
  FROM transportista t
  WHERE fn_es_compatible(t.usuario_id, p_solicitud_id)
  ON CONFLICT (destinatario_usuario_id, entidad_referencia_id)
    WHERE tipo_notificacion_codigo = 'solicitud_compatible'
    DO NOTHING;
  GET DIAGNOSTICS v_creadas = ROW_COUNT;
  RETURN v_creadas;
END;
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_notificar_solicitud_compatible(uuid) FROM PUBLIC, anon;
GRANT EXECUTE ON FUNCTION public.fn_notificar_solicitud_compatible(uuid) TO authenticated;

COMMIT;

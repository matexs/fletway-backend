-- 0018_crear_aceptacion_oferta.sql
-- Requisito: RF-07, RN-05, RN-06. Decisiones D-23, D-26. Módulo 9 de docs/PLAN_CONSTRUCCION.md.
-- Reversible: sí, mientras no haya viajes (DROP de viaje_costo, viaje_pin y fn_aceptar_oferta;
--   volver a agregar las columnas a viaje y la policy viaje_insert original).
-- Afecta: viaje_costo (tabla nueva, 1 policy); viaje_pin (tabla nueva, 1 policy); viaje (se van 9
--   columnas de costo y pin_inicio/pin_fin; trigger reescrito; viaje_insert sólo Administrador);
--   función nueva fn_aceptar_oferta.
--
-- APLICADA el 2026-10-03 vía apply_migration (confirmación humana explícita), después de
-- probarla en Supabase local. dbFletway tenía 0 viajes.
--
-- 1. viaje_costo (D-23): el desglose del viaje sale de viaje, que el Cliente lee, a una tabla 1 a 1
--    que sólo ven el Transportista del viaje y el Administrador. Se mueven las columnas que agregó
--    0006 salvo las que el Cliente necesita: cantidad_viajes_snapshot, monto_total_snapshot (el
--    precio final), distancia_km_snapshot (no es sensible) y porcentaje_comision_snapshot (lo usa
--    el pago, RN-03). viaje está vacía en dbFletway (verificar antes de aplicar).
-- 2. viaje_pin (D-26): los PIN de inicio y fin y sus intentos. Sólo la leen el Cliente del viaje y
--    el Administrador: el Transportista nunca ve el PIN. Nadie la escribe directo: la llena
--    fn_aceptar_oferta y la actualiza fn_validar_pin (módulo 10). Se van viaje.pin_inicio y
--    viaje.pin_fin, que el Transportista podía leer.
-- 3. El viaje lo crea sólo fn_aceptar_oferta. viaje_insert dejaba al Cliente insertar un viaje con
--    cualquier precio y cualquier Transportista por PostgREST; queda sólo para el Administrador.
-- 4. fn_aceptar_oferta(oferta) (D-23), SECURITY DEFINER porque el Cliente no puede leer
--    oferta_costo ni escribir viaje_pin. Verifica que quien llama sea el Cliente de la solicitud y,
--    en la transacción del llamado: crea el viaje con sus snapshots, copia oferta_costo a
--    viaje_costo, genera los dos PIN de 4 dígitos, acepta la oferta, pasa las demás pendientes a
--    no_seleccionada y la solicitud a asignada. Los triggers de protección de 0014 y 0016 no la
--    bloquean porque corre como su dueño. Errores con SQLSTATE propio para que el backend los
--    traduzca sin mirar el texto:
--      FW001  la oferta no está pendiente;
--      FW002  la solicitud ya no está publicada o venció;
--      FW003  el Transportista ya no puede tomar el viaje (no habilitado o vetado);
--      42501  la oferta no existe o no es de una solicitud del Cliente que llama.

BEGIN;

-- 1. viaje_costo --------------------------------------------------------------------------

CREATE TABLE viaje_costo (
  viaje_id              uuid PRIMARY KEY REFERENCES viaje(id) ON DELETE CASCADE,
  duracion_ruta_h       numeric(6,2)  NOT NULL CHECK (duracion_ruta_h >= 0),
  duracion_operacion_h  numeric(6,2)  NOT NULL CHECK (duracion_operacion_h >= 0),
  costo_laboral         numeric(12,2) NOT NULL CHECK (costo_laboral >= 0),
  costo_vehiculo        numeric(12,2) NOT NULL CHECK (costo_vehiculo >= 0),
  costos_adicionales    numeric(12,2) NOT NULL DEFAULT 0 CHECK (costos_adicionales >= 0),
  costo_operativo       numeric(12,2) NOT NULL CHECK (costo_operativo >= 0),
  margen_pct            numeric(5,2)  NOT NULL CHECK (margen_pct >= 0),
  precio_neto           numeric(12,2) NOT NULL CHECK (precio_neto >= 0),
  iva_pct               numeric(5,2)  NOT NULL CHECK (iva_pct BETWEEN 0 AND 100)
);
ALTER TABLE viaje_costo ENABLE ROW LEVEL SECURITY;

CREATE POLICY viaje_costo_select ON viaje_costo FOR SELECT USING (
  fn_es_administrador() OR EXISTS (
    SELECT 1 FROM viaje v
    WHERE v.id = viaje_costo.viaje_id AND v.transportista_id = (select auth.uid())
  )
);

-- 2. viaje_pin ----------------------------------------------------------------------------

CREATE TABLE viaje_pin (
  viaje_id         uuid PRIMARY KEY REFERENCES viaje(id) ON DELETE CASCADE,
  pin_inicio       text    NOT NULL CHECK (pin_inicio ~ '^[0-9]{4}$'),
  pin_fin          text    NOT NULL CHECK (pin_fin ~ '^[0-9]{4}$'),
  intentos_inicio  integer NOT NULL DEFAULT 0 CHECK (intentos_inicio >= 0),
  intentos_fin     integer NOT NULL DEFAULT 0 CHECK (intentos_fin >= 0),
  CONSTRAINT chk_viaje_pin_distintos CHECK (pin_inicio <> pin_fin)
);
ALTER TABLE viaje_pin ENABLE ROW LEVEL SECURITY;

CREATE POLICY viaje_pin_select ON viaje_pin FOR SELECT USING (
  fn_es_administrador() OR EXISTS (
    SELECT 1 FROM viaje v
    WHERE v.id = viaje_pin.viaje_id AND v.cliente_id = (select auth.uid())
  )
);

-- viaje: el trigger de protección nombra las columnas que se van; se reescribe antes del DROP.
CREATE OR REPLACE FUNCTION public.fn_proteger_campos_viaje()
RETURNS trigger
LANGUAGE plpgsql
SET search_path TO 'public'
AS $function$
BEGIN
  IF NOT fn_es_administrador() THEN
    NEW.oferta_id := OLD.oferta_id;
    NEW.solicitud_id := OLD.solicitud_id;
    NEW.cliente_id := OLD.cliente_id;
    NEW.transportista_id := OLD.transportista_id;
    NEW.monto_total_snapshot := OLD.monto_total_snapshot;
    NEW.porcentaje_comision_snapshot := OLD.porcentaje_comision_snapshot;
    NEW.cantidad_viajes_snapshot := OLD.cantidad_viajes_snapshot;
    NEW.cantidad_ayudantes := OLD.cantidad_ayudantes;
    NEW.distancia_km_snapshot := OLD.distancia_km_snapshot;
    NEW.origen_direccion_snapshot := OLD.origen_direccion_snapshot;
    NEW.destino_direccion_snapshot := OLD.destino_direccion_snapshot;
    NEW.origen_lat_snapshot := OLD.origen_lat_snapshot;
    NEW.origen_lng_snapshot := OLD.origen_lng_snapshot;
    NEW.destino_lat_snapshot := OLD.destino_lat_snapshot;
    NEW.destino_lng_snapshot := OLD.destino_lng_snapshot;
    NEW.transportista_nombre_snapshot := OLD.transportista_nombre_snapshot;
    NEW.vehiculo_patente_snapshot := OLD.vehiculo_patente_snapshot;
    NEW.vehiculo_marca_modelo_snapshot := OLD.vehiculo_marca_modelo_snapshot;
    NEW.creado_en := OLD.creado_en;
    -- DEPRECATED: se quitan de acá en la misma migración que haga su DROP
    NEW.tarifa_base_snapshot := OLD.tarifa_base_snapshot;
    NEW.valor_por_km_snapshot := OLD.valor_por_km_snapshot;
    NEW.valor_por_m3_snapshot := OLD.valor_por_m3_snapshot;
    NEW.valor_por_hora_snapshot := OLD.valor_por_hora_snapshot;
    NEW.recargo_escalera_snapshot := OLD.recargo_escalera_snapshot;
    NEW.recargo_ayudante_snapshot := OLD.recargo_ayudante_snapshot;
  END IF;
  RETURN NEW;
END;
$function$;

ALTER TABLE viaje
  DROP COLUMN duracion_ruta_h_snapshot,
  DROP COLUMN duracion_operacion_h_snapshot,
  DROP COLUMN costo_laboral_snapshot,
  DROP COLUMN costo_vehiculo_snapshot,
  DROP COLUMN costos_adicionales_snapshot,
  DROP COLUMN costo_operativo_snapshot,
  DROP COLUMN margen_pct_snapshot,
  DROP COLUMN precio_neto_snapshot,
  DROP COLUMN iva_pct_snapshot,
  DROP COLUMN pin_inicio,
  DROP COLUMN pin_fin;

-- 3. El viaje sólo nace de la aceptación ---------------------------------------------------

DROP POLICY viaje_insert ON viaje;
CREATE POLICY viaje_insert ON viaje FOR INSERT WITH CHECK (fn_es_administrador());

-- 4. fn_aceptar_oferta ---------------------------------------------------------------------

-- PIN de 4 dígitos con el generador criptográfico de pgcrypto (no random()).
CREATE FUNCTION public.fn_generar_pin()
RETURNS text
LANGUAGE sql
VOLATILE
SET search_path TO 'public'
AS $function$
  SELECT lpad(((('x' || encode(extensions.gen_random_bytes(4), 'hex'))::bit(32)::bigint
                & 2147483647) % 10000)::text, 4, '0');
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_generar_pin() FROM PUBLIC, anon, authenticated;

CREATE FUNCTION public.fn_aceptar_oferta(p_oferta_id uuid)
RETURNS uuid
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
DECLARE
  v_oferta    oferta;
  v_solicitud solicitud;
  v_viaje_id  uuid;
  v_pin_ini   text;
  v_pin_fin   text;
BEGIN
  SELECT * INTO v_oferta FROM oferta WHERE id = p_oferta_id FOR UPDATE;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'aceptar oferta: no existe' USING ERRCODE = 'insufficient_privilege';
  END IF;
  SELECT * INTO v_solicitud FROM solicitud WHERE id = v_oferta.solicitud_id FOR UPDATE;
  IF v_solicitud.cliente_id IS DISTINCT FROM (select auth.uid()) THEN
    RAISE EXCEPTION 'aceptar oferta: la solicitud no es del usuario' USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF v_oferta.estado_codigo <> 'pendiente' THEN
    RAISE EXCEPTION 'aceptar oferta: la oferta no está pendiente' USING ERRCODE = 'FW001';
  END IF;
  IF v_solicitud.estado_codigo <> 'publicada'
     OR v_solicitud.fecha_servicio_deseada < (now() AT TIME ZONE 'America/Argentina/Buenos_Aires')::date THEN
    RAISE EXCEPTION 'aceptar oferta: la solicitud ya no está publicada' USING ERRCODE = 'FW002';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM transportista t
                 WHERE t.usuario_id = v_oferta.transportista_id
                   AND t.estado_habilitacion_codigo = 'habilitado')
     OR fn_veto_vigente(v_oferta.transportista_id) THEN
    RAISE EXCEPTION 'aceptar oferta: el Transportista no puede tomar el viaje' USING ERRCODE = 'FW003';
  END IF;

  INSERT INTO viaje (oferta_id, solicitud_id, cliente_id, transportista_id, estado_codigo,
      origen_direccion_snapshot, destino_direccion_snapshot,
      origen_lat_snapshot, origen_lng_snapshot, destino_lat_snapshot, destino_lng_snapshot,
      distancia_km_snapshot, transportista_nombre_snapshot, vehiculo_patente_snapshot,
      vehiculo_marca_modelo_snapshot, monto_total_snapshot, porcentaje_comision_snapshot,
      cantidad_ayudantes, cantidad_viajes_snapshot)
  SELECT v_oferta.id, v_solicitud.id, v_solicitud.cliente_id, v_oferta.transportista_id, 'confirmado',
         v_solicitud.origen_direccion, v_solicitud.destino_direccion,
         v_solicitud.origen_lat, v_solicitud.origen_lng, v_solicitud.destino_lat, v_solicitud.destino_lng,
         oc.distancia_km, u.nombre_completo, ve.patente,
         NULLIF(btrim(concat_ws(' ', ve.marca, ve.modelo)), ''),
         v_oferta.precio_calculado, oc.porcentaje_comision,
         v_oferta.cantidad_ayudantes, v_oferta.cantidad_viajes
  FROM oferta_costo oc, usuario u, vehiculo ve
  WHERE oc.oferta_id = v_oferta.id AND u.id = v_oferta.transportista_id AND ve.id = v_oferta.vehiculo_id
  RETURNING id INTO v_viaje_id;

  INSERT INTO viaje_costo (viaje_id, duracion_ruta_h, duracion_operacion_h, costo_laboral, costo_vehiculo,
      costos_adicionales, costo_operativo, margen_pct, precio_neto, iva_pct)
  SELECT v_viaje_id, duracion_ruta_h, duracion_operacion_h, costo_laboral, costo_vehiculo,
         costos_adicionales, costo_operativo, margen_pct, precio_neto, iva_pct
  FROM oferta_costo WHERE oferta_id = v_oferta.id;

  v_pin_ini := fn_generar_pin();
  LOOP
    v_pin_fin := fn_generar_pin();
    EXIT WHEN v_pin_fin <> v_pin_ini;
  END LOOP;
  INSERT INTO viaje_pin (viaje_id, pin_inicio, pin_fin) VALUES (v_viaje_id, v_pin_ini, v_pin_fin);

  UPDATE oferta SET estado_codigo = 'aceptada' WHERE id = v_oferta.id;
  UPDATE oferta SET estado_codigo = 'no_seleccionada'
  WHERE solicitud_id = v_solicitud.id AND id <> v_oferta.id AND estado_codigo = 'pendiente';
  UPDATE solicitud SET estado_codigo = 'asignada' WHERE id = v_solicitud.id;

  RETURN v_viaje_id;
END;
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_aceptar_oferta(uuid) FROM PUBLIC, anon;
GRANT EXECUTE ON FUNCTION public.fn_aceptar_oferta(uuid) TO authenticated;

COMMIT;

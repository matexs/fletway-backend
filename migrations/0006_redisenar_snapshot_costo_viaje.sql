-- 0006_redisenar_snapshot_costo_viaje.sql
-- Requisito: RNF-03 (snapshot histórico), RN-01, RN-03
-- Reversible: sí (DROP COLUMN de las nuevas; SET NOT NULL sobre las deprecadas; restaurar la versión
--             anterior de fn_proteger_campos_viaje, transcripta al final de este archivo)
-- Afecta: viaje (columnas nuevas + deprecación de 6 snapshots de tarifa plana),
--         función fn_proteger_campos_viaje (CREATE OR REPLACE; el trigger trg_proteger_campos_viaje no cambia)
--
-- APLICADA el 2026-09-24 vía apply_migration (confirmación humana explícita).
--    Diseño: docs/ALGORITMO_COTIZACION.md y docs/ALGORITMO_VIAJES_EMPAQUETADO.md.
--    Depende de 0005 (los valores se copian desde oferta al confirmar el viaje).
--
-- Se REUTILIZAN sin cambios: distancia_km_snapshot, monto_total_snapshot
-- (= oferta.precio_calculado) y porcentaje_comision_snapshot.

BEGIN;

-- Tabla vacía hoy → NOT NULL sin default es seguro.
ALTER TABLE viaje
  ADD COLUMN cantidad_viajes_snapshot       integer       NOT NULL CHECK (cantidad_viajes_snapshot > 0),
  ADD COLUMN duracion_ruta_h_snapshot       numeric(6,2)  NOT NULL,
  ADD COLUMN duracion_operacion_h_snapshot  numeric(6,2)  NOT NULL,
  ADD COLUMN costo_laboral_snapshot         numeric(12,2) NOT NULL,
  ADD COLUMN costo_vehiculo_snapshot        numeric(12,2) NOT NULL,
  ADD COLUMN costos_adicionales_snapshot    numeric(12,2) NOT NULL DEFAULT 0,
  ADD COLUMN costo_operativo_snapshot       numeric(12,2) NOT NULL,
  ADD COLUMN margen_pct_snapshot            numeric(5,2)  NOT NULL,
  ADD COLUMN precio_neto_snapshot           numeric(12,2) NOT NULL,
  ADD COLUMN iva_pct_snapshot               numeric(5,2)  NOT NULL;

-- ─── Deprecación del modelo de tarifa plana (DROP en migración posterior) ──
ALTER TABLE viaje
  ALTER COLUMN tarifa_base_snapshot      DROP NOT NULL,
  ALTER COLUMN valor_por_km_snapshot     DROP NOT NULL,
  ALTER COLUMN valor_por_m3_snapshot     DROP NOT NULL,
  ALTER COLUMN valor_por_hora_snapshot   DROP NOT NULL,
  ALTER COLUMN recargo_escalera_snapshot DROP NOT NULL,
  ALTER COLUMN recargo_ayudante_snapshot DROP NOT NULL;

COMMENT ON COLUMN viaje.tarifa_base_snapshot      IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';
COMMENT ON COLUMN viaje.valor_por_km_snapshot     IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';
COMMENT ON COLUMN viaje.valor_por_m3_snapshot     IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';
COMMENT ON COLUMN viaje.valor_por_hora_snapshot   IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';
COMMENT ON COLUMN viaje.recargo_escalera_snapshot IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';
COMMENT ON COLUMN viaje.recargo_ayudante_snapshot IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';
COMMENT ON COLUMN viaje.monto_total_snapshot      IS 'Precio final pagado por el Cliente (copia de oferta.precio_calculado).';

-- ─── Trigger de protección: agregar los snapshots nuevos ───────────────────
-- Se mantienen las columnas deprecadas hasta su DROP (si no, un DROP posterior
-- rompería la función; el DROP debe venir junto con otro CREATE OR REPLACE).
CREATE OR REPLACE FUNCTION public.fn_proteger_campos_viaje()
 RETURNS trigger
 LANGUAGE plpgsql
 SECURITY DEFINER
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
    -- nuevos (0006)
    NEW.cantidad_viajes_snapshot := OLD.cantidad_viajes_snapshot;
    NEW.duracion_ruta_h_snapshot := OLD.duracion_ruta_h_snapshot;
    NEW.duracion_operacion_h_snapshot := OLD.duracion_operacion_h_snapshot;
    NEW.costo_laboral_snapshot := OLD.costo_laboral_snapshot;
    NEW.costo_vehiculo_snapshot := OLD.costo_vehiculo_snapshot;
    NEW.costos_adicionales_snapshot := OLD.costos_adicionales_snapshot;
    NEW.costo_operativo_snapshot := OLD.costo_operativo_snapshot;
    NEW.margen_pct_snapshot := OLD.margen_pct_snapshot;
    NEW.precio_neto_snapshot := OLD.precio_neto_snapshot;
    NEW.iva_pct_snapshot := OLD.iva_pct_snapshot;
    NEW.distancia_km_snapshot := OLD.distancia_km_snapshot;
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

COMMIT;

-- ─── Versión ANTERIOR de fn_proteger_campos_viaje (para revertir) ─────────
-- Transcripta de pg_get_functiondef antes de aplicar esta migración.
--
-- CREATE OR REPLACE FUNCTION public.fn_proteger_campos_viaje()
--  RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path TO 'public'
-- AS $function$
-- BEGIN
--   IF NOT fn_es_administrador() THEN
--     NEW.oferta_id := OLD.oferta_id;
--     NEW.solicitud_id := OLD.solicitud_id;
--     NEW.cliente_id := OLD.cliente_id;
--     NEW.transportista_id := OLD.transportista_id;
--     NEW.monto_total_snapshot := OLD.monto_total_snapshot;
--     NEW.tarifa_base_snapshot := OLD.tarifa_base_snapshot;
--     NEW.valor_por_km_snapshot := OLD.valor_por_km_snapshot;
--     NEW.valor_por_m3_snapshot := OLD.valor_por_m3_snapshot;
--     NEW.valor_por_hora_snapshot := OLD.valor_por_hora_snapshot;
--     NEW.recargo_escalera_snapshot := OLD.recargo_escalera_snapshot;
--     NEW.recargo_ayudante_snapshot := OLD.recargo_ayudante_snapshot;
--     NEW.porcentaje_comision_snapshot := OLD.porcentaje_comision_snapshot;
--   END IF;
--   RETURN NEW;
-- END;
-- $function$;

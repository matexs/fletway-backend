-- 0003_add_acceso_origen_destino_solicitud.sql
-- Requisito: RN-01 (escalera/altura), RF-06; decisión de producto "sin cotización estimada"
-- Reversible: sí (DROP COLUMN de las nuevas; SET NOT NULL sobre requiere_escalera)
-- Afecta: solicitud (columnas nuevas + deprecación de requiere_escalera, pisos_escalera, cotizacion_estimada_monto). Sin cambios de RLS.
--
-- ✅ APLICADA el 2026-09-24 vía apply_migration (confirmación humana explícita).
--    Diseño: docs/ALGORITMO_COTIZACION.md y docs/ALGORITMO_VIAJES_EMPAQUETADO.md.

BEGIN;

-- Tabla vacía hoy. Defaults = "planta baja, sin ascensor, vehículo en la puerta".
ALTER TABLE solicitud
  ADD COLUMN pisos_origen                 integer      NOT NULL DEFAULT 0 CHECK (pisos_origen  >= 0),
  ADD COLUMN ascensor_utilizable_origen   boolean      NOT NULL DEFAULT false,
  ADD COLUMN distancia_vehiculo_origen_m  numeric(6,1) NOT NULL DEFAULT 0 CHECK (distancia_vehiculo_origen_m  >= 0),
  ADD COLUMN pisos_destino                integer      NOT NULL DEFAULT 0 CHECK (pisos_destino >= 0),
  ADD COLUMN ascensor_utilizable_destino  boolean      NOT NULL DEFAULT false,
  ADD COLUMN distancia_vehiculo_destino_m numeric(6,1) NOT NULL DEFAULT 0 CHECK (distancia_vehiculo_destino_m >= 0);

COMMENT ON COLUMN solicitud.distancia_vehiculo_origen_m IS
  'Distancia a pie (m) entre el vehículo estacionado y la puerta en origen. NO es la distancia del Transportista al origen (no se calcula ni se cobra).';

-- ─── Deprecaciones (DROP en migración posterior) ───────────────────────────
ALTER TABLE solicitud ALTER COLUMN requiere_escalera DROP NOT NULL;
ALTER TABLE solicitud ALTER COLUMN requiere_escalera DROP DEFAULT;
COMMENT ON COLUMN solicitud.requiere_escalera IS
  'DEPRECATED: reemplazada por pisos_origen/pisos_destino + ascensor_utilizable_*.';
COMMENT ON COLUMN solicitud.pisos_escalera IS
  'DEPRECATED: reemplazada por pisos_origen/pisos_destino.';
COMMENT ON COLUMN solicitud.cotizacion_estimada_monto IS
  'DEPRECATED: no hay cotización estimada al publicar (decisión de producto). El único precio es oferta.precio_calculado.';

COMMIT;

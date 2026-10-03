-- 0013_reemplazar_tipo_vehiculo.sql
-- Requisito: RF-18, RN-02. Decisiones A-1, D-32. Módulo 4 de docs/PLAN_CONSTRUCCION.md (§2.3).
-- Reversible: sí mientras no haya vehículos (DELETE de los 6 tipos, reinsertar los 4 originales
--   desde migrations/baseline/02_datos_catalogos_2026-10-02.sql y DROP de las 3 columnas).
-- Afecta: tipo_vehiculo (3 columnas nuevas, datos reemplazados, comentario de deprecación).
--   Sin cambios de RLS.
--
-- APLICADA el 2026-10-03 vía apply_migration (confirmación humana explícita), después de
-- probarla en Supabase local (make db-local).
--
-- Las medidas estándar son de referencia: la app las propone al registrar un vehículo y el
-- Transportista las corrige. El cálculo de viajes y el matchmaking usan siempre las del
-- vehículo real (vehiculo.largo_util_m, ancho_util_m, alto_util_m).
--
-- El DELETE es seguro porque vehiculo está vacía (verificado en dbFletway el 2026-10-03); si
-- hubiera vehículos, la FK vehiculo_tipo_vehiculo_id_fkey hace fallar la migración entera.

BEGIN;

ALTER TABLE tipo_vehiculo
  ADD COLUMN largo_estandar_m numeric(5,2) CHECK (largo_estandar_m > 0),
  ADD COLUMN ancho_estandar_m numeric(5,2) CHECK (ancho_estandar_m > 0),
  ADD COLUMN alto_estandar_m  numeric(5,2) CHECK (alto_estandar_m > 0);

DELETE FROM tipo_vehiculo;

INSERT INTO tipo_vehiculo (nombre, largo_estandar_m, ancho_estandar_m, alto_estandar_m,
                           volumen_estandar_m3, peso_maximo_estandar_kg) VALUES
  ('Utilitario',     1.60, 1.20, 1.05,  2.02,  500),
  ('Furgón chico',   2.50, 1.50, 1.20,  4.50,  700),
  ('Furgón grande',  3.00, 1.70, 1.70,  8.67, 1200),
  ('Camión chico',   4.00, 2.00, 1.90, 15.20, 2500),
  ('Camión mediano', 5.00, 2.20, 2.20, 24.20, 3500),
  ('Camión grande',  7.00, 2.40, 2.40, 40.32, 7000);

ALTER TABLE tipo_vehiculo
  ALTER COLUMN largo_estandar_m SET NOT NULL,
  ALTER COLUMN ancho_estandar_m SET NOT NULL,
  ALTER COLUMN alto_estandar_m  SET NOT NULL;

COMMENT ON COLUMN tipo_vehiculo.volumen_estandar_m3 IS
  'DEPRECATED: derivable de largo_estandar_m*ancho_estandar_m*alto_estandar_m. Se elimina en una migración posterior.';

COMMIT;

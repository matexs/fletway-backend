-- 0017_crear_config_costo_vehiculo.sql
-- Requisito: RN-01, RF-17. Decisión D-34. Módulo 8 de docs/PLAN_CONSTRUCCION.md.
-- Reversible: sí (DROP TABLE config_costo_vehiculo; quitar el COMMENT de vehiculo_costo).
-- Afecta: config_costo_vehiculo (tabla nueva, 4 policies, seed de 6 filas); vehiculo_costo (sólo
--   COMMENT de deprecación).
--
-- APLICADA el 2026-10-03 vía apply_migration (confirmación humana explícita), después de
-- probarla en Supabase local. vehiculo_costo estaba vacía en dbFletway.
--
-- D-34: los costos del vehículo que entran en el precio (combustible, neumáticos, mantenimiento,
-- depreciación, seguro y patente) los define la plataforma por tipo de vehículo, no cada
-- Transportista. Si los cargara el Transportista, podría inflarlos para subir su precio; con un
-- valor de referencia por tipo, dos ofertas con el mismo tipo de vehículo sólo difieren por la
-- cantidad de viajes y de ayudantes. Las medidas y la carga útil siguen siendo las del vehículo
-- real, porque el cálculo de viajes las necesita.
--
-- Versionada como las demás config_* (una fila vigente por tipo). Lectura para autenticados
-- (el cálculo corre con el rol del Transportista, D-02); escritura sólo Administrador.
--
-- La seed son VALORES DE TRABAJO estimados (octubre de 2026, en pesos), igual que las demás
-- config_* (D-24): reemplazarlos por cifras definitivas es cerrar la fila vigente e insertar otra.
--
-- vehiculo_costo queda DEPRECATED: el backend deja de leerla y escribirla. Está vacía en
-- dbFletway; el DROP va en una migración posterior junto con lo demás deprecado.

BEGIN;

CREATE TABLE config_costo_vehiculo (
  id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tipo_vehiculo_id        uuid NOT NULL REFERENCES tipo_vehiculo(id),
  combustible_precio_l    numeric(10,2) NOT NULL CHECK (combustible_precio_l >= 0),
  rendimiento_km_l        numeric(6,2)  NOT NULL CHECK (rendimiento_km_l > 0),
  cantidad_neumaticos     integer       NOT NULL CHECK (cantidad_neumaticos > 0),
  costo_neumatico         numeric(12,2) NOT NULL CHECK (costo_neumatico >= 0),
  vida_neumatico_km       numeric(10,0) NOT NULL CHECK (vida_neumatico_km > 0),
  costo_mantenimiento_km  numeric(10,2) NOT NULL CHECK (costo_mantenimiento_km >= 0),
  valor_compra            numeric(14,2) NOT NULL CHECK (valor_compra >= 0),
  valor_residual          numeric(14,2) NOT NULL CHECK (valor_residual >= 0),
  vida_util_km            numeric(10,0) NOT NULL CHECK (vida_util_km > 0),
  seguro_mensual          numeric(12,2) NOT NULL CHECK (seguro_mensual >= 0),
  patente_mensual         numeric(12,2) NOT NULL CHECK (patente_mensual >= 0),
  vigente_desde           timestamptz   NOT NULL DEFAULT now(),
  vigente_hasta           timestamptz,
  CONSTRAINT chk_config_costo_vehiculo_residual CHECK (valor_residual <= valor_compra),
  CONSTRAINT chk_vigencia_costo_vehiculo CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde)
);
ALTER TABLE config_costo_vehiculo ENABLE ROW LEVEL SECURITY;
CREATE UNIQUE INDEX idx_config_costo_vehiculo_vigente
  ON config_costo_vehiculo (tipo_vehiculo_id) WHERE vigente_hasta IS NULL;

CREATE POLICY config_costo_vehiculo_select ON config_costo_vehiculo FOR SELECT USING (true);
CREATE POLICY config_costo_vehiculo_insert ON config_costo_vehiculo FOR INSERT WITH CHECK (fn_es_administrador());
CREATE POLICY config_costo_vehiculo_update ON config_costo_vehiculo FOR UPDATE USING (fn_es_administrador()) WITH CHECK (fn_es_administrador());
CREATE POLICY config_costo_vehiculo_delete ON config_costo_vehiculo FOR DELETE USING (fn_es_administrador());

-- Valores de trabajo. Combustible diésel a $1.300/l para todos; el resto crece con el tamaño.
INSERT INTO config_costo_vehiculo (tipo_vehiculo_id, combustible_precio_l, rendimiento_km_l,
    cantidad_neumaticos, costo_neumatico, vida_neumatico_km, costo_mantenimiento_km, valor_compra,
    valor_residual, vida_util_km, seguro_mensual, patente_mensual)
SELECT t.id, v.combustible, v.rendimiento, v.neumaticos, v.costo_neumatico, v.vida_neumatico,
       v.mantenimiento, v.compra, v.residual, v.vida_util, v.seguro, v.patente
FROM (VALUES
  ('Utilitario',     1300, 11.0,  4, 120000, 50000,  45,  22000000,  7000000,  300000,  70000,  25000),
  ('Furgón chico',   1300, 10.0,  4, 140000, 50000,  55,  30000000,  9000000,  350000,  85000,  30000),
  ('Furgón grande',  1300,  8.0,  6, 180000, 60000,  75,  50000000, 15000000,  450000, 120000,  45000),
  ('Camión chico',   1300,  6.0,  6, 250000, 70000, 100,  70000000, 20000000,  600000, 160000,  60000),
  ('Camión mediano', 1300,  4.5,  6, 350000, 80000, 140, 110000000, 30000000,  800000, 220000,  90000),
  ('Camión grande',  1300,  3.5, 10, 450000, 90000, 190, 160000000, 45000000, 1000000, 300000, 130000)
) AS v(nombre, combustible, rendimiento, neumaticos, costo_neumatico, vida_neumatico, mantenimiento,
       compra, residual, vida_util, seguro, patente)
JOIN tipo_vehiculo t ON t.nombre = v.nombre;

DO $$
BEGIN
  IF (SELECT count(*) FROM config_costo_vehiculo) <> (SELECT count(*) FROM tipo_vehiculo) THEN
    RAISE EXCEPTION 'config_costo_vehiculo: falta el costo de referencia de algún tipo de vehículo';
  END IF;
END;
$$;

COMMENT ON TABLE vehiculo_costo IS
  'DEPRECATED (0017, D-34): los costos del vehículo los define la plataforma por tipo en config_costo_vehiculo. El backend no la lee ni la escribe.';

COMMIT;

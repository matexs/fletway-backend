-- 0002_add_dimensiones_vehiculo_create_vehiculo_costo.sql
-- Requisito: RN-01 (costo por vehículo real), RN-02 (capacidad por dimensiones)
-- Reversible: sí (DROP TABLE vehiculo_costo; DROP COLUMN de dimensiones; SET NOT NULL sobre volumen_carga_m3)
-- Afecta: vehiculo (columnas nuevas + deprecación de volumen_carga_m3), vehiculo_costo (tabla nueva + 3 policies)
--
-- APLICADA el 2026-09-24 vía apply_migration (confirmación humana explícita).
--    Diseño: docs/ALGORITMO_COTIZACION.md y docs/ALGORITMO_VIAJES_EMPAQUETADO.md.
--
-- Decisiones confirmadas por el humano (2026-09-24):
--   * Los campos de costo van en una tabla 1:1 separada (vehiculo_costo), porque
--     vehiculo_select es USING (true) y expondría datos financieros del Transportista.
--   * vehiculo.peso_maximo_kg YA es carga útil → no se agrega peso_vehiculo_kg.

BEGIN;

-- ─── vehiculo (vacía hoy, lectura pública) ─────────────────────────────────
ALTER TABLE vehiculo
  ADD COLUMN largo_util_m numeric(5,2) NOT NULL CHECK (largo_util_m > 0),
  ADD COLUMN ancho_util_m numeric(5,2) NOT NULL CHECK (ancho_util_m > 0),
  ADD COLUMN alto_util_m  numeric(5,2) NOT NULL CHECK (alto_util_m  > 0);

COMMENT ON COLUMN vehiculo.peso_maximo_kg IS
  'Carga útil máxima en kg (NO peso bruto total). Se usa directo como capacidad de peso en el empaquetado.';

ALTER TABLE vehiculo ALTER COLUMN volumen_carga_m3 DROP NOT NULL;
COMMENT ON COLUMN vehiculo.volumen_carga_m3 IS
  'DEPRECATED: derivable de largo_util_m*ancho_util_m*alto_util_m. Se elimina en una migración posterior.';

-- ─── vehiculo_costo (nueva, 1:1 con vehiculo, privada) ─────────────────────
CREATE TABLE vehiculo_costo (
  vehiculo_id            uuid PRIMARY KEY REFERENCES vehiculo(id) ON DELETE CASCADE,
  combustible_precio_l   numeric(10,2) NOT NULL CHECK (combustible_precio_l >= 0),
  rendimiento_km_l       numeric(6,2)  NOT NULL CHECK (rendimiento_km_l > 0),
  cantidad_neumaticos    integer       NOT NULL CHECK (cantidad_neumaticos > 0),
  costo_neumatico        numeric(12,2) NOT NULL CHECK (costo_neumatico >= 0),
  vida_neumatico_km      numeric(10,0) NOT NULL CHECK (vida_neumatico_km > 0),
  costo_mantenimiento_km numeric(10,2) NOT NULL CHECK (costo_mantenimiento_km >= 0),
  valor_compra           numeric(14,2) NOT NULL CHECK (valor_compra >= 0),
  valor_residual         numeric(14,2) NOT NULL CHECK (valor_residual >= 0),
  vida_util_km           numeric(10,0) NOT NULL CHECK (vida_util_km > 0),
  seguro_mensual         numeric(12,2) NOT NULL CHECK (seguro_mensual >= 0),
  patente_mensual        numeric(12,2) NOT NULL CHECK (patente_mensual >= 0),
  actualizado_en         timestamptz   NOT NULL DEFAULT now(),
  CONSTRAINT chk_vehiculo_costo_residual CHECK (valor_residual <= valor_compra)
);

ALTER TABLE vehiculo_costo ENABLE ROW LEVEL SECURITY;

COMMENT ON TABLE vehiculo_costo IS
  'Variables de costo del vehículo real para RN-01. Privada: solo el Transportista dueño y el Administrador.';
COMMENT ON COLUMN vehiculo_costo.combustible_precio_l IS
  'Por vehículo (diesel/nafta/GNC); el Transportista lo actualiza manualmente.';

-- Mismo esquema que vehiculo: select/insert/update (sin DELETE; se borra por CASCADE).
-- El filtro de ownership usa idx_vehiculo_transportista (ya existe) vía la PK de vehiculo.
CREATE POLICY vehiculo_costo_select ON vehiculo_costo FOR SELECT
  USING (
    fn_es_administrador()
    OR EXISTS (SELECT 1 FROM vehiculo v
               WHERE v.id = vehiculo_costo.vehiculo_id
                 AND v.transportista_id = (select auth.uid()))
  );

CREATE POLICY vehiculo_costo_insert ON vehiculo_costo FOR INSERT
  WITH CHECK (
    fn_es_administrador()
    OR EXISTS (SELECT 1 FROM vehiculo v
               WHERE v.id = vehiculo_costo.vehiculo_id
                 AND v.transportista_id = (select auth.uid()))
  );

CREATE POLICY vehiculo_costo_update ON vehiculo_costo FOR UPDATE
  USING (
    fn_es_administrador()
    OR EXISTS (SELECT 1 FROM vehiculo v
               WHERE v.id = vehiculo_costo.vehiculo_id
                 AND v.transportista_id = (select auth.uid()))
  )
  WITH CHECK (
    fn_es_administrador()
    OR EXISTS (SELECT 1 FROM vehiculo v
               WHERE v.id = vehiculo_costo.vehiculo_id
                 AND v.transportista_id = (select auth.uid()))
  );

COMMIT;

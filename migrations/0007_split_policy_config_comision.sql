-- 0007_split_policy_config_comision.sql
-- Requisito: RN-01, RN-03 (el cálculo de la oferta necesita leer la comisión vigente)
-- Reversible: sí (DROP de las 4 policies nuevas; recrear config_comision_admin FOR ALL USING/WITH CHECK fn_es_administrador())
-- Afecta: policies de config_comision
--
-- ✅ APLICADA el 2026-09-24 vía apply_migration (confirmación humana explícita).
--    Diseño: docs/ALGORITMO_COTIZACION.md y docs/ALGORITMO_VIAJES_EMPAQUETADO.md.
--
-- Problema: hoy config_comision tiene una única policy config_comision_admin
-- FOR ALL con fn_es_administrador(). Con RLS pass-through (D-02), la creación de
-- la oferta corre con el rol del Transportista → no puede leer el % de comisión
-- vigente y el precio final no se puede calcular. Se abre la LECTURA (el %
-- de comisión no es un dato sensible) y se separa la escritura (patrón P3).
-- config_tarifa tiene el mismo problema, pero queda DEPRECATED (0004): no se toca.

BEGIN;

DROP POLICY config_comision_admin ON config_comision;

CREATE POLICY config_comision_select ON config_comision FOR SELECT USING (true);
CREATE POLICY config_comision_insert ON config_comision FOR INSERT WITH CHECK (fn_es_administrador());
CREATE POLICY config_comision_update ON config_comision FOR UPDATE USING (fn_es_administrador()) WITH CHECK (fn_es_administrador());
CREATE POLICY config_comision_delete ON config_comision FOR DELETE USING (fn_es_administrador());

COMMIT;

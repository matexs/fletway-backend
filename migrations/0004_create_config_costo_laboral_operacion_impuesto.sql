-- 0004_create_config_costo_laboral_operacion_impuesto.sql
-- Requisito: RN-01 (parámetros de costo versionados)
-- Reversible: sí (DROP TABLE de las 3 tablas nuevas; quitar COMMENT de config_tarifa)
-- Afecta: config_costo_laboral, config_operacion, config_impuesto (tablas nuevas + 4 policies c/u + seed ilustrativa);
--         config_tarifa (solo COMMENT de deprecación)
--
-- APLICADA el 2026-09-24 vía apply_migration (confirmación humana explícita).
--    Diseño: docs/ALGORITMO_COTIZACION.md y docs/ALGORITMO_VIAJES_EMPAQUETADO.md.
--
-- Decisión confirmada por el humano (2026-09-24): tablas nuevas versionadas;
-- config_tarifa queda DEPRECATED, sin DROP en esta tanda.
--
-- Convención de porcentajes: 0–100 con numeric(5,2), igual que config_comision.porcentaje.
--
-- ATENCIÓN: margen_pct NO está en ninguna de estas tablas a propósito: su ubicación
--    (config de plataforma vs. por Transportista) es una DECISIÓN ABIERTA.

BEGIN;

-- ─── config_costo_laboral ──────────────────────────────────────────────────
CREATE TABLE config_costo_laboral (
  id                             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  salario_basico_chofer          numeric(14,2) NOT NULL CHECK (salario_basico_chofer   >= 0),
  salario_basico_ayudante        numeric(14,2) NOT NULL CHECK (salario_basico_ayudante >= 0),
  adicionales_pct_chofer         numeric(5,2)  NOT NULL CHECK (adicionales_pct_chofer   BETWEEN 0 AND 100),
  adicionales_pct_ayudante       numeric(5,2)  NOT NULL CHECK (adicionales_pct_ayudante BETWEEN 0 AND 100),
  viaticos_diarios               numeric(12,2) NOT NULL CHECK (viaticos_diarios >= 0),
  contribuciones_seg_social_pct  numeric(5,2)  NOT NULL CHECK (contribuciones_seg_social_pct BETWEEN 0 AND 100),
  obra_social_pct                numeric(5,2)  NOT NULL CHECK (obra_social_pct BETWEEN 0 AND 100),
  art_pct                        numeric(5,2)  NOT NULL CHECK (art_pct BETWEEN 0 AND 100),
  seguro_vida_mensual            numeric(12,2) NOT NULL CHECK (seguro_vida_mensual >= 0),
  horas_mensuales                numeric(6,2)  NOT NULL CHECK (horas_mensuales > 0),
  horas_diarias                  numeric(4,2)  NOT NULL CHECK (horas_diarias > 0),
  vigente_desde                  timestamptz   NOT NULL DEFAULT now(),
  vigente_hasta                  timestamptz,
  CONSTRAINT chk_vigencia_costo_laboral CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde)
);
ALTER TABLE config_costo_laboral ENABLE ROW LEVEL SECURITY;
CREATE UNIQUE INDEX idx_config_costo_laboral_vigente
  ON config_costo_laboral (vigente_hasta) WHERE vigente_hasta IS NULL;

-- ─── config_operacion (tiempos de carga/descarga) ──────────────────────────
CREATE TABLE config_operacion (
  id                         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tiempo_base_operacion_min  numeric(6,2) NOT NULL CHECK (tiempo_base_operacion_min >= 0),
  tiempo_espera_min          numeric(6,2) NOT NULL CHECK (tiempo_espera_min >= 0),
  tiempo_por_objeto_min      numeric(6,3) NOT NULL CHECK (tiempo_por_objeto_min >= 0),
  tiempo_por_kg_min          numeric(6,3) NOT NULL CHECK (tiempo_por_kg_min >= 0),
  tiempo_por_m3_min          numeric(6,3) NOT NULL CHECK (tiempo_por_m3_min >= 0),
  tiempo_por_metro_min       numeric(6,3) NOT NULL CHECK (tiempo_por_metro_min >= 0),
  tiempo_por_escalera_min    numeric(6,3) NOT NULL CHECK (tiempo_por_escalera_min >= 0),
  eficiencia_ayudante        numeric(4,3) NOT NULL CHECK (eficiencia_ayudante BETWEEN 0 AND 1),
  vigente_desde              timestamptz  NOT NULL DEFAULT now(),
  vigente_hasta              timestamptz,
  CONSTRAINT chk_vigencia_operacion CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde)
);
ALTER TABLE config_operacion ENABLE ROW LEVEL SECURITY;
CREATE UNIQUE INDEX idx_config_operacion_vigente
  ON config_operacion (vigente_hasta) WHERE vigente_hasta IS NULL;

COMMENT ON COLUMN config_operacion.tiempo_espera_min IS
  'Reservado: declarado en el documento fuente pero no usado en la fórmula. Confirmar si se suma.';

-- ─── config_impuesto (IVA) ─────────────────────────────────────────────────
CREATE TABLE config_impuesto (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  iva_pct        numeric(5,2) NOT NULL CHECK (iva_pct BETWEEN 0 AND 100),
  vigente_desde  timestamptz  NOT NULL DEFAULT now(),
  vigente_hasta  timestamptz,
  CONSTRAINT chk_vigencia_impuesto CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde)
);
ALTER TABLE config_impuesto ENABLE ROW LEVEL SECURITY;
CREATE UNIQUE INDEX idx_config_impuesto_vigente
  ON config_impuesto (vigente_hasta) WHERE vigente_hasta IS NULL;

-- ─── RLS: patrón de catálogo (lectura para autenticados, ABM solo admin) ───
-- Lectura abierta porque con RLS pass-through (D-02) el cálculo de la oferta
-- corre con el rol del Transportista, que necesita leer estos parámetros.
-- Sin FOR ALL: split explícito (P3).
CREATE POLICY config_costo_laboral_select ON config_costo_laboral FOR SELECT USING (true);
CREATE POLICY config_costo_laboral_insert ON config_costo_laboral FOR INSERT WITH CHECK (fn_es_administrador());
CREATE POLICY config_costo_laboral_update ON config_costo_laboral FOR UPDATE USING (fn_es_administrador()) WITH CHECK (fn_es_administrador());
CREATE POLICY config_costo_laboral_delete ON config_costo_laboral FOR DELETE USING (fn_es_administrador());

CREATE POLICY config_operacion_select ON config_operacion FOR SELECT USING (true);
CREATE POLICY config_operacion_insert ON config_operacion FOR INSERT WITH CHECK (fn_es_administrador());
CREATE POLICY config_operacion_update ON config_operacion FOR UPDATE USING (fn_es_administrador()) WITH CHECK (fn_es_administrador());
CREATE POLICY config_operacion_delete ON config_operacion FOR DELETE USING (fn_es_administrador());

CREATE POLICY config_impuesto_select ON config_impuesto FOR SELECT USING (true);
CREATE POLICY config_impuesto_insert ON config_impuesto FOR INSERT WITH CHECK (fn_es_administrador());
CREATE POLICY config_impuesto_update ON config_impuesto FOR UPDATE USING (fn_es_administrador()) WITH CHECK (fn_es_administrador());
CREATE POLICY config_impuesto_delete ON config_impuesto FOR DELETE USING (fn_es_administrador());

-- ─── Seed ILUSTRATIVA (valores del borrador / documento fuente) ────────────
-- Reemplazar por valores vigentes reales antes de producción.
INSERT INTO config_costo_laboral (
  salario_basico_chofer, salario_basico_ayudante,
  adicionales_pct_chofer, adicionales_pct_ayudante,
  viaticos_diarios, contribuciones_seg_social_pct, obra_social_pct, art_pct,
  seguro_vida_mensual, horas_mensuales, horas_diarias
) VALUES (
  1037544.76, 963809.78,
  18.00, 16.00,
  24724.33, 18.00, 6.00, 10.00,
  424.62, 192.00, 8.00
);

INSERT INTO config_operacion (
  tiempo_base_operacion_min, tiempo_espera_min, tiempo_por_objeto_min, tiempo_por_kg_min,
  tiempo_por_m3_min, tiempo_por_metro_min, tiempo_por_escalera_min, eficiencia_ayudante
) VALUES (10.00, 30.00, 2.000, 0.010, 8.000, 0.020, 5.000, 0.700);

INSERT INTO config_impuesto (iva_pct) VALUES (21.00);

-- ─── config_tarifa: deprecación (DROP en migración posterior) ──────────────
COMMENT ON TABLE config_tarifa IS
  'DEPRECATED: modelo de tarifa plana reemplazado por config_costo_laboral + config_operacion + vehiculo_costo (RN-01). Se elimina en una migración posterior.';

COMMIT;

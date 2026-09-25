-- 0001_add_dimensiones_objeto_solicitud_objeto.sql
-- Requisito: RN-02, RN-08 (dimensiones reales para empaquetado con boxpacker3 v2)
-- Reversible: sí (DROP COLUMN de las columnas nuevas; SET NOT NULL de nuevo sobre volumen_unitario_m3)
-- Afecta: objeto, solicitud_objeto (columnas nuevas + deprecación de volumen_unitario_m3). Sin cambios de RLS.
--
-- ✅ APLICADA el 2026-09-24 vía apply_migration (confirmación humana explícita).
--    Diseño: docs/ALGORITMO_COTIZACION.md y docs/ALGORITMO_VIAJES_EMPAQUETADO.md.

BEGIN;

-- ─── objeto (catálogo, 5 filas semilla hoy) ────────────────────────────────
-- Nullable en esta etapa: las 5 filas semilla no tienen dimensiones cargadas.
-- Pasar a NOT NULL en una migración posterior, una vez que el Administrador
-- complete largo/ancho/alto de cada objeto del catálogo.
ALTER TABLE objeto
  ADD COLUMN largo_m             numeric(6,3) CHECK (largo_m > 0),
  ADD COLUMN ancho_m             numeric(6,3) CHECK (ancho_m > 0),
  ADD COLUMN alto_m              numeric(6,3) CHECK (alto_m  > 0),
  -- Restricciones aplicadas por boxpacker3 v2 (ver docs/ALGORITMO_VIAJES_EMPAQUETADO.md §2.1).
  -- alto_m es el eje vertical. Default true = sin restricción.
  ADD COLUMN rotacion_horizontal boolean NOT NULL DEFAULT true,
  ADD COLUMN rotacion_vertical   boolean NOT NULL DEFAULT true,
  ADD COLUMN apilable            boolean NOT NULL DEFAULT true;

COMMENT ON COLUMN objeto.rotacion_horizontal IS
  'false = no girar sobre el piso. Junto con rotacion_vertical=false → sin rotación (RotationNever). Sola (con rotacion_vertical=true) es informativa: boxpacker3 v2 no la soporta directo.';
COMMENT ON COLUMN objeto.rotacion_vertical IS
  'false = no se puede volcar: alto_m queda vertical (boxpacker3 v2 VerticalAxes=[DepthAxis]).';
COMMENT ON COLUMN objeto.apilable IS
  'false = nada encima (boxpacker3 v2 NothingOnTop).';
COMMENT ON COLUMN objeto.volumen_estimado_m3 IS
  'Redundante con largo_m*ancho_m*alto_m una vez cargadas las dimensiones. Candidata a DEPRECATED.';

-- ─── solicitud_objeto (vacía hoy) ──────────────────────────────────────────
-- Sigue el patrón actual: la fila guarda su propia copia de peso (y ahora de
-- dimensiones) tanto para ítems de catálogo como manuales. Tabla vacía →
-- NOT NULL sin default es seguro.
ALTER TABLE solicitud_objeto
  ADD COLUMN largo_m             numeric(6,3) NOT NULL CHECK (largo_m > 0),
  ADD COLUMN ancho_m             numeric(6,3) NOT NULL CHECK (ancho_m > 0),
  ADD COLUMN alto_m              numeric(6,3) NOT NULL CHECK (alto_m  > 0),
  ADD COLUMN rotacion_horizontal boolean NOT NULL DEFAULT true,
  ADD COLUMN rotacion_vertical   boolean NOT NULL DEFAULT true,
  ADD COLUMN apilable            boolean NOT NULL DEFAULT true;

-- volumen_unitario_m3 pasa a ser derivable → deprecar (DROP en migración posterior).
ALTER TABLE solicitud_objeto ALTER COLUMN volumen_unitario_m3 DROP NOT NULL;
COMMENT ON COLUMN solicitud_objeto.volumen_unitario_m3 IS
  'DEPRECATED: derivable de largo_m*ancho_m*alto_m. Se elimina en una migración posterior.';

COMMIT;

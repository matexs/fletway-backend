-- 0005_add_costos_oferta.sql
-- Requisito: RF-17, RN-01 (precio real por oferta, auditable)
-- Reversible: sí (DROP COLUMN de las columnas nuevas)
-- Afecta: oferta (columnas nuevas). Sin cambios de RLS (ver advertencia abajo).
--
-- ✅ APLICADA el 2026-09-24 vía apply_migration (confirmación humana explícita).
--    Diseño: docs/ALGORITMO_COTIZACION.md y docs/ALGORITMO_VIAJES_EMPAQUETADO.md.
--
-- Por qué el desglose vive en oferta (y no solo costo_operativo):
--   viaje_insert lo ejecuta el CLIENTE (policy: cliente_id = uid). Con RLS
--   pass-through, esa sesión no puede leer vehiculo_costo ni recalcular nada:
--   el snapshot de viaje tiene que COPIARSE desde la oferta aceptada.
--
-- ⚠️ Advertencia (no resuelta acá): oferta_select permite al Cliente leer
--   todas las columnas de las ofertas de sus solicitudes → verá costo_operativo
--   y el desglose. Y oferta_update permite a Cliente y Transportista modificar
--   cualquier columna (incluido precio_calculado, ya hoy). Ver §7 del doc.

BEGIN;

-- Tabla vacía hoy → NOT NULL sin default es seguro.
ALTER TABLE oferta
  ADD COLUMN distancia_km          numeric(8,2)  NOT NULL CHECK (distancia_km >= 0),
  ADD COLUMN duracion_ruta_h       numeric(6,2)  NOT NULL CHECK (duracion_ruta_h >= 0),
  ADD COLUMN duracion_operacion_h  numeric(6,2)  NOT NULL CHECK (duracion_operacion_h >= 0),
  ADD COLUMN costo_laboral         numeric(12,2) NOT NULL CHECK (costo_laboral >= 0),
  ADD COLUMN costo_vehiculo        numeric(12,2) NOT NULL CHECK (costo_vehiculo >= 0),
  ADD COLUMN costos_adicionales    numeric(12,2) NOT NULL DEFAULT 0 CHECK (costos_adicionales >= 0),
  ADD COLUMN costo_operativo       numeric(12,2) NOT NULL CHECK (costo_operativo >= 0),
  ADD COLUMN margen_pct            numeric(5,2)  NOT NULL CHECK (margen_pct >= 0),
  ADD COLUMN precio_neto           numeric(12,2) NOT NULL CHECK (precio_neto >= 0),
  ADD COLUMN porcentaje_comision   numeric(5,2)  NOT NULL CHECK (porcentaje_comision BETWEEN 0 AND 100),
  ADD COLUMN iva_pct               numeric(5,2)  NOT NULL CHECK (iva_pct BETWEEN 0 AND 100);

COMMENT ON COLUMN oferta.duracion_operacion_h IS 'Suma de duracion_operacion() de todos los viajes de la oferta.';
COMMENT ON COLUMN oferta.costo_operativo IS
  'Suma de calcular_costo_viaje() de todos los viajes. Sin margen, comisión ni IVA. No debe exponerse al Cliente por API.';
COMMENT ON COLUMN oferta.margen_pct IS
  'Margen APLICADO a esta oferta (valor congelado). De dónde sale (plataforma vs. Transportista) es una decisión abierta.';
COMMENT ON COLUMN oferta.precio_calculado IS
  'Precio final al Cliente = precio_neto * (1 + iva_pct/100). Único monto visible para el Cliente.';

COMMIT;

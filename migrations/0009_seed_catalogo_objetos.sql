-- 0009_seed_catalogo_objetos.sql
-- Requisito: RN-08 (catálogo de objetos), RN-02. Módulo 0 de docs/PLAN_CONSTRUCCION.md (§2.2).
-- Reversible: sí (DROP NOT NULL de las medidas; DELETE de los 23 objetos nuevos; UPDATE de las
--             medidas de los 5 existentes a NULL)
-- Afecta: objeto (datos + NOT NULL en largo_m, ancho_m y alto_m).
--
-- APLICADA el 2026-10-02 vía apply_migration (confirmación humana explícita), después de
-- probarla en Supabase local (make db-local).
--
-- Completa las medidas de los 5 objetos existentes y suma 23. alto_m es el eje vertical;
-- rotacion_vertical = false en lo que no se puede acostar; apilable = false en lo frágil o
-- pesado. volumen_estimado_m3 es NOT NULL: en los INSERT se carga el producto de las medidas.
-- Al final las medidas pasan a NOT NULL (pendiente desde 0001).

BEGIN;

UPDATE objeto SET largo_m=1.40, ancho_m=0.80, alto_m=0.75, rotacion_horizontal=true,  rotacion_vertical=true,  apilable=false WHERE nombre='Mesa de comedor';
UPDATE objeto SET largo_m=0.70, ancho_m=0.70, alto_m=1.80, rotacion_horizontal=true,  rotacion_vertical=false, apilable=false WHERE nombre='Heladera';
UPDATE objeto SET largo_m=1.40, ancho_m=1.90, alto_m=0.55, rotacion_horizontal=true,  rotacion_vertical=true,  apilable=false WHERE nombre='Cama matrimonial (colchón + base)';
UPDATE objeto SET largo_m=1.60, ancho_m=0.90, alto_m=0.85, rotacion_horizontal=true,  rotacion_vertical=true,  apilable=false WHERE nombre='Sofá 2 cuerpos';
UPDATE objeto SET largo_m=0.50, ancho_m=0.40, alto_m=0.40, rotacion_horizontal=true,  rotacion_vertical=true,  apilable=true  WHERE nombre='Caja mudanza estándar';

INSERT INTO objeto (nombre, peso_estimado_kg, volumen_estimado_m3, largo_m, ancho_m, alto_m, rotacion_horizontal, rotacion_vertical, apilable) VALUES
  ('Caja mudanza chica', 10, 0.036, 0.40, 0.30, 0.30, true,  true,  true),
  ('Caja mudanza grande', 25, 0.150, 0.60, 0.50, 0.50, true,  true,  true),
  ('Cama 1 plaza (colchón + base)', 30, 0.855, 0.90, 1.90, 0.50, true,  true,  false),
  ('Cama 2 plazas (colchón + base)', 45, 1.463, 1.40, 1.90, 0.55, true,  true,  false),
  ('Ropero / placard 2 cuerpos', 60, 1.440, 1.20, 0.60, 2.00, true,  false, false),
  ('Cómoda / cajonera', 35, 0.450, 1.00, 0.50, 0.90, true,  false, true),
  ('Sofá 3 cuerpos', 55, 1.530, 2.00, 0.90, 0.85, true,  true,  false),
  ('Sillón individual', 20, 0.544, 0.80, 0.80, 0.85, true,  true,  false),
  ('Mesa ratona', 12, 0.225, 1.00, 0.50, 0.45, true,  true,  true),
  ('Silla (comedor)', 5, 0.182, 0.45, 0.45, 0.90, true,  true,  true),
  ('Mesa de luz', 8, 0.099, 0.45, 0.40, 0.55, true,  true,  true),
  ('Escritorio', 25, 0.540, 1.20, 0.60, 0.75, true,  true,  false),
  ('Biblioteca / estantería', 35, 0.486, 0.90, 0.30, 1.80, true,  false, false),
  ('TV (hasta 55", embalada)', 15, 0.156, 1.30, 0.15, 0.80, false, false, false),
  ('Microondas', 12, 0.060, 0.50, 0.40, 0.30, true,  false, true),
  ('Horno eléctrico / anafe', 15, 0.120, 0.60, 0.50, 0.40, true,  false, true),
  ('Lavarropas', 65, 0.306, 0.60, 0.60, 0.85, true,  false, false),
  ('Lavavajillas', 40, 0.306, 0.60, 0.60, 0.85, true,  false, false),
  ('Aire acondicionado split (embalado)', 25, 0.081, 0.90, 0.30, 0.30, true,  false, true),
  ('Estufa / calefactor portátil', 10, 0.112, 0.40, 0.40, 0.70, true,  false, true),
  ('Bicicleta', 15, 1.020, 1.70, 0.60, 1.00, true,  true,  false),
  ('Espejo / cuadro grande', 8, 0.048, 1.20, 0.05, 0.80, false, false, false),
  ('Bulto de ropa / valija', 10, 0.072, 0.60, 0.40, 0.30, true,  true,  true);

ALTER TABLE objeto
  ALTER COLUMN largo_m SET NOT NULL,
  ALTER COLUMN ancho_m SET NOT NULL,
  ALTER COLUMN alto_m  SET NOT NULL;

COMMIT;

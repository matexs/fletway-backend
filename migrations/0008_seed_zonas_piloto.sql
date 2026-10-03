-- 0008_seed_zonas_piloto.sql
-- Requisito: RN-04 (matchmaking por zona), D-20, D-21. Módulo 0 de docs/PLAN_CONSTRUCCION.md (§2.1).
-- Reversible: sí (DELETE FROM zona WHERE nombre IN (...) mientras no haya transportista_zona ni solicitudes que las usen)
-- Afecta: zona (datos). Sin cambios de esquema ni de RLS.
--
-- Zonas piloto: CABA y Zona Norte del GBA, granularidad partido/localidad. Agregar zonas
-- después es un ABM del Administrador.

BEGIN;

INSERT INTO zona (nombre, provincia) VALUES
  ('Ciudad Autónoma de Buenos Aires', 'CABA'),
  ('Campana', 'Buenos Aires'),
  ('Zárate', 'Buenos Aires'),
  ('Escobar', 'Buenos Aires'),
  ('Pilar', 'Buenos Aires'),
  ('Tigre', 'Buenos Aires'),
  ('San Fernando', 'Buenos Aires'),
  ('San Isidro', 'Buenos Aires'),
  ('Vicente López', 'Buenos Aires'),
  ('San Martín', 'Buenos Aires'),
  ('Tres de Febrero', 'Buenos Aires'),
  ('San Miguel', 'Buenos Aires'),
  ('Malvinas Argentinas', 'Buenos Aires'),
  ('José C. Paz', 'Buenos Aires');

COMMIT;

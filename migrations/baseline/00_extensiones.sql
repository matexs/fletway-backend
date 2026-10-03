-- Extensiones que el esquema de dbFletway necesita y que el dump del schema public no
-- incluye (D-16). Mismas extensiones y schema que en producción
-- (ver docs/VERIFICACION_ESQUEMA_2026-09-07.md §3.6). Idempotente.
CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA extensions;
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA extensions;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA extensions;

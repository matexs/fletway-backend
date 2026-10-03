-- 0010_corregir_recursion_rls_solicitud.sql
-- Requisito: RNF-01 (RLS pass-through, D-02). Bloquea el módulo 2: GET /api/me lee usuario.
-- Reversible: sí (restaurar solicitud_select con el EXISTS sobre oferta y DROP de la función;
--   vuelve a dejar la recursión).
-- Afecta: policy solicitud_select (reemplazada); función nueva fn_tiene_oferta_en_solicitud.
--
-- APLICADA el 2026-10-02 vía apply_migration (confirmación humana explícita), después de
-- probarla en Supabase local (make db-local).
--
-- Problema (presente en dbFletway desde el esquema inicial): solicitud_select consulta oferta
-- y oferta_select consulta solicitud. Postgres aplica RLS dentro de esas subconsultas, así que
-- cualquier lectura autenticada de solicitud u oferta termina en
--   ERROR: infinite recursion detected in policy for relation "oferta"
-- y arrastra a las tablas cuyas policies leen alguna de las dos (usuario, cliente,
-- solicitud_objeto). Con el JWT de cualquier usuario, SELECT * FROM usuario falla.
--
-- Corrección: la rama "el Transportista ya ofertó en esta solicitud" de solicitud_select pasa
-- a una función SECURITY DEFINER que lee oferta sin RLS, filtrando por auth.uid(). El resto de
-- la policy queda igual. Es el mismo patrón que fn_es_administrador.

BEGIN;

CREATE FUNCTION public.fn_tiene_oferta_en_solicitud(p_solicitud_id uuid)
RETURNS boolean
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
  SELECT EXISTS (
    SELECT 1 FROM oferta
    WHERE solicitud_id = p_solicitud_id AND transportista_id = (select auth.uid())
  );
$function$;

DROP POLICY solicitud_select ON solicitud;
CREATE POLICY solicitud_select ON solicitud FOR SELECT
  USING (
    cliente_id = (select auth.uid())
    OR fn_es_administrador()
    OR (
      estado_codigo = 'publicada'
      AND EXISTS (
        SELECT 1 FROM transportista_zona tz
        WHERE tz.transportista_id = (select auth.uid())
          AND tz.zona_id = ANY (ARRAY[solicitud.origen_zona_id, solicitud.destino_zona_id])
      )
    )
    OR fn_tiene_oferta_en_solicitud(id)
  );

COMMIT;

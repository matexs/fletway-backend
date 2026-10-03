-- 0011_crear_alta_usuario.sql
-- Requisito: RF-05, RF-16 (alta de cuenta), D-18. Módulo 2 de docs/PLAN_CONSTRUCCION.md.
-- Reversible: sí (DROP de los triggers y funciones nuevas; restaurar las policies
--   cliente_insert y transportista_insert con WITH CHECK (usuario_id = (select auth.uid()))).
-- Afecta: trigger nuevo en auth.users; triggers nuevos en usuario y transportista;
--   policies cliente_insert y transportista_insert (reemplazadas).
--
-- 1. Alta (D-18): la app hace signUp con nombre_completo, telefono y rol en la metadata y
--    este trigger crea la fila usuario. Sólo acepta rol cliente o transportista: la metadata
--    la controla el propio usuario, así que cualquier otro valor (incluido administrador)
--    aborta el signUp.
-- 2. usuario: un usuario común no puede cambiar su rol, su email ni su estado activo
--    (la policy usuario_update le deja editar su fila entera).
-- 3. transportista: el alta directa (PostgREST) podía nacer con estado 'habilitado' o con
--    calificación cargada; trg_proteger_campos_transportista sólo cubría UPDATE.
-- 4. cliente_insert / transportista_insert: la fila de rol sólo se crea si coincide con
--    usuario.rol (un rol por cuenta, D-17).
--
-- Las protecciones aplican a requests con JWT (auth.role() anon o authenticated) que no son
-- de un Administrador. Las operaciones de mantenimiento sin JWT (bootstrap del primer
-- Administrador, D-18) no se ven afectadas.

BEGIN;

-- 1. Alta de usuario desde Supabase Auth -------------------------------------------------

CREATE FUNCTION public.fn_alta_usuario()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
DECLARE
  v_rol             text := NEW.raw_user_meta_data ->> 'rol';
  v_nombre_completo text := btrim(NEW.raw_user_meta_data ->> 'nombre_completo');
  v_telefono        text := btrim(NEW.raw_user_meta_data ->> 'telefono');
BEGIN
  IF v_rol IS NULL OR v_rol NOT IN ('cliente', 'transportista') THEN
    RAISE EXCEPTION 'alta de usuario: rol inválido'
      USING ERRCODE = 'check_violation', HINT = 'rol debe ser cliente o transportista';
  END IF;
  IF coalesce(v_nombre_completo, '') = '' OR coalesce(v_telefono, '') = '' THEN
    RAISE EXCEPTION 'alta de usuario: faltan nombre_completo o telefono'
      USING ERRCODE = 'not_null_violation';
  END IF;

  INSERT INTO usuario (id, email, telefono, nombre_completo, rol)
  VALUES (NEW.id, NEW.email, v_telefono, v_nombre_completo, v_rol);
  RETURN NEW;
END;
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_alta_usuario() FROM PUBLIC, anon, authenticated;

CREATE TRIGGER trg_alta_usuario
  AFTER INSERT ON auth.users
  FOR EACH ROW EXECUTE FUNCTION public.fn_alta_usuario();

-- 2. Campos de usuario que sólo cambia un Administrador ----------------------------------

CREATE FUNCTION public.fn_proteger_campos_usuario()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
BEGIN
  IF (select auth.role()) IN ('anon', 'authenticated') AND NOT fn_es_administrador() THEN
    NEW.id := OLD.id;
    NEW.rol := OLD.rol;
    NEW.email := OLD.email;
    NEW.activo := OLD.activo;
  END IF;
  RETURN NEW;
END;
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_proteger_campos_usuario() FROM PUBLIC, anon, authenticated;

CREATE TRIGGER trg_proteger_campos_usuario
  BEFORE UPDATE ON usuario
  FOR EACH ROW EXECUTE FUNCTION fn_proteger_campos_usuario();

-- 3. Alta de transportista siempre pendiente ---------------------------------------------

CREATE FUNCTION public.fn_proteger_alta_transportista()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'public'
AS $function$
BEGIN
  IF (select auth.role()) IN ('anon', 'authenticated') AND NOT fn_es_administrador() THEN
    NEW.estado_habilitacion_codigo := 'pendiente';
    NEW.calificacion_promedio := NULL;
    NEW.tasa_cumplimiento := NULL;
  END IF;
  RETURN NEW;
END;
$function$;

REVOKE EXECUTE ON FUNCTION public.fn_proteger_alta_transportista() FROM PUBLIC, anon, authenticated;

CREATE TRIGGER trg_proteger_alta_transportista
  BEFORE INSERT ON transportista
  FOR EACH ROW EXECUTE FUNCTION fn_proteger_alta_transportista();

-- 4. La fila de rol tiene que coincidir con usuario.rol ----------------------------------

DROP POLICY cliente_insert ON cliente;
CREATE POLICY cliente_insert ON cliente FOR INSERT TO authenticated
  WITH CHECK (
    usuario_id = (select auth.uid())
    AND EXISTS (SELECT 1 FROM usuario u WHERE u.id = usuario_id AND u.rol = 'cliente')
  );

DROP POLICY transportista_insert ON transportista;
CREATE POLICY transportista_insert ON transportista FOR INSERT TO authenticated
  WITH CHECK (
    usuario_id = (select auth.uid())
    AND EXISTS (SELECT 1 FROM usuario u WHERE u.id = usuario_id AND u.rol = 'transportista')
  );

COMMIT;

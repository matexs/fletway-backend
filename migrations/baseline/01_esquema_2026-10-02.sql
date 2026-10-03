


SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;


CREATE SCHEMA IF NOT EXISTS "public";


ALTER SCHEMA "public" OWNER TO "pg_database_owner";


COMMENT ON SCHEMA "public" IS 'standard public schema';



CREATE OR REPLACE FUNCTION "public"."fn_es_administrador"() RETURNS boolean
    LANGUAGE "sql" STABLE SECURITY DEFINER
    SET "search_path" TO 'public'
    AS $$
  SELECT EXISTS (SELECT 1 FROM administrador WHERE usuario_id = (select auth.uid()));
$$;


ALTER FUNCTION "public"."fn_es_administrador"() OWNER TO "postgres";


CREATE OR REPLACE FUNCTION "public"."fn_es_transportista_habilitado"() RETURNS boolean
    LANGUAGE "sql" STABLE SECURITY DEFINER
    SET "search_path" TO 'public'
    AS $$
  SELECT EXISTS (
    SELECT 1 FROM transportista
    WHERE usuario_id = (select auth.uid()) AND estado_habilitacion_codigo = 'habilitado'
  );
$$;


ALTER FUNCTION "public"."fn_es_transportista_habilitado"() OWNER TO "postgres";


CREATE OR REPLACE FUNCTION "public"."fn_proteger_campos_transportista"() RETURNS "trigger"
    LANGUAGE "plpgsql" SECURITY DEFINER
    SET "search_path" TO 'public'
    AS $$
BEGIN
  IF NOT fn_es_administrador() THEN
    NEW.estado_habilitacion_codigo := OLD.estado_habilitacion_codigo;
    NEW.calificacion_promedio := OLD.calificacion_promedio;
    NEW.tasa_cumplimiento := OLD.tasa_cumplimiento;
  END IF;
  RETURN NEW;
END;
$$;


ALTER FUNCTION "public"."fn_proteger_campos_transportista"() OWNER TO "postgres";


CREATE OR REPLACE FUNCTION "public"."fn_proteger_campos_viaje"() RETURNS "trigger"
    LANGUAGE "plpgsql" SECURITY DEFINER
    SET "search_path" TO 'public'
    AS $$
BEGIN
  IF NOT fn_es_administrador() THEN
    NEW.oferta_id := OLD.oferta_id;
    NEW.solicitud_id := OLD.solicitud_id;
    NEW.cliente_id := OLD.cliente_id;
    NEW.transportista_id := OLD.transportista_id;
    NEW.monto_total_snapshot := OLD.monto_total_snapshot;
    NEW.porcentaje_comision_snapshot := OLD.porcentaje_comision_snapshot;
    -- nuevos (0006)
    NEW.cantidad_viajes_snapshot := OLD.cantidad_viajes_snapshot;
    NEW.duracion_ruta_h_snapshot := OLD.duracion_ruta_h_snapshot;
    NEW.duracion_operacion_h_snapshot := OLD.duracion_operacion_h_snapshot;
    NEW.costo_laboral_snapshot := OLD.costo_laboral_snapshot;
    NEW.costo_vehiculo_snapshot := OLD.costo_vehiculo_snapshot;
    NEW.costos_adicionales_snapshot := OLD.costos_adicionales_snapshot;
    NEW.costo_operativo_snapshot := OLD.costo_operativo_snapshot;
    NEW.margen_pct_snapshot := OLD.margen_pct_snapshot;
    NEW.precio_neto_snapshot := OLD.precio_neto_snapshot;
    NEW.iva_pct_snapshot := OLD.iva_pct_snapshot;
    NEW.distancia_km_snapshot := OLD.distancia_km_snapshot;
    -- DEPRECATED: se quitan de acá en la misma migración que haga su DROP
    NEW.tarifa_base_snapshot := OLD.tarifa_base_snapshot;
    NEW.valor_por_km_snapshot := OLD.valor_por_km_snapshot;
    NEW.valor_por_m3_snapshot := OLD.valor_por_m3_snapshot;
    NEW.valor_por_hora_snapshot := OLD.valor_por_hora_snapshot;
    NEW.recargo_escalera_snapshot := OLD.recargo_escalera_snapshot;
    NEW.recargo_ayudante_snapshot := OLD.recargo_ayudante_snapshot;
  END IF;
  RETURN NEW;
END;
$$;


ALTER FUNCTION "public"."fn_proteger_campos_viaje"() OWNER TO "postgres";

SET default_tablespace = '';

SET default_table_access_method = "heap";


CREATE TABLE IF NOT EXISTS "public"."administrador" (
    "usuario_id" "uuid" NOT NULL
);


ALTER TABLE "public"."administrador" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."cliente" (
    "usuario_id" "uuid" NOT NULL
);


ALTER TABLE "public"."cliente" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."config_comision" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "porcentaje" numeric(5,2) NOT NULL,
    "vigente_desde" timestamp with time zone DEFAULT "now"() NOT NULL,
    "vigente_hasta" timestamp with time zone,
    CONSTRAINT "chk_vigencia_comision" CHECK ((("vigente_hasta" IS NULL) OR ("vigente_hasta" > "vigente_desde"))),
    CONSTRAINT "config_comision_porcentaje_check" CHECK ((("porcentaje" >= (0)::numeric) AND ("porcentaje" <= (100)::numeric)))
);


ALTER TABLE "public"."config_comision" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."config_costo_laboral" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "salario_basico_chofer" numeric(14,2) NOT NULL,
    "salario_basico_ayudante" numeric(14,2) NOT NULL,
    "adicionales_pct_chofer" numeric(5,2) NOT NULL,
    "adicionales_pct_ayudante" numeric(5,2) NOT NULL,
    "viaticos_diarios" numeric(12,2) NOT NULL,
    "contribuciones_seg_social_pct" numeric(5,2) NOT NULL,
    "obra_social_pct" numeric(5,2) NOT NULL,
    "art_pct" numeric(5,2) NOT NULL,
    "seguro_vida_mensual" numeric(12,2) NOT NULL,
    "horas_mensuales" numeric(6,2) NOT NULL,
    "horas_diarias" numeric(4,2) NOT NULL,
    "vigente_desde" timestamp with time zone DEFAULT "now"() NOT NULL,
    "vigente_hasta" timestamp with time zone,
    CONSTRAINT "chk_vigencia_costo_laboral" CHECK ((("vigente_hasta" IS NULL) OR ("vigente_hasta" > "vigente_desde"))),
    CONSTRAINT "config_costo_laboral_adicionales_pct_ayudante_check" CHECK ((("adicionales_pct_ayudante" >= (0)::numeric) AND ("adicionales_pct_ayudante" <= (100)::numeric))),
    CONSTRAINT "config_costo_laboral_adicionales_pct_chofer_check" CHECK ((("adicionales_pct_chofer" >= (0)::numeric) AND ("adicionales_pct_chofer" <= (100)::numeric))),
    CONSTRAINT "config_costo_laboral_art_pct_check" CHECK ((("art_pct" >= (0)::numeric) AND ("art_pct" <= (100)::numeric))),
    CONSTRAINT "config_costo_laboral_contribuciones_seg_social_pct_check" CHECK ((("contribuciones_seg_social_pct" >= (0)::numeric) AND ("contribuciones_seg_social_pct" <= (100)::numeric))),
    CONSTRAINT "config_costo_laboral_horas_diarias_check" CHECK (("horas_diarias" > (0)::numeric)),
    CONSTRAINT "config_costo_laboral_horas_mensuales_check" CHECK (("horas_mensuales" > (0)::numeric)),
    CONSTRAINT "config_costo_laboral_obra_social_pct_check" CHECK ((("obra_social_pct" >= (0)::numeric) AND ("obra_social_pct" <= (100)::numeric))),
    CONSTRAINT "config_costo_laboral_salario_basico_ayudante_check" CHECK (("salario_basico_ayudante" >= (0)::numeric)),
    CONSTRAINT "config_costo_laboral_salario_basico_chofer_check" CHECK (("salario_basico_chofer" >= (0)::numeric)),
    CONSTRAINT "config_costo_laboral_seguro_vida_mensual_check" CHECK (("seguro_vida_mensual" >= (0)::numeric)),
    CONSTRAINT "config_costo_laboral_viaticos_diarios_check" CHECK (("viaticos_diarios" >= (0)::numeric))
);


ALTER TABLE "public"."config_costo_laboral" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."config_impuesto" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "iva_pct" numeric(5,2) NOT NULL,
    "vigente_desde" timestamp with time zone DEFAULT "now"() NOT NULL,
    "vigente_hasta" timestamp with time zone,
    CONSTRAINT "chk_vigencia_impuesto" CHECK ((("vigente_hasta" IS NULL) OR ("vigente_hasta" > "vigente_desde"))),
    CONSTRAINT "config_impuesto_iva_pct_check" CHECK ((("iva_pct" >= (0)::numeric) AND ("iva_pct" <= (100)::numeric)))
);


ALTER TABLE "public"."config_impuesto" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."config_operacion" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "tiempo_base_operacion_min" numeric(6,2) NOT NULL,
    "tiempo_espera_min" numeric(6,2) NOT NULL,
    "tiempo_por_objeto_min" numeric(6,3) NOT NULL,
    "tiempo_por_kg_min" numeric(6,3) NOT NULL,
    "tiempo_por_m3_min" numeric(6,3) NOT NULL,
    "tiempo_por_metro_min" numeric(6,3) NOT NULL,
    "tiempo_por_escalera_min" numeric(6,3) NOT NULL,
    "eficiencia_ayudante" numeric(4,3) NOT NULL,
    "vigente_desde" timestamp with time zone DEFAULT "now"() NOT NULL,
    "vigente_hasta" timestamp with time zone,
    CONSTRAINT "chk_vigencia_operacion" CHECK ((("vigente_hasta" IS NULL) OR ("vigente_hasta" > "vigente_desde"))),
    CONSTRAINT "config_operacion_eficiencia_ayudante_check" CHECK ((("eficiencia_ayudante" >= (0)::numeric) AND ("eficiencia_ayudante" <= (1)::numeric))),
    CONSTRAINT "config_operacion_tiempo_base_operacion_min_check" CHECK (("tiempo_base_operacion_min" >= (0)::numeric)),
    CONSTRAINT "config_operacion_tiempo_espera_min_check" CHECK (("tiempo_espera_min" >= (0)::numeric)),
    CONSTRAINT "config_operacion_tiempo_por_escalera_min_check" CHECK (("tiempo_por_escalera_min" >= (0)::numeric)),
    CONSTRAINT "config_operacion_tiempo_por_kg_min_check" CHECK (("tiempo_por_kg_min" >= (0)::numeric)),
    CONSTRAINT "config_operacion_tiempo_por_m3_min_check" CHECK (("tiempo_por_m3_min" >= (0)::numeric)),
    CONSTRAINT "config_operacion_tiempo_por_metro_min_check" CHECK (("tiempo_por_metro_min" >= (0)::numeric)),
    CONSTRAINT "config_operacion_tiempo_por_objeto_min_check" CHECK (("tiempo_por_objeto_min" >= (0)::numeric))
);


ALTER TABLE "public"."config_operacion" OWNER TO "postgres";


COMMENT ON COLUMN "public"."config_operacion"."tiempo_espera_min" IS 'Reservado: declarado en el documento fuente pero no usado en la fórmula. Confirmar si se suma.';



CREATE TABLE IF NOT EXISTS "public"."config_tarifa" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "tarifa_base" numeric(12,2) NOT NULL,
    "valor_por_km" numeric(10,2) NOT NULL,
    "valor_por_m3" numeric(10,2) NOT NULL,
    "valor_por_hora" numeric(10,2) NOT NULL,
    "recargo_escalera_por_piso" numeric(10,2) NOT NULL,
    "recargo_por_ayudante" numeric(10,2) NOT NULL,
    "vigente_desde" timestamp with time zone DEFAULT "now"() NOT NULL,
    "vigente_hasta" timestamp with time zone,
    CONSTRAINT "chk_vigencia_tarifa" CHECK ((("vigente_hasta" IS NULL) OR ("vigente_hasta" > "vigente_desde")))
);


ALTER TABLE "public"."config_tarifa" OWNER TO "postgres";


COMMENT ON TABLE "public"."config_tarifa" IS 'DEPRECATED: modelo de tarifa plana reemplazado por config_costo_laboral + config_operacion + vehiculo_costo (RN-01). Se elimina en una migración posterior.';



CREATE TABLE IF NOT EXISTS "public"."documento_transportista" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "transportista_id" "uuid" NOT NULL,
    "tipo_documento_codigo" "text" NOT NULL,
    "url_archivo" "text" NOT NULL,
    "estado" "text" DEFAULT 'pendiente'::"text" NOT NULL,
    "motivo_rechazo" "text",
    "revisado_por_admin_id" "uuid",
    "cargado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    "revisado_en" timestamp with time zone,
    CONSTRAINT "documento_transportista_estado_check" CHECK (("estado" = ANY (ARRAY['pendiente'::"text", 'aprobado'::"text", 'rechazado'::"text"])))
);


ALTER TABLE "public"."documento_transportista" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."estado_habilitacion_transportista" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."estado_habilitacion_transportista" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."estado_incidente" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."estado_incidente" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."estado_oferta" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."estado_oferta" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."estado_pago" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."estado_pago" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."estado_solicitud" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."estado_solicitud" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."estado_viaje" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."estado_viaje" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."incidente" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "reportante_usuario_id" "uuid" NOT NULL,
    "viaje_id" "uuid",
    "tipo_incidente_codigo" "text" NOT NULL,
    "asunto" "text" NOT NULL,
    "descripcion" "text" NOT NULL,
    "estado_codigo" "text" DEFAULT 'abierto'::"text" NOT NULL,
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL
);


ALTER TABLE "public"."incidente" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."incidente_resolucion" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "incidente_id" "uuid" NOT NULL,
    "tipo_resolucion_codigo" "text" NOT NULL,
    "monto_afectado" numeric(12,2),
    "veto_id" "uuid",
    "admin_id" "uuid" NOT NULL,
    "resuelto_en" timestamp with time zone DEFAULT "now"() NOT NULL
);


ALTER TABLE "public"."incidente_resolucion" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."mensaje" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "viaje_id" "uuid" NOT NULL,
    "emisor_usuario_id" "uuid" NOT NULL,
    "contenido" "text" NOT NULL,
    "enviado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    "leido_en" timestamp with time zone
);


ALTER TABLE "public"."mensaje" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."notificacion" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "destinatario_usuario_id" "uuid" NOT NULL,
    "tipo_notificacion_codigo" "text" NOT NULL,
    "contenido" "text" NOT NULL,
    "leida" boolean DEFAULT false NOT NULL,
    "entidad_referencia_tipo" "text",
    "entidad_referencia_id" "uuid",
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL
);


ALTER TABLE "public"."notificacion" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."objeto" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "nombre" "text" NOT NULL,
    "peso_estimado_kg" numeric(8,2) NOT NULL,
    "volumen_estimado_m3" numeric(8,3) NOT NULL,
    "largo_m" numeric(6,3),
    "ancho_m" numeric(6,3),
    "alto_m" numeric(6,3),
    "rotacion_horizontal" boolean DEFAULT true NOT NULL,
    "rotacion_vertical" boolean DEFAULT true NOT NULL,
    "apilable" boolean DEFAULT true NOT NULL,
    CONSTRAINT "objeto_alto_m_check" CHECK (("alto_m" > (0)::numeric)),
    CONSTRAINT "objeto_ancho_m_check" CHECK (("ancho_m" > (0)::numeric)),
    CONSTRAINT "objeto_largo_m_check" CHECK (("largo_m" > (0)::numeric))
);


ALTER TABLE "public"."objeto" OWNER TO "postgres";


COMMENT ON COLUMN "public"."objeto"."volumen_estimado_m3" IS 'Redundante con largo_m*ancho_m*alto_m una vez cargadas las dimensiones. Candidata a DEPRECATED.';



COMMENT ON COLUMN "public"."objeto"."rotacion_horizontal" IS 'false = no girar sobre el piso. Junto con rotacion_vertical=false → sin rotación (RotationNever). Sola (con rotacion_vertical=true) es informativa: boxpacker3 v2 no la soporta directo.';



COMMENT ON COLUMN "public"."objeto"."rotacion_vertical" IS 'false = no se puede volcar: alto_m queda vertical (boxpacker3 v2 VerticalAxes=[DepthAxis]).';



COMMENT ON COLUMN "public"."objeto"."apilable" IS 'false = nada encima (boxpacker3 v2 NothingOnTop).';



CREATE TABLE IF NOT EXISTS "public"."oferta" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "solicitud_id" "uuid" NOT NULL,
    "transportista_id" "uuid" NOT NULL,
    "vehiculo_id" "uuid" NOT NULL,
    "cantidad_viajes" integer NOT NULL,
    "cantidad_ayudantes" integer DEFAULT 0 NOT NULL,
    "precio_calculado" numeric(12,2) NOT NULL,
    "estado_codigo" "text" DEFAULT 'pendiente'::"text" NOT NULL,
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    "distancia_km" numeric(8,2) NOT NULL,
    "duracion_ruta_h" numeric(6,2) NOT NULL,
    "duracion_operacion_h" numeric(6,2) NOT NULL,
    "costo_laboral" numeric(12,2) NOT NULL,
    "costo_vehiculo" numeric(12,2) NOT NULL,
    "costos_adicionales" numeric(12,2) DEFAULT 0 NOT NULL,
    "costo_operativo" numeric(12,2) NOT NULL,
    "margen_pct" numeric(5,2) NOT NULL,
    "precio_neto" numeric(12,2) NOT NULL,
    "porcentaje_comision" numeric(5,2) NOT NULL,
    "iva_pct" numeric(5,2) NOT NULL,
    CONSTRAINT "oferta_cantidad_viajes_check" CHECK (("cantidad_viajes" > 0)),
    CONSTRAINT "oferta_costo_laboral_check" CHECK (("costo_laboral" >= (0)::numeric)),
    CONSTRAINT "oferta_costo_operativo_check" CHECK (("costo_operativo" >= (0)::numeric)),
    CONSTRAINT "oferta_costo_vehiculo_check" CHECK (("costo_vehiculo" >= (0)::numeric)),
    CONSTRAINT "oferta_costos_adicionales_check" CHECK (("costos_adicionales" >= (0)::numeric)),
    CONSTRAINT "oferta_distancia_km_check" CHECK (("distancia_km" >= (0)::numeric)),
    CONSTRAINT "oferta_duracion_operacion_h_check" CHECK (("duracion_operacion_h" >= (0)::numeric)),
    CONSTRAINT "oferta_duracion_ruta_h_check" CHECK (("duracion_ruta_h" >= (0)::numeric)),
    CONSTRAINT "oferta_iva_pct_check" CHECK ((("iva_pct" >= (0)::numeric) AND ("iva_pct" <= (100)::numeric))),
    CONSTRAINT "oferta_margen_pct_check" CHECK (("margen_pct" >= (0)::numeric)),
    CONSTRAINT "oferta_porcentaje_comision_check" CHECK ((("porcentaje_comision" >= (0)::numeric) AND ("porcentaje_comision" <= (100)::numeric))),
    CONSTRAINT "oferta_precio_neto_check" CHECK (("precio_neto" >= (0)::numeric))
);


ALTER TABLE "public"."oferta" OWNER TO "postgres";


COMMENT ON COLUMN "public"."oferta"."precio_calculado" IS 'Precio final al Cliente = precio_neto * (1 + iva_pct/100). Único monto visible para el Cliente.';



COMMENT ON COLUMN "public"."oferta"."duracion_operacion_h" IS 'Suma de duracion_operacion() de todos los viajes de la oferta.';



COMMENT ON COLUMN "public"."oferta"."costo_operativo" IS 'Suma de calcular_costo_viaje() de todos los viajes. Sin margen, comisión ni IVA. No debe exponerse al Cliente por API.';



COMMENT ON COLUMN "public"."oferta"."margen_pct" IS 'Margen APLICADO a esta oferta (valor congelado). De dónde sale (plataforma vs. Transportista) es una decisión abierta.';



CREATE TABLE IF NOT EXISTS "public"."pago" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "viaje_id" "uuid" NOT NULL,
    "monto" numeric(12,2) NOT NULL,
    "comision_monto" numeric(12,2) NOT NULL,
    "estado_codigo" "text" DEFAULT 'pendiente'::"text" NOT NULL,
    "referencia_externa" "text",
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    "actualizado_en" timestamp with time zone DEFAULT "now"() NOT NULL
);


ALTER TABLE "public"."pago" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."pago_movimiento" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "pago_id" "uuid" NOT NULL,
    "tipo" "text" NOT NULL,
    "monto" numeric(12,2) NOT NULL,
    "incidente_origen_id" "uuid",
    "admin_id" "uuid",
    "referencia_externa" "text",
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    CONSTRAINT "pago_movimiento_tipo_check" CHECK (("tipo" = ANY (ARRAY['captura'::"text", 'reembolso_parcial'::"text", 'reembolso_total'::"text", 'liberacion_parcial'::"text", 'liberacion_total'::"text"])))
);


ALTER TABLE "public"."pago_movimiento" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."resena" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "viaje_id" "uuid" NOT NULL,
    "transportista_id" "uuid" NOT NULL,
    "cliente_id" "uuid" NOT NULL,
    "calificacion" integer NOT NULL,
    "mensaje" "text",
    "nombre_cliente_snapshot" "text" NOT NULL,
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    CONSTRAINT "resena_calificacion_check" CHECK ((("calificacion" >= 1) AND ("calificacion" <= 5)))
);


ALTER TABLE "public"."resena" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."solicitud" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "cliente_id" "uuid" NOT NULL,
    "origen_zona_id" "uuid" NOT NULL,
    "destino_zona_id" "uuid" NOT NULL,
    "origen_direccion" "text" NOT NULL,
    "destino_direccion" "text" NOT NULL,
    "origen_lat" numeric(9,6) NOT NULL,
    "origen_lng" numeric(9,6) NOT NULL,
    "destino_lat" numeric(9,6) NOT NULL,
    "destino_lng" numeric(9,6) NOT NULL,
    "requiere_escalera" boolean,
    "pisos_escalera" integer,
    "cantidad_ayudantes_solicitados" integer DEFAULT 0 NOT NULL,
    "estado_codigo" "text" DEFAULT 'publicada'::"text" NOT NULL,
    "cotizacion_estimada_monto" numeric(12,2),
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    "pisos_origen" integer DEFAULT 0 NOT NULL,
    "ascensor_utilizable_origen" boolean DEFAULT false NOT NULL,
    "distancia_vehiculo_origen_m" numeric(6,1) DEFAULT 0 NOT NULL,
    "pisos_destino" integer DEFAULT 0 NOT NULL,
    "ascensor_utilizable_destino" boolean DEFAULT false NOT NULL,
    "distancia_vehiculo_destino_m" numeric(6,1) DEFAULT 0 NOT NULL,
    CONSTRAINT "solicitud_distancia_vehiculo_destino_m_check" CHECK (("distancia_vehiculo_destino_m" >= (0)::numeric)),
    CONSTRAINT "solicitud_distancia_vehiculo_origen_m_check" CHECK (("distancia_vehiculo_origen_m" >= (0)::numeric)),
    CONSTRAINT "solicitud_pisos_destino_check" CHECK (("pisos_destino" >= 0)),
    CONSTRAINT "solicitud_pisos_origen_check" CHECK (("pisos_origen" >= 0))
);


ALTER TABLE "public"."solicitud" OWNER TO "postgres";


COMMENT ON COLUMN "public"."solicitud"."requiere_escalera" IS 'DEPRECATED: reemplazada por pisos_origen/pisos_destino + ascensor_utilizable_*.';



COMMENT ON COLUMN "public"."solicitud"."pisos_escalera" IS 'DEPRECATED: reemplazada por pisos_origen/pisos_destino.';



COMMENT ON COLUMN "public"."solicitud"."cotizacion_estimada_monto" IS 'DEPRECATED: no hay cotización estimada al publicar (decisión de producto). El único precio es oferta.precio_calculado.';



COMMENT ON COLUMN "public"."solicitud"."distancia_vehiculo_origen_m" IS 'Distancia a pie (m) entre el vehículo estacionado y la puerta en origen. NO es la distancia del Transportista al origen (no se calcula ni se cobra).';



CREATE TABLE IF NOT EXISTS "public"."solicitud_objeto" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "solicitud_id" "uuid" NOT NULL,
    "objeto_id" "uuid",
    "nombre_personalizado" "text",
    "cantidad" integer DEFAULT 1 NOT NULL,
    "peso_unitario_kg" numeric(8,2) NOT NULL,
    "volumen_unitario_m3" numeric(8,3),
    "largo_m" numeric(6,3) NOT NULL,
    "ancho_m" numeric(6,3) NOT NULL,
    "alto_m" numeric(6,3) NOT NULL,
    "rotacion_horizontal" boolean DEFAULT true NOT NULL,
    "rotacion_vertical" boolean DEFAULT true NOT NULL,
    "apilable" boolean DEFAULT true NOT NULL,
    CONSTRAINT "chk_objeto_o_personalizado" CHECK (((("objeto_id" IS NOT NULL) AND ("nombre_personalizado" IS NULL)) OR (("objeto_id" IS NULL) AND ("nombre_personalizado" IS NOT NULL)))),
    CONSTRAINT "solicitud_objeto_alto_m_check" CHECK (("alto_m" > (0)::numeric)),
    CONSTRAINT "solicitud_objeto_ancho_m_check" CHECK (("ancho_m" > (0)::numeric)),
    CONSTRAINT "solicitud_objeto_cantidad_check" CHECK (("cantidad" > 0)),
    CONSTRAINT "solicitud_objeto_largo_m_check" CHECK (("largo_m" > (0)::numeric))
);


ALTER TABLE "public"."solicitud_objeto" OWNER TO "postgres";


COMMENT ON COLUMN "public"."solicitud_objeto"."volumen_unitario_m3" IS 'DEPRECATED: derivable de largo_m*ancho_m*alto_m. Se elimina en una migración posterior.';



CREATE TABLE IF NOT EXISTS "public"."tipo_documento" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."tipo_documento" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."tipo_incidente" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."tipo_incidente" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."tipo_notificacion" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."tipo_notificacion" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."tipo_resolucion_incidente" (
    "codigo" "text" NOT NULL,
    "descripcion" "text" NOT NULL
);


ALTER TABLE "public"."tipo_resolucion_incidente" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."tipo_vehiculo" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "nombre" "text" NOT NULL,
    "volumen_estandar_m3" numeric(8,2) NOT NULL,
    "peso_maximo_estandar_kg" numeric(9,2) NOT NULL
);


ALTER TABLE "public"."tipo_vehiculo" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."transportista" (
    "usuario_id" "uuid" NOT NULL,
    "forma_trabajo" "text",
    "disponible" boolean DEFAULT true NOT NULL,
    "estado_habilitacion_codigo" "text" DEFAULT 'pendiente'::"text" NOT NULL,
    "calificacion_promedio" numeric(3,2),
    "tasa_cumplimiento" numeric(5,2),
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL
);


ALTER TABLE "public"."transportista" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."transportista_zona" (
    "transportista_id" "uuid" NOT NULL,
    "zona_id" "uuid" NOT NULL
);


ALTER TABLE "public"."transportista_zona" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."usuario" (
    "id" "uuid" NOT NULL,
    "email" "extensions"."citext" NOT NULL,
    "telefono" "text" NOT NULL,
    "nombre_completo" "text" NOT NULL,
    "rol" "text" NOT NULL,
    "activo" boolean DEFAULT true NOT NULL,
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    "actualizado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    CONSTRAINT "usuario_rol_check" CHECK (("rol" = ANY (ARRAY['cliente'::"text", 'transportista'::"text", 'administrador'::"text"])))
);


ALTER TABLE "public"."usuario" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."vehiculo" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "transportista_id" "uuid" NOT NULL,
    "tipo_vehiculo_id" "uuid" NOT NULL,
    "patente" "text" NOT NULL,
    "marca" "text",
    "modelo" "text",
    "volumen_carga_m3" numeric(8,2),
    "peso_maximo_kg" numeric(9,2) NOT NULL,
    "activo" boolean DEFAULT true NOT NULL,
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    "largo_util_m" numeric(5,2) NOT NULL,
    "ancho_util_m" numeric(5,2) NOT NULL,
    "alto_util_m" numeric(5,2) NOT NULL,
    CONSTRAINT "vehiculo_alto_util_m_check" CHECK (("alto_util_m" > (0)::numeric)),
    CONSTRAINT "vehiculo_ancho_util_m_check" CHECK (("ancho_util_m" > (0)::numeric)),
    CONSTRAINT "vehiculo_largo_util_m_check" CHECK (("largo_util_m" > (0)::numeric))
);


ALTER TABLE "public"."vehiculo" OWNER TO "postgres";


COMMENT ON COLUMN "public"."vehiculo"."volumen_carga_m3" IS 'DEPRECATED: derivable de largo_util_m*ancho_util_m*alto_util_m. Se elimina en una migración posterior.';



COMMENT ON COLUMN "public"."vehiculo"."peso_maximo_kg" IS 'Carga útil máxima en kg (NO peso bruto total). Se usa directo como capacidad de peso en el empaquetado.';



CREATE TABLE IF NOT EXISTS "public"."vehiculo_costo" (
    "vehiculo_id" "uuid" NOT NULL,
    "combustible_precio_l" numeric(10,2) NOT NULL,
    "rendimiento_km_l" numeric(6,2) NOT NULL,
    "cantidad_neumaticos" integer NOT NULL,
    "costo_neumatico" numeric(12,2) NOT NULL,
    "vida_neumatico_km" numeric(10,0) NOT NULL,
    "costo_mantenimiento_km" numeric(10,2) NOT NULL,
    "valor_compra" numeric(14,2) NOT NULL,
    "valor_residual" numeric(14,2) NOT NULL,
    "vida_util_km" numeric(10,0) NOT NULL,
    "seguro_mensual" numeric(12,2) NOT NULL,
    "patente_mensual" numeric(12,2) NOT NULL,
    "actualizado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    CONSTRAINT "chk_vehiculo_costo_residual" CHECK (("valor_residual" <= "valor_compra")),
    CONSTRAINT "vehiculo_costo_cantidad_neumaticos_check" CHECK (("cantidad_neumaticos" > 0)),
    CONSTRAINT "vehiculo_costo_combustible_precio_l_check" CHECK (("combustible_precio_l" >= (0)::numeric)),
    CONSTRAINT "vehiculo_costo_costo_mantenimiento_km_check" CHECK (("costo_mantenimiento_km" >= (0)::numeric)),
    CONSTRAINT "vehiculo_costo_costo_neumatico_check" CHECK (("costo_neumatico" >= (0)::numeric)),
    CONSTRAINT "vehiculo_costo_patente_mensual_check" CHECK (("patente_mensual" >= (0)::numeric)),
    CONSTRAINT "vehiculo_costo_rendimiento_km_l_check" CHECK (("rendimiento_km_l" > (0)::numeric)),
    CONSTRAINT "vehiculo_costo_seguro_mensual_check" CHECK (("seguro_mensual" >= (0)::numeric)),
    CONSTRAINT "vehiculo_costo_valor_compra_check" CHECK (("valor_compra" >= (0)::numeric)),
    CONSTRAINT "vehiculo_costo_valor_residual_check" CHECK (("valor_residual" >= (0)::numeric)),
    CONSTRAINT "vehiculo_costo_vida_neumatico_km_check" CHECK (("vida_neumatico_km" > (0)::numeric)),
    CONSTRAINT "vehiculo_costo_vida_util_km_check" CHECK (("vida_util_km" > (0)::numeric))
);


ALTER TABLE "public"."vehiculo_costo" OWNER TO "postgres";


COMMENT ON TABLE "public"."vehiculo_costo" IS 'Variables de costo del vehículo real para RN-01. Privada: solo el Transportista dueño y el Administrador.';



COMMENT ON COLUMN "public"."vehiculo_costo"."combustible_precio_l" IS 'Por vehículo (diesel/nafta/GNC); el Transportista lo actualiza manualmente.';



CREATE TABLE IF NOT EXISTS "public"."veto" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "usuario_id" "uuid" NOT NULL,
    "tipo" "text" NOT NULL,
    "fecha_inicio" timestamp with time zone DEFAULT "now"() NOT NULL,
    "fecha_fin" timestamp with time zone,
    "motivo" "text" NOT NULL,
    "incidente_origen_id" "uuid",
    "admin_id" "uuid" NOT NULL,
    "estado" "text" DEFAULT 'activo'::"text" NOT NULL,
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    CONSTRAINT "chk_veto_fecha_fin" CHECK (((("tipo" = 'definitivo'::"text") AND ("fecha_fin" IS NULL)) OR (("tipo" = 'temporal'::"text") AND ("fecha_fin" IS NOT NULL) AND ("fecha_fin" > "fecha_inicio")))),
    CONSTRAINT "veto_estado_check" CHECK (("estado" = ANY (ARRAY['activo'::"text", 'expirado'::"text", 'revocado'::"text"]))),
    CONSTRAINT "veto_tipo_check" CHECK (("tipo" = ANY (ARRAY['temporal'::"text", 'definitivo'::"text"])))
);


ALTER TABLE "public"."veto" OWNER TO "postgres";


CREATE TABLE IF NOT EXISTS "public"."viaje" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "oferta_id" "uuid" NOT NULL,
    "solicitud_id" "uuid" NOT NULL,
    "cliente_id" "uuid" NOT NULL,
    "transportista_id" "uuid" NOT NULL,
    "estado_codigo" "text" DEFAULT 'confirmado'::"text" NOT NULL,
    "origen_direccion_snapshot" "text" NOT NULL,
    "destino_direccion_snapshot" "text" NOT NULL,
    "origen_lat_snapshot" numeric(9,6) NOT NULL,
    "origen_lng_snapshot" numeric(9,6) NOT NULL,
    "destino_lat_snapshot" numeric(9,6) NOT NULL,
    "destino_lng_snapshot" numeric(9,6) NOT NULL,
    "distancia_km_snapshot" numeric(8,2) NOT NULL,
    "transportista_nombre_snapshot" "text" NOT NULL,
    "vehiculo_patente_snapshot" "text" NOT NULL,
    "vehiculo_marca_modelo_snapshot" "text",
    "tarifa_base_snapshot" numeric(12,2),
    "valor_por_km_snapshot" numeric(10,2),
    "valor_por_m3_snapshot" numeric(10,2),
    "valor_por_hora_snapshot" numeric(10,2),
    "recargo_escalera_snapshot" numeric(10,2),
    "recargo_ayudante_snapshot" numeric(10,2),
    "monto_total_snapshot" numeric(12,2) NOT NULL,
    "porcentaje_comision_snapshot" numeric(5,2) NOT NULL,
    "cantidad_ayudantes" integer DEFAULT 0 NOT NULL,
    "pin_inicio" "text" NOT NULL,
    "pin_inicio_validado_en" timestamp with time zone,
    "pin_fin" "text" NOT NULL,
    "pin_fin_validado_en" timestamp with time zone,
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL,
    "finalizado_en" timestamp with time zone,
    "cantidad_viajes_snapshot" integer NOT NULL,
    "duracion_ruta_h_snapshot" numeric(6,2) NOT NULL,
    "duracion_operacion_h_snapshot" numeric(6,2) NOT NULL,
    "costo_laboral_snapshot" numeric(12,2) NOT NULL,
    "costo_vehiculo_snapshot" numeric(12,2) NOT NULL,
    "costos_adicionales_snapshot" numeric(12,2) DEFAULT 0 NOT NULL,
    "costo_operativo_snapshot" numeric(12,2) NOT NULL,
    "margen_pct_snapshot" numeric(5,2) NOT NULL,
    "precio_neto_snapshot" numeric(12,2) NOT NULL,
    "iva_pct_snapshot" numeric(5,2) NOT NULL,
    CONSTRAINT "viaje_cantidad_viajes_snapshot_check" CHECK (("cantidad_viajes_snapshot" > 0))
);


ALTER TABLE "public"."viaje" OWNER TO "postgres";


COMMENT ON COLUMN "public"."viaje"."tarifa_base_snapshot" IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';



COMMENT ON COLUMN "public"."viaje"."valor_por_km_snapshot" IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';



COMMENT ON COLUMN "public"."viaje"."valor_por_m3_snapshot" IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';



COMMENT ON COLUMN "public"."viaje"."valor_por_hora_snapshot" IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';



COMMENT ON COLUMN "public"."viaje"."recargo_escalera_snapshot" IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';



COMMENT ON COLUMN "public"."viaje"."recargo_ayudante_snapshot" IS 'DEPRECATED: modelo de tarifa plana (config_tarifa). Se elimina en una migración posterior.';



COMMENT ON COLUMN "public"."viaje"."monto_total_snapshot" IS 'Precio final pagado por el Cliente (copia de oferta.precio_calculado).';



CREATE TABLE IF NOT EXISTS "public"."viaje_ubicacion" (
    "id" bigint NOT NULL,
    "viaje_id" "uuid" NOT NULL,
    "lat" numeric(9,6) NOT NULL,
    "lng" numeric(9,6) NOT NULL,
    "registrado_en" timestamp with time zone DEFAULT "now"() NOT NULL
);


ALTER TABLE "public"."viaje_ubicacion" OWNER TO "postgres";


ALTER TABLE "public"."viaje_ubicacion" ALTER COLUMN "id" ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME "public"."viaje_ubicacion_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);



CREATE TABLE IF NOT EXISTS "public"."zona" (
    "id" "uuid" DEFAULT "gen_random_uuid"() NOT NULL,
    "nombre" "text" NOT NULL,
    "provincia" "text" NOT NULL,
    "creado_en" timestamp with time zone DEFAULT "now"() NOT NULL
);


ALTER TABLE "public"."zona" OWNER TO "postgres";


ALTER TABLE ONLY "public"."administrador"
    ADD CONSTRAINT "administrador_pkey" PRIMARY KEY ("usuario_id");



ALTER TABLE ONLY "public"."cliente"
    ADD CONSTRAINT "cliente_pkey" PRIMARY KEY ("usuario_id");



ALTER TABLE ONLY "public"."config_comision"
    ADD CONSTRAINT "config_comision_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."config_costo_laboral"
    ADD CONSTRAINT "config_costo_laboral_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."config_impuesto"
    ADD CONSTRAINT "config_impuesto_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."config_operacion"
    ADD CONSTRAINT "config_operacion_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."config_tarifa"
    ADD CONSTRAINT "config_tarifa_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."documento_transportista"
    ADD CONSTRAINT "documento_transportista_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."estado_habilitacion_transportista"
    ADD CONSTRAINT "estado_habilitacion_transportista_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."estado_incidente"
    ADD CONSTRAINT "estado_incidente_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."estado_oferta"
    ADD CONSTRAINT "estado_oferta_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."estado_pago"
    ADD CONSTRAINT "estado_pago_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."estado_solicitud"
    ADD CONSTRAINT "estado_solicitud_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."estado_viaje"
    ADD CONSTRAINT "estado_viaje_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."incidente"
    ADD CONSTRAINT "incidente_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."incidente_resolucion"
    ADD CONSTRAINT "incidente_resolucion_incidente_id_key" UNIQUE ("incidente_id");



ALTER TABLE ONLY "public"."incidente_resolucion"
    ADD CONSTRAINT "incidente_resolucion_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."mensaje"
    ADD CONSTRAINT "mensaje_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."notificacion"
    ADD CONSTRAINT "notificacion_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."objeto"
    ADD CONSTRAINT "objeto_nombre_key" UNIQUE ("nombre");



ALTER TABLE ONLY "public"."objeto"
    ADD CONSTRAINT "objeto_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."oferta"
    ADD CONSTRAINT "oferta_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."oferta"
    ADD CONSTRAINT "oferta_solicitud_id_transportista_id_vehiculo_id_key" UNIQUE ("solicitud_id", "transportista_id", "vehiculo_id");



ALTER TABLE ONLY "public"."pago_movimiento"
    ADD CONSTRAINT "pago_movimiento_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."pago"
    ADD CONSTRAINT "pago_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."pago"
    ADD CONSTRAINT "pago_viaje_id_key" UNIQUE ("viaje_id");



ALTER TABLE ONLY "public"."resena"
    ADD CONSTRAINT "resena_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."resena"
    ADD CONSTRAINT "resena_viaje_id_key" UNIQUE ("viaje_id");



ALTER TABLE ONLY "public"."solicitud_objeto"
    ADD CONSTRAINT "solicitud_objeto_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."solicitud"
    ADD CONSTRAINT "solicitud_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."tipo_documento"
    ADD CONSTRAINT "tipo_documento_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."tipo_incidente"
    ADD CONSTRAINT "tipo_incidente_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."tipo_notificacion"
    ADD CONSTRAINT "tipo_notificacion_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."tipo_resolucion_incidente"
    ADD CONSTRAINT "tipo_resolucion_incidente_pkey" PRIMARY KEY ("codigo");



ALTER TABLE ONLY "public"."tipo_vehiculo"
    ADD CONSTRAINT "tipo_vehiculo_nombre_key" UNIQUE ("nombre");



ALTER TABLE ONLY "public"."tipo_vehiculo"
    ADD CONSTRAINT "tipo_vehiculo_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."transportista"
    ADD CONSTRAINT "transportista_pkey" PRIMARY KEY ("usuario_id");



ALTER TABLE ONLY "public"."transportista_zona"
    ADD CONSTRAINT "transportista_zona_pkey" PRIMARY KEY ("transportista_id", "zona_id");



ALTER TABLE ONLY "public"."usuario"
    ADD CONSTRAINT "usuario_email_key" UNIQUE ("email");



ALTER TABLE ONLY "public"."usuario"
    ADD CONSTRAINT "usuario_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."vehiculo_costo"
    ADD CONSTRAINT "vehiculo_costo_pkey" PRIMARY KEY ("vehiculo_id");



ALTER TABLE ONLY "public"."vehiculo"
    ADD CONSTRAINT "vehiculo_patente_key" UNIQUE ("patente");



ALTER TABLE ONLY "public"."vehiculo"
    ADD CONSTRAINT "vehiculo_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."veto"
    ADD CONSTRAINT "veto_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."viaje"
    ADD CONSTRAINT "viaje_oferta_id_key" UNIQUE ("oferta_id");



ALTER TABLE ONLY "public"."viaje"
    ADD CONSTRAINT "viaje_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."viaje_ubicacion"
    ADD CONSTRAINT "viaje_ubicacion_pkey" PRIMARY KEY ("id");



ALTER TABLE ONLY "public"."zona"
    ADD CONSTRAINT "zona_nombre_provincia_key" UNIQUE ("nombre", "provincia");



ALTER TABLE ONLY "public"."zona"
    ADD CONSTRAINT "zona_pkey" PRIMARY KEY ("id");



CREATE UNIQUE INDEX "idx_config_comision_vigente" ON "public"."config_comision" USING "btree" ("vigente_hasta") WHERE ("vigente_hasta" IS NULL);



CREATE UNIQUE INDEX "idx_config_costo_laboral_vigente" ON "public"."config_costo_laboral" USING "btree" ("vigente_hasta") WHERE ("vigente_hasta" IS NULL);



CREATE UNIQUE INDEX "idx_config_impuesto_vigente" ON "public"."config_impuesto" USING "btree" ("vigente_hasta") WHERE ("vigente_hasta" IS NULL);



CREATE UNIQUE INDEX "idx_config_operacion_vigente" ON "public"."config_operacion" USING "btree" ("vigente_hasta") WHERE ("vigente_hasta" IS NULL);



CREATE UNIQUE INDEX "idx_config_tarifa_vigente" ON "public"."config_tarifa" USING "btree" ("vigente_hasta") WHERE ("vigente_hasta" IS NULL);



CREATE INDEX "idx_documento_transportista_admin" ON "public"."documento_transportista" USING "btree" ("revisado_por_admin_id");



CREATE INDEX "idx_documento_transportista_tipo" ON "public"."documento_transportista" USING "btree" ("tipo_documento_codigo");



CREATE INDEX "idx_documento_transportista_transportista" ON "public"."documento_transportista" USING "btree" ("transportista_id");



CREATE INDEX "idx_incidente_estado" ON "public"."incidente" USING "btree" ("estado_codigo");



CREATE INDEX "idx_incidente_reportante" ON "public"."incidente" USING "btree" ("reportante_usuario_id");



CREATE INDEX "idx_incidente_resolucion_admin" ON "public"."incidente_resolucion" USING "btree" ("admin_id");



CREATE INDEX "idx_incidente_resolucion_tipo" ON "public"."incidente_resolucion" USING "btree" ("tipo_resolucion_codigo");



CREATE INDEX "idx_incidente_resolucion_veto" ON "public"."incidente_resolucion" USING "btree" ("veto_id");



CREATE INDEX "idx_incidente_tipo" ON "public"."incidente" USING "btree" ("tipo_incidente_codigo");



CREATE INDEX "idx_incidente_viaje" ON "public"."incidente" USING "btree" ("viaje_id");



CREATE INDEX "idx_mensaje_emisor" ON "public"."mensaje" USING "btree" ("emisor_usuario_id");



CREATE INDEX "idx_mensaje_viaje" ON "public"."mensaje" USING "btree" ("viaje_id", "enviado_en");



CREATE INDEX "idx_notificacion_destinatario" ON "public"."notificacion" USING "btree" ("destinatario_usuario_id", "leida");



CREATE INDEX "idx_notificacion_tipo" ON "public"."notificacion" USING "btree" ("tipo_notificacion_codigo");



CREATE INDEX "idx_oferta_estado" ON "public"."oferta" USING "btree" ("estado_codigo");



CREATE INDEX "idx_oferta_solicitud" ON "public"."oferta" USING "btree" ("solicitud_id");



CREATE INDEX "idx_oferta_transportista" ON "public"."oferta" USING "btree" ("transportista_id");



CREATE INDEX "idx_oferta_vehiculo" ON "public"."oferta" USING "btree" ("vehiculo_id");



CREATE INDEX "idx_pago_estado" ON "public"."pago" USING "btree" ("estado_codigo");



CREATE INDEX "idx_pago_movimiento_admin" ON "public"."pago_movimiento" USING "btree" ("admin_id");



CREATE INDEX "idx_pago_movimiento_incidente" ON "public"."pago_movimiento" USING "btree" ("incidente_origen_id");



CREATE INDEX "idx_pago_movimiento_pago" ON "public"."pago_movimiento" USING "btree" ("pago_id");



CREATE INDEX "idx_resena_cliente" ON "public"."resena" USING "btree" ("cliente_id");



CREATE INDEX "idx_resena_transportista" ON "public"."resena" USING "btree" ("transportista_id");



CREATE INDEX "idx_solicitud_cliente" ON "public"."solicitud" USING "btree" ("cliente_id");



CREATE INDEX "idx_solicitud_destino_zona" ON "public"."solicitud" USING "btree" ("destino_zona_id");



CREATE INDEX "idx_solicitud_estado" ON "public"."solicitud" USING "btree" ("estado_codigo");



CREATE INDEX "idx_solicitud_objeto_objeto" ON "public"."solicitud_objeto" USING "btree" ("objeto_id");



CREATE INDEX "idx_solicitud_objeto_solicitud" ON "public"."solicitud_objeto" USING "btree" ("solicitud_id");



CREATE INDEX "idx_solicitud_origen_zona" ON "public"."solicitud" USING "btree" ("origen_zona_id");



CREATE INDEX "idx_transportista_estado_habilitacion" ON "public"."transportista" USING "btree" ("estado_habilitacion_codigo");



CREATE INDEX "idx_transportista_zona_zona" ON "public"."transportista_zona" USING "btree" ("zona_id");



CREATE INDEX "idx_vehiculo_tipo" ON "public"."vehiculo" USING "btree" ("tipo_vehiculo_id");



CREATE INDEX "idx_vehiculo_transportista" ON "public"."vehiculo" USING "btree" ("transportista_id");



CREATE INDEX "idx_veto_admin" ON "public"."veto" USING "btree" ("admin_id");



CREATE INDEX "idx_veto_incidente_origen" ON "public"."veto" USING "btree" ("incidente_origen_id");



CREATE INDEX "idx_veto_usuario_activo" ON "public"."veto" USING "btree" ("usuario_id") WHERE ("estado" = 'activo'::"text");



CREATE INDEX "idx_viaje_cliente" ON "public"."viaje" USING "btree" ("cliente_id");



CREATE INDEX "idx_viaje_estado" ON "public"."viaje" USING "btree" ("estado_codigo");



CREATE INDEX "idx_viaje_solicitud" ON "public"."viaje" USING "btree" ("solicitud_id");



CREATE INDEX "idx_viaje_transportista" ON "public"."viaje" USING "btree" ("transportista_id");



CREATE INDEX "idx_viaje_ubicacion_viaje_tiempo" ON "public"."viaje_ubicacion" USING "btree" ("viaje_id", "registrado_en" DESC);



CREATE OR REPLACE TRIGGER "trg_proteger_campos_transportista" BEFORE UPDATE ON "public"."transportista" FOR EACH ROW EXECUTE FUNCTION "public"."fn_proteger_campos_transportista"();



CREATE OR REPLACE TRIGGER "trg_proteger_campos_viaje" BEFORE UPDATE ON "public"."viaje" FOR EACH ROW EXECUTE FUNCTION "public"."fn_proteger_campos_viaje"();



ALTER TABLE ONLY "public"."administrador"
    ADD CONSTRAINT "administrador_usuario_id_fkey" FOREIGN KEY ("usuario_id") REFERENCES "public"."usuario"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."cliente"
    ADD CONSTRAINT "cliente_usuario_id_fkey" FOREIGN KEY ("usuario_id") REFERENCES "public"."usuario"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."documento_transportista"
    ADD CONSTRAINT "documento_transportista_revisado_por_admin_id_fkey" FOREIGN KEY ("revisado_por_admin_id") REFERENCES "public"."administrador"("usuario_id");



ALTER TABLE ONLY "public"."documento_transportista"
    ADD CONSTRAINT "documento_transportista_tipo_documento_codigo_fkey" FOREIGN KEY ("tipo_documento_codigo") REFERENCES "public"."tipo_documento"("codigo");



ALTER TABLE ONLY "public"."documento_transportista"
    ADD CONSTRAINT "documento_transportista_transportista_id_fkey" FOREIGN KEY ("transportista_id") REFERENCES "public"."transportista"("usuario_id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."pago_movimiento"
    ADD CONSTRAINT "fk_pago_movimiento_incidente" FOREIGN KEY ("incidente_origen_id") REFERENCES "public"."incidente"("id");



ALTER TABLE ONLY "public"."incidente"
    ADD CONSTRAINT "incidente_estado_codigo_fkey" FOREIGN KEY ("estado_codigo") REFERENCES "public"."estado_incidente"("codigo");



ALTER TABLE ONLY "public"."incidente"
    ADD CONSTRAINT "incidente_reportante_usuario_id_fkey" FOREIGN KEY ("reportante_usuario_id") REFERENCES "public"."usuario"("id");



ALTER TABLE ONLY "public"."incidente_resolucion"
    ADD CONSTRAINT "incidente_resolucion_admin_id_fkey" FOREIGN KEY ("admin_id") REFERENCES "public"."administrador"("usuario_id");



ALTER TABLE ONLY "public"."incidente_resolucion"
    ADD CONSTRAINT "incidente_resolucion_incidente_id_fkey" FOREIGN KEY ("incidente_id") REFERENCES "public"."incidente"("id");



ALTER TABLE ONLY "public"."incidente_resolucion"
    ADD CONSTRAINT "incidente_resolucion_tipo_resolucion_codigo_fkey" FOREIGN KEY ("tipo_resolucion_codigo") REFERENCES "public"."tipo_resolucion_incidente"("codigo");



ALTER TABLE ONLY "public"."incidente_resolucion"
    ADD CONSTRAINT "incidente_resolucion_veto_id_fkey" FOREIGN KEY ("veto_id") REFERENCES "public"."veto"("id");



ALTER TABLE ONLY "public"."incidente"
    ADD CONSTRAINT "incidente_tipo_incidente_codigo_fkey" FOREIGN KEY ("tipo_incidente_codigo") REFERENCES "public"."tipo_incidente"("codigo");



ALTER TABLE ONLY "public"."incidente"
    ADD CONSTRAINT "incidente_viaje_id_fkey" FOREIGN KEY ("viaje_id") REFERENCES "public"."viaje"("id");



ALTER TABLE ONLY "public"."mensaje"
    ADD CONSTRAINT "mensaje_emisor_usuario_id_fkey" FOREIGN KEY ("emisor_usuario_id") REFERENCES "public"."usuario"("id");



ALTER TABLE ONLY "public"."mensaje"
    ADD CONSTRAINT "mensaje_viaje_id_fkey" FOREIGN KEY ("viaje_id") REFERENCES "public"."viaje"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."notificacion"
    ADD CONSTRAINT "notificacion_destinatario_usuario_id_fkey" FOREIGN KEY ("destinatario_usuario_id") REFERENCES "public"."usuario"("id");



ALTER TABLE ONLY "public"."notificacion"
    ADD CONSTRAINT "notificacion_tipo_notificacion_codigo_fkey" FOREIGN KEY ("tipo_notificacion_codigo") REFERENCES "public"."tipo_notificacion"("codigo");



ALTER TABLE ONLY "public"."oferta"
    ADD CONSTRAINT "oferta_estado_codigo_fkey" FOREIGN KEY ("estado_codigo") REFERENCES "public"."estado_oferta"("codigo");



ALTER TABLE ONLY "public"."oferta"
    ADD CONSTRAINT "oferta_solicitud_id_fkey" FOREIGN KEY ("solicitud_id") REFERENCES "public"."solicitud"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."oferta"
    ADD CONSTRAINT "oferta_transportista_id_fkey" FOREIGN KEY ("transportista_id") REFERENCES "public"."transportista"("usuario_id");



ALTER TABLE ONLY "public"."oferta"
    ADD CONSTRAINT "oferta_vehiculo_id_fkey" FOREIGN KEY ("vehiculo_id") REFERENCES "public"."vehiculo"("id");



ALTER TABLE ONLY "public"."pago"
    ADD CONSTRAINT "pago_estado_codigo_fkey" FOREIGN KEY ("estado_codigo") REFERENCES "public"."estado_pago"("codigo");



ALTER TABLE ONLY "public"."pago_movimiento"
    ADD CONSTRAINT "pago_movimiento_admin_id_fkey" FOREIGN KEY ("admin_id") REFERENCES "public"."administrador"("usuario_id");



ALTER TABLE ONLY "public"."pago_movimiento"
    ADD CONSTRAINT "pago_movimiento_pago_id_fkey" FOREIGN KEY ("pago_id") REFERENCES "public"."pago"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."pago"
    ADD CONSTRAINT "pago_viaje_id_fkey" FOREIGN KEY ("viaje_id") REFERENCES "public"."viaje"("id");



ALTER TABLE ONLY "public"."resena"
    ADD CONSTRAINT "resena_cliente_id_fkey" FOREIGN KEY ("cliente_id") REFERENCES "public"."cliente"("usuario_id");



ALTER TABLE ONLY "public"."resena"
    ADD CONSTRAINT "resena_transportista_id_fkey" FOREIGN KEY ("transportista_id") REFERENCES "public"."transportista"("usuario_id");



ALTER TABLE ONLY "public"."resena"
    ADD CONSTRAINT "resena_viaje_id_fkey" FOREIGN KEY ("viaje_id") REFERENCES "public"."viaje"("id");



ALTER TABLE ONLY "public"."solicitud"
    ADD CONSTRAINT "solicitud_cliente_id_fkey" FOREIGN KEY ("cliente_id") REFERENCES "public"."cliente"("usuario_id");



ALTER TABLE ONLY "public"."solicitud"
    ADD CONSTRAINT "solicitud_destino_zona_id_fkey" FOREIGN KEY ("destino_zona_id") REFERENCES "public"."zona"("id");



ALTER TABLE ONLY "public"."solicitud"
    ADD CONSTRAINT "solicitud_estado_codigo_fkey" FOREIGN KEY ("estado_codigo") REFERENCES "public"."estado_solicitud"("codigo");



ALTER TABLE ONLY "public"."solicitud_objeto"
    ADD CONSTRAINT "solicitud_objeto_objeto_id_fkey" FOREIGN KEY ("objeto_id") REFERENCES "public"."objeto"("id");



ALTER TABLE ONLY "public"."solicitud_objeto"
    ADD CONSTRAINT "solicitud_objeto_solicitud_id_fkey" FOREIGN KEY ("solicitud_id") REFERENCES "public"."solicitud"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."solicitud"
    ADD CONSTRAINT "solicitud_origen_zona_id_fkey" FOREIGN KEY ("origen_zona_id") REFERENCES "public"."zona"("id");



ALTER TABLE ONLY "public"."transportista"
    ADD CONSTRAINT "transportista_estado_habilitacion_codigo_fkey" FOREIGN KEY ("estado_habilitacion_codigo") REFERENCES "public"."estado_habilitacion_transportista"("codigo");



ALTER TABLE ONLY "public"."transportista"
    ADD CONSTRAINT "transportista_usuario_id_fkey" FOREIGN KEY ("usuario_id") REFERENCES "public"."usuario"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."transportista_zona"
    ADD CONSTRAINT "transportista_zona_transportista_id_fkey" FOREIGN KEY ("transportista_id") REFERENCES "public"."transportista"("usuario_id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."transportista_zona"
    ADD CONSTRAINT "transportista_zona_zona_id_fkey" FOREIGN KEY ("zona_id") REFERENCES "public"."zona"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."usuario"
    ADD CONSTRAINT "usuario_id_fkey" FOREIGN KEY ("id") REFERENCES "auth"."users"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."vehiculo_costo"
    ADD CONSTRAINT "vehiculo_costo_vehiculo_id_fkey" FOREIGN KEY ("vehiculo_id") REFERENCES "public"."vehiculo"("id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."vehiculo"
    ADD CONSTRAINT "vehiculo_tipo_vehiculo_id_fkey" FOREIGN KEY ("tipo_vehiculo_id") REFERENCES "public"."tipo_vehiculo"("id");



ALTER TABLE ONLY "public"."vehiculo"
    ADD CONSTRAINT "vehiculo_transportista_id_fkey" FOREIGN KEY ("transportista_id") REFERENCES "public"."transportista"("usuario_id") ON DELETE CASCADE;



ALTER TABLE ONLY "public"."veto"
    ADD CONSTRAINT "veto_admin_id_fkey" FOREIGN KEY ("admin_id") REFERENCES "public"."administrador"("usuario_id");



ALTER TABLE ONLY "public"."veto"
    ADD CONSTRAINT "veto_incidente_origen_id_fkey" FOREIGN KEY ("incidente_origen_id") REFERENCES "public"."incidente"("id");



ALTER TABLE ONLY "public"."veto"
    ADD CONSTRAINT "veto_usuario_id_fkey" FOREIGN KEY ("usuario_id") REFERENCES "public"."usuario"("id");



ALTER TABLE ONLY "public"."viaje"
    ADD CONSTRAINT "viaje_cliente_id_fkey" FOREIGN KEY ("cliente_id") REFERENCES "public"."cliente"("usuario_id");



ALTER TABLE ONLY "public"."viaje"
    ADD CONSTRAINT "viaje_estado_codigo_fkey" FOREIGN KEY ("estado_codigo") REFERENCES "public"."estado_viaje"("codigo");



ALTER TABLE ONLY "public"."viaje"
    ADD CONSTRAINT "viaje_oferta_id_fkey" FOREIGN KEY ("oferta_id") REFERENCES "public"."oferta"("id");



ALTER TABLE ONLY "public"."viaje"
    ADD CONSTRAINT "viaje_solicitud_id_fkey" FOREIGN KEY ("solicitud_id") REFERENCES "public"."solicitud"("id");



ALTER TABLE ONLY "public"."viaje"
    ADD CONSTRAINT "viaje_transportista_id_fkey" FOREIGN KEY ("transportista_id") REFERENCES "public"."transportista"("usuario_id");



ALTER TABLE ONLY "public"."viaje_ubicacion"
    ADD CONSTRAINT "viaje_ubicacion_viaje_id_fkey" FOREIGN KEY ("viaje_id") REFERENCES "public"."viaje"("id") ON DELETE CASCADE;



ALTER TABLE "public"."administrador" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "administrador_insert" ON "public"."administrador" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "administrador_select" ON "public"."administrador" FOR SELECT USING ((("usuario_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



ALTER TABLE "public"."cliente" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "cliente_insert" ON "public"."cliente" FOR INSERT WITH CHECK (("usuario_id" = ( SELECT "auth"."uid"() AS "uid")));



CREATE POLICY "cliente_select" ON "public"."cliente" FOR SELECT USING ((("usuario_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM ("public"."oferta" "o"
     JOIN "public"."solicitud" "s" ON (("s"."id" = "o"."solicitud_id")))
  WHERE (("o"."transportista_id" = ( SELECT "auth"."uid"() AS "uid")) AND ("s"."cliente_id" = "cliente"."usuario_id"))))));



ALTER TABLE "public"."config_comision" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "config_comision_delete" ON "public"."config_comision" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "config_comision_insert" ON "public"."config_comision" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "config_comision_select" ON "public"."config_comision" FOR SELECT USING (true);



CREATE POLICY "config_comision_update" ON "public"."config_comision" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."config_costo_laboral" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "config_costo_laboral_delete" ON "public"."config_costo_laboral" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "config_costo_laboral_insert" ON "public"."config_costo_laboral" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "config_costo_laboral_select" ON "public"."config_costo_laboral" FOR SELECT USING (true);



CREATE POLICY "config_costo_laboral_update" ON "public"."config_costo_laboral" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."config_impuesto" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "config_impuesto_delete" ON "public"."config_impuesto" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "config_impuesto_insert" ON "public"."config_impuesto" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "config_impuesto_select" ON "public"."config_impuesto" FOR SELECT USING (true);



CREATE POLICY "config_impuesto_update" ON "public"."config_impuesto" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."config_operacion" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "config_operacion_delete" ON "public"."config_operacion" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "config_operacion_insert" ON "public"."config_operacion" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "config_operacion_select" ON "public"."config_operacion" FOR SELECT USING (true);



CREATE POLICY "config_operacion_update" ON "public"."config_operacion" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."config_tarifa" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "config_tarifa_admin" ON "public"."config_tarifa" USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."documento_transportista" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "documento_transportista_insert" ON "public"."documento_transportista" FOR INSERT WITH CHECK (("transportista_id" = ( SELECT "auth"."uid"() AS "uid")));



CREATE POLICY "documento_transportista_select" ON "public"."documento_transportista" FOR SELECT USING ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



CREATE POLICY "documento_transportista_update" ON "public"."documento_transportista" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."estado_habilitacion_transportista" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "estado_habilitacion_transportista_delete" ON "public"."estado_habilitacion_transportista" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "estado_habilitacion_transportista_insert" ON "public"."estado_habilitacion_transportista" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "estado_habilitacion_transportista_select" ON "public"."estado_habilitacion_transportista" FOR SELECT USING (true);



CREATE POLICY "estado_habilitacion_transportista_update" ON "public"."estado_habilitacion_transportista" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."estado_incidente" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "estado_incidente_delete" ON "public"."estado_incidente" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "estado_incidente_insert" ON "public"."estado_incidente" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "estado_incidente_select" ON "public"."estado_incidente" FOR SELECT USING (true);



CREATE POLICY "estado_incidente_update" ON "public"."estado_incidente" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."estado_oferta" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "estado_oferta_delete" ON "public"."estado_oferta" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "estado_oferta_insert" ON "public"."estado_oferta" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "estado_oferta_select" ON "public"."estado_oferta" FOR SELECT USING (true);



CREATE POLICY "estado_oferta_update" ON "public"."estado_oferta" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."estado_pago" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "estado_pago_delete" ON "public"."estado_pago" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "estado_pago_insert" ON "public"."estado_pago" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "estado_pago_select" ON "public"."estado_pago" FOR SELECT USING (true);



CREATE POLICY "estado_pago_update" ON "public"."estado_pago" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."estado_solicitud" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "estado_solicitud_delete" ON "public"."estado_solicitud" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "estado_solicitud_insert" ON "public"."estado_solicitud" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "estado_solicitud_select" ON "public"."estado_solicitud" FOR SELECT USING (true);



CREATE POLICY "estado_solicitud_update" ON "public"."estado_solicitud" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."estado_viaje" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "estado_viaje_delete" ON "public"."estado_viaje" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "estado_viaje_insert" ON "public"."estado_viaje" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "estado_viaje_select" ON "public"."estado_viaje" FOR SELECT USING (true);



CREATE POLICY "estado_viaje_update" ON "public"."estado_viaje" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."incidente" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "incidente_insert" ON "public"."incidente" FOR INSERT WITH CHECK (("reportante_usuario_id" = ( SELECT "auth"."uid"() AS "uid")));



ALTER TABLE "public"."incidente_resolucion" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "incidente_resolucion_insert" ON "public"."incidente_resolucion" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "incidente_resolucion_select" ON "public"."incidente_resolucion" FOR SELECT USING (("public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."incidente" "i"
  WHERE (("i"."id" = "incidente_resolucion"."incidente_id") AND ("i"."reportante_usuario_id" = ( SELECT "auth"."uid"() AS "uid")))))));



CREATE POLICY "incidente_select" ON "public"."incidente" FOR SELECT USING ((("reportante_usuario_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



CREATE POLICY "incidente_update" ON "public"."incidente" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."mensaje" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "mensaje_insert" ON "public"."mensaje" FOR INSERT WITH CHECK ((("emisor_usuario_id" = ( SELECT "auth"."uid"() AS "uid")) AND (EXISTS ( SELECT 1
   FROM "public"."viaje" "v"
  WHERE (("v"."id" = "mensaje"."viaje_id") AND (("v"."cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid"))))))));



CREATE POLICY "mensaje_select" ON "public"."mensaje" FOR SELECT USING (("public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."viaje" "v"
  WHERE (("v"."id" = "mensaje"."viaje_id") AND (("v"."cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid"))))))));



CREATE POLICY "mensaje_update" ON "public"."mensaje" FOR UPDATE USING ((EXISTS ( SELECT 1
   FROM "public"."viaje" "v"
  WHERE (("v"."id" = "mensaje"."viaje_id") AND (("v"."cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid")))))));



ALTER TABLE "public"."notificacion" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "notificacion_admin_insert" ON "public"."notificacion" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "notificacion_select" ON "public"."notificacion" FOR SELECT USING ((("destinatario_usuario_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



CREATE POLICY "notificacion_update" ON "public"."notificacion" FOR UPDATE USING ((("destinatario_usuario_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"())) WITH CHECK ((("destinatario_usuario_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



ALTER TABLE "public"."objeto" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "objeto_delete" ON "public"."objeto" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "objeto_insert" ON "public"."objeto" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "objeto_select" ON "public"."objeto" FOR SELECT USING (true);



CREATE POLICY "objeto_update" ON "public"."objeto" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."oferta" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "oferta_insert" ON "public"."oferta" FOR INSERT WITH CHECK ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) AND "public"."fn_es_transportista_habilitado"()));



CREATE POLICY "oferta_select" ON "public"."oferta" FOR SELECT USING ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."solicitud" "s"
  WHERE (("s"."id" = "oferta"."solicitud_id") AND ("s"."cliente_id" = ( SELECT "auth"."uid"() AS "uid")))))));



CREATE POLICY "oferta_update" ON "public"."oferta" FOR UPDATE USING ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."solicitud" "s"
  WHERE (("s"."id" = "oferta"."solicitud_id") AND ("s"."cliente_id" = ( SELECT "auth"."uid"() AS "uid"))))))) WITH CHECK ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."solicitud" "s"
  WHERE (("s"."id" = "oferta"."solicitud_id") AND ("s"."cliente_id" = ( SELECT "auth"."uid"() AS "uid")))))));



ALTER TABLE "public"."pago" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "pago_insert" ON "public"."pago" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."pago_movimiento" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "pago_movimiento_insert" ON "public"."pago_movimiento" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "pago_movimiento_select" ON "public"."pago_movimiento" FOR SELECT USING (("public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM ("public"."pago" "p"
     JOIN "public"."viaje" "v" ON (("v"."id" = "p"."viaje_id")))
  WHERE (("p"."id" = "pago_movimiento"."pago_id") AND (("v"."cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid"))))))));



CREATE POLICY "pago_select" ON "public"."pago" FOR SELECT USING (("public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."viaje" "v"
  WHERE (("v"."id" = "pago"."viaje_id") AND (("v"."cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid"))))))));



CREATE POLICY "pago_update" ON "public"."pago" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."resena" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "resena_insert" ON "public"."resena" FOR INSERT WITH CHECK ((("cliente_id" = ( SELECT "auth"."uid"() AS "uid")) AND (EXISTS ( SELECT 1
   FROM "public"."viaje" "v"
  WHERE (("v"."id" = "resena"."viaje_id") AND ("v"."cliente_id" = ( SELECT "auth"."uid"() AS "uid")) AND ("v"."estado_codigo" = 'finalizado'::"text"))))));



CREATE POLICY "resena_select" ON "public"."resena" FOR SELECT USING (true);



ALTER TABLE "public"."solicitud" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "solicitud_insert" ON "public"."solicitud" FOR INSERT WITH CHECK (("cliente_id" = ( SELECT "auth"."uid"() AS "uid")));



ALTER TABLE "public"."solicitud_objeto" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "solicitud_objeto_delete" ON "public"."solicitud_objeto" FOR DELETE USING (((EXISTS ( SELECT 1
   FROM "public"."solicitud" "s"
  WHERE (("s"."id" = "solicitud_objeto"."solicitud_id") AND ("s"."cliente_id" = ( SELECT "auth"."uid"() AS "uid"))))) OR "public"."fn_es_administrador"()));



CREATE POLICY "solicitud_objeto_insert" ON "public"."solicitud_objeto" FOR INSERT WITH CHECK (((EXISTS ( SELECT 1
   FROM "public"."solicitud" "s"
  WHERE (("s"."id" = "solicitud_objeto"."solicitud_id") AND ("s"."cliente_id" = ( SELECT "auth"."uid"() AS "uid"))))) OR "public"."fn_es_administrador"()));



CREATE POLICY "solicitud_objeto_select" ON "public"."solicitud_objeto" FOR SELECT USING ((EXISTS ( SELECT 1
   FROM "public"."solicitud" "s"
  WHERE ("s"."id" = "solicitud_objeto"."solicitud_id"))));



CREATE POLICY "solicitud_objeto_update" ON "public"."solicitud_objeto" FOR UPDATE USING (((EXISTS ( SELECT 1
   FROM "public"."solicitud" "s"
  WHERE (("s"."id" = "solicitud_objeto"."solicitud_id") AND ("s"."cliente_id" = ( SELECT "auth"."uid"() AS "uid"))))) OR "public"."fn_es_administrador"())) WITH CHECK (((EXISTS ( SELECT 1
   FROM "public"."solicitud" "s"
  WHERE (("s"."id" = "solicitud_objeto"."solicitud_id") AND ("s"."cliente_id" = ( SELECT "auth"."uid"() AS "uid"))))) OR "public"."fn_es_administrador"()));



CREATE POLICY "solicitud_select" ON "public"."solicitud" FOR SELECT USING ((("cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"() OR (("estado_codigo" = 'publicada'::"text") AND (EXISTS ( SELECT 1
   FROM "public"."transportista_zona" "tz"
  WHERE (("tz"."transportista_id" = ( SELECT "auth"."uid"() AS "uid")) AND ("tz"."zona_id" = ANY (ARRAY["solicitud"."origen_zona_id", "solicitud"."destino_zona_id"])))))) OR (EXISTS ( SELECT 1
   FROM "public"."oferta" "o"
  WHERE (("o"."solicitud_id" = "solicitud"."id") AND ("o"."transportista_id" = ( SELECT "auth"."uid"() AS "uid")))))));



CREATE POLICY "solicitud_update" ON "public"."solicitud" FOR UPDATE USING ((("cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"())) WITH CHECK ((("cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



ALTER TABLE "public"."tipo_documento" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "tipo_documento_delete" ON "public"."tipo_documento" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "tipo_documento_insert" ON "public"."tipo_documento" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "tipo_documento_select" ON "public"."tipo_documento" FOR SELECT USING (true);



CREATE POLICY "tipo_documento_update" ON "public"."tipo_documento" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."tipo_incidente" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "tipo_incidente_delete" ON "public"."tipo_incidente" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "tipo_incidente_insert" ON "public"."tipo_incidente" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "tipo_incidente_select" ON "public"."tipo_incidente" FOR SELECT USING (true);



CREATE POLICY "tipo_incidente_update" ON "public"."tipo_incidente" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."tipo_notificacion" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "tipo_notificacion_delete" ON "public"."tipo_notificacion" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "tipo_notificacion_insert" ON "public"."tipo_notificacion" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "tipo_notificacion_select" ON "public"."tipo_notificacion" FOR SELECT USING (true);



CREATE POLICY "tipo_notificacion_update" ON "public"."tipo_notificacion" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."tipo_resolucion_incidente" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "tipo_resolucion_incidente_delete" ON "public"."tipo_resolucion_incidente" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "tipo_resolucion_incidente_insert" ON "public"."tipo_resolucion_incidente" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "tipo_resolucion_incidente_select" ON "public"."tipo_resolucion_incidente" FOR SELECT USING (true);



CREATE POLICY "tipo_resolucion_incidente_update" ON "public"."tipo_resolucion_incidente" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."tipo_vehiculo" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "tipo_vehiculo_delete" ON "public"."tipo_vehiculo" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "tipo_vehiculo_insert" ON "public"."tipo_vehiculo" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "tipo_vehiculo_select" ON "public"."tipo_vehiculo" FOR SELECT USING (true);



CREATE POLICY "tipo_vehiculo_update" ON "public"."tipo_vehiculo" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."transportista" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "transportista_insert" ON "public"."transportista" FOR INSERT WITH CHECK (("usuario_id" = ( SELECT "auth"."uid"() AS "uid")));



CREATE POLICY "transportista_select" ON "public"."transportista" FOR SELECT USING (true);



CREATE POLICY "transportista_update" ON "public"."transportista" FOR UPDATE USING ((("usuario_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"())) WITH CHECK ((("usuario_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



ALTER TABLE "public"."transportista_zona" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "transportista_zona_delete" ON "public"."transportista_zona" FOR DELETE USING ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



CREATE POLICY "transportista_zona_insert" ON "public"."transportista_zona" FOR INSERT WITH CHECK ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



CREATE POLICY "transportista_zona_select" ON "public"."transportista_zona" FOR SELECT USING (true);



ALTER TABLE "public"."usuario" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "usuario_insert" ON "public"."usuario" FOR INSERT WITH CHECK (("id" = ( SELECT "auth"."uid"() AS "uid")));



CREATE POLICY "usuario_select" ON "public"."usuario" FOR SELECT USING ((("id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"() OR ("rol" = 'transportista'::"text") OR (EXISTS ( SELECT 1
   FROM ("public"."oferta" "o"
     JOIN "public"."solicitud" "s" ON (("s"."id" = "o"."solicitud_id")))
  WHERE (("o"."transportista_id" = ( SELECT "auth"."uid"() AS "uid")) AND ("s"."cliente_id" = "usuario"."id"))))));



CREATE POLICY "usuario_update" ON "public"."usuario" FOR UPDATE USING ((("id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"())) WITH CHECK ((("id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



ALTER TABLE "public"."vehiculo" ENABLE ROW LEVEL SECURITY;


ALTER TABLE "public"."vehiculo_costo" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "vehiculo_costo_insert" ON "public"."vehiculo_costo" FOR INSERT WITH CHECK (("public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."vehiculo" "v"
  WHERE (("v"."id" = "vehiculo_costo"."vehiculo_id") AND ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid")))))));



CREATE POLICY "vehiculo_costo_select" ON "public"."vehiculo_costo" FOR SELECT USING (("public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."vehiculo" "v"
  WHERE (("v"."id" = "vehiculo_costo"."vehiculo_id") AND ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid")))))));



CREATE POLICY "vehiculo_costo_update" ON "public"."vehiculo_costo" FOR UPDATE USING (("public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."vehiculo" "v"
  WHERE (("v"."id" = "vehiculo_costo"."vehiculo_id") AND ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid"))))))) WITH CHECK (("public"."fn_es_administrador"() OR (EXISTS ( SELECT 1
   FROM "public"."vehiculo" "v"
  WHERE (("v"."id" = "vehiculo_costo"."vehiculo_id") AND ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid")))))));



CREATE POLICY "vehiculo_insert" ON "public"."vehiculo" FOR INSERT WITH CHECK ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



CREATE POLICY "vehiculo_select" ON "public"."vehiculo" FOR SELECT USING (true);



CREATE POLICY "vehiculo_update" ON "public"."vehiculo" FOR UPDATE USING ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"())) WITH CHECK ((("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



ALTER TABLE "public"."veto" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "veto_insert" ON "public"."veto" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "veto_select" ON "public"."veto" FOR SELECT USING ((("usuario_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



CREATE POLICY "veto_update" ON "public"."veto" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



ALTER TABLE "public"."viaje" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "viaje_insert" ON "public"."viaje" FOR INSERT WITH CHECK ((("cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



CREATE POLICY "viaje_select" ON "public"."viaje" FOR SELECT USING ((("cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR ("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



ALTER TABLE "public"."viaje_ubicacion" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "viaje_ubicacion_insert" ON "public"."viaje_ubicacion" FOR INSERT WITH CHECK ((EXISTS ( SELECT 1
   FROM "public"."viaje" "v"
  WHERE (("v"."id" = "viaje_ubicacion"."viaje_id") AND ("v"."transportista_id" = ( SELECT "auth"."uid"() AS "uid"))))));



CREATE POLICY "viaje_ubicacion_select" ON "public"."viaje_ubicacion" FOR SELECT USING ((EXISTS ( SELECT 1
   FROM "public"."viaje" "v"
  WHERE ("v"."id" = "viaje_ubicacion"."viaje_id"))));



CREATE POLICY "viaje_update" ON "public"."viaje" FOR UPDATE USING ((("cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR ("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"())) WITH CHECK ((("cliente_id" = ( SELECT "auth"."uid"() AS "uid")) OR ("transportista_id" = ( SELECT "auth"."uid"() AS "uid")) OR "public"."fn_es_administrador"()));



ALTER TABLE "public"."zona" ENABLE ROW LEVEL SECURITY;


CREATE POLICY "zona_delete" ON "public"."zona" FOR DELETE USING ("public"."fn_es_administrador"());



CREATE POLICY "zona_insert" ON "public"."zona" FOR INSERT WITH CHECK ("public"."fn_es_administrador"());



CREATE POLICY "zona_select" ON "public"."zona" FOR SELECT USING (true);



CREATE POLICY "zona_update" ON "public"."zona" FOR UPDATE USING ("public"."fn_es_administrador"()) WITH CHECK ("public"."fn_es_administrador"());



GRANT USAGE ON SCHEMA "public" TO "postgres";
GRANT USAGE ON SCHEMA "public" TO "anon";
GRANT USAGE ON SCHEMA "public" TO "authenticated";
GRANT USAGE ON SCHEMA "public" TO "service_role";



GRANT ALL ON FUNCTION "public"."fn_es_administrador"() TO "anon";
GRANT ALL ON FUNCTION "public"."fn_es_administrador"() TO "authenticated";
GRANT ALL ON FUNCTION "public"."fn_es_administrador"() TO "service_role";



GRANT ALL ON FUNCTION "public"."fn_es_transportista_habilitado"() TO "anon";
GRANT ALL ON FUNCTION "public"."fn_es_transportista_habilitado"() TO "authenticated";
GRANT ALL ON FUNCTION "public"."fn_es_transportista_habilitado"() TO "service_role";



REVOKE ALL ON FUNCTION "public"."fn_proteger_campos_transportista"() FROM PUBLIC;
GRANT ALL ON FUNCTION "public"."fn_proteger_campos_transportista"() TO "service_role";



REVOKE ALL ON FUNCTION "public"."fn_proteger_campos_viaje"() FROM PUBLIC;
GRANT ALL ON FUNCTION "public"."fn_proteger_campos_viaje"() TO "service_role";



GRANT ALL ON TABLE "public"."administrador" TO "anon";
GRANT ALL ON TABLE "public"."administrador" TO "authenticated";
GRANT ALL ON TABLE "public"."administrador" TO "service_role";



GRANT ALL ON TABLE "public"."cliente" TO "anon";
GRANT ALL ON TABLE "public"."cliente" TO "authenticated";
GRANT ALL ON TABLE "public"."cliente" TO "service_role";



GRANT ALL ON TABLE "public"."config_comision" TO "anon";
GRANT ALL ON TABLE "public"."config_comision" TO "authenticated";
GRANT ALL ON TABLE "public"."config_comision" TO "service_role";



GRANT ALL ON TABLE "public"."config_costo_laboral" TO "anon";
GRANT ALL ON TABLE "public"."config_costo_laboral" TO "authenticated";
GRANT ALL ON TABLE "public"."config_costo_laboral" TO "service_role";



GRANT ALL ON TABLE "public"."config_impuesto" TO "anon";
GRANT ALL ON TABLE "public"."config_impuesto" TO "authenticated";
GRANT ALL ON TABLE "public"."config_impuesto" TO "service_role";



GRANT ALL ON TABLE "public"."config_operacion" TO "anon";
GRANT ALL ON TABLE "public"."config_operacion" TO "authenticated";
GRANT ALL ON TABLE "public"."config_operacion" TO "service_role";



GRANT ALL ON TABLE "public"."config_tarifa" TO "anon";
GRANT ALL ON TABLE "public"."config_tarifa" TO "authenticated";
GRANT ALL ON TABLE "public"."config_tarifa" TO "service_role";



GRANT ALL ON TABLE "public"."documento_transportista" TO "anon";
GRANT ALL ON TABLE "public"."documento_transportista" TO "authenticated";
GRANT ALL ON TABLE "public"."documento_transportista" TO "service_role";



GRANT ALL ON TABLE "public"."estado_habilitacion_transportista" TO "anon";
GRANT ALL ON TABLE "public"."estado_habilitacion_transportista" TO "authenticated";
GRANT ALL ON TABLE "public"."estado_habilitacion_transportista" TO "service_role";



GRANT ALL ON TABLE "public"."estado_incidente" TO "anon";
GRANT ALL ON TABLE "public"."estado_incidente" TO "authenticated";
GRANT ALL ON TABLE "public"."estado_incidente" TO "service_role";



GRANT ALL ON TABLE "public"."estado_oferta" TO "anon";
GRANT ALL ON TABLE "public"."estado_oferta" TO "authenticated";
GRANT ALL ON TABLE "public"."estado_oferta" TO "service_role";



GRANT ALL ON TABLE "public"."estado_pago" TO "anon";
GRANT ALL ON TABLE "public"."estado_pago" TO "authenticated";
GRANT ALL ON TABLE "public"."estado_pago" TO "service_role";



GRANT ALL ON TABLE "public"."estado_solicitud" TO "anon";
GRANT ALL ON TABLE "public"."estado_solicitud" TO "authenticated";
GRANT ALL ON TABLE "public"."estado_solicitud" TO "service_role";



GRANT ALL ON TABLE "public"."estado_viaje" TO "anon";
GRANT ALL ON TABLE "public"."estado_viaje" TO "authenticated";
GRANT ALL ON TABLE "public"."estado_viaje" TO "service_role";



GRANT ALL ON TABLE "public"."incidente" TO "anon";
GRANT ALL ON TABLE "public"."incidente" TO "authenticated";
GRANT ALL ON TABLE "public"."incidente" TO "service_role";



GRANT ALL ON TABLE "public"."incidente_resolucion" TO "anon";
GRANT ALL ON TABLE "public"."incidente_resolucion" TO "authenticated";
GRANT ALL ON TABLE "public"."incidente_resolucion" TO "service_role";



GRANT ALL ON TABLE "public"."mensaje" TO "anon";
GRANT ALL ON TABLE "public"."mensaje" TO "authenticated";
GRANT ALL ON TABLE "public"."mensaje" TO "service_role";



GRANT ALL ON TABLE "public"."notificacion" TO "anon";
GRANT ALL ON TABLE "public"."notificacion" TO "authenticated";
GRANT ALL ON TABLE "public"."notificacion" TO "service_role";



GRANT ALL ON TABLE "public"."objeto" TO "anon";
GRANT ALL ON TABLE "public"."objeto" TO "authenticated";
GRANT ALL ON TABLE "public"."objeto" TO "service_role";



GRANT ALL ON TABLE "public"."oferta" TO "anon";
GRANT ALL ON TABLE "public"."oferta" TO "authenticated";
GRANT ALL ON TABLE "public"."oferta" TO "service_role";



GRANT ALL ON TABLE "public"."pago" TO "anon";
GRANT ALL ON TABLE "public"."pago" TO "authenticated";
GRANT ALL ON TABLE "public"."pago" TO "service_role";



GRANT ALL ON TABLE "public"."pago_movimiento" TO "anon";
GRANT ALL ON TABLE "public"."pago_movimiento" TO "authenticated";
GRANT ALL ON TABLE "public"."pago_movimiento" TO "service_role";



GRANT ALL ON TABLE "public"."resena" TO "anon";
GRANT ALL ON TABLE "public"."resena" TO "authenticated";
GRANT ALL ON TABLE "public"."resena" TO "service_role";



GRANT ALL ON TABLE "public"."solicitud" TO "anon";
GRANT ALL ON TABLE "public"."solicitud" TO "authenticated";
GRANT ALL ON TABLE "public"."solicitud" TO "service_role";



GRANT ALL ON TABLE "public"."solicitud_objeto" TO "anon";
GRANT ALL ON TABLE "public"."solicitud_objeto" TO "authenticated";
GRANT ALL ON TABLE "public"."solicitud_objeto" TO "service_role";



GRANT ALL ON TABLE "public"."tipo_documento" TO "anon";
GRANT ALL ON TABLE "public"."tipo_documento" TO "authenticated";
GRANT ALL ON TABLE "public"."tipo_documento" TO "service_role";



GRANT ALL ON TABLE "public"."tipo_incidente" TO "anon";
GRANT ALL ON TABLE "public"."tipo_incidente" TO "authenticated";
GRANT ALL ON TABLE "public"."tipo_incidente" TO "service_role";



GRANT ALL ON TABLE "public"."tipo_notificacion" TO "anon";
GRANT ALL ON TABLE "public"."tipo_notificacion" TO "authenticated";
GRANT ALL ON TABLE "public"."tipo_notificacion" TO "service_role";



GRANT ALL ON TABLE "public"."tipo_resolucion_incidente" TO "anon";
GRANT ALL ON TABLE "public"."tipo_resolucion_incidente" TO "authenticated";
GRANT ALL ON TABLE "public"."tipo_resolucion_incidente" TO "service_role";



GRANT ALL ON TABLE "public"."tipo_vehiculo" TO "anon";
GRANT ALL ON TABLE "public"."tipo_vehiculo" TO "authenticated";
GRANT ALL ON TABLE "public"."tipo_vehiculo" TO "service_role";



GRANT ALL ON TABLE "public"."transportista" TO "anon";
GRANT ALL ON TABLE "public"."transportista" TO "authenticated";
GRANT ALL ON TABLE "public"."transportista" TO "service_role";



GRANT ALL ON TABLE "public"."transportista_zona" TO "anon";
GRANT ALL ON TABLE "public"."transportista_zona" TO "authenticated";
GRANT ALL ON TABLE "public"."transportista_zona" TO "service_role";



GRANT ALL ON TABLE "public"."usuario" TO "anon";
GRANT ALL ON TABLE "public"."usuario" TO "authenticated";
GRANT ALL ON TABLE "public"."usuario" TO "service_role";



GRANT ALL ON TABLE "public"."vehiculo" TO "anon";
GRANT ALL ON TABLE "public"."vehiculo" TO "authenticated";
GRANT ALL ON TABLE "public"."vehiculo" TO "service_role";



GRANT ALL ON TABLE "public"."vehiculo_costo" TO "anon";
GRANT ALL ON TABLE "public"."vehiculo_costo" TO "authenticated";
GRANT ALL ON TABLE "public"."vehiculo_costo" TO "service_role";



GRANT ALL ON TABLE "public"."veto" TO "anon";
GRANT ALL ON TABLE "public"."veto" TO "authenticated";
GRANT ALL ON TABLE "public"."veto" TO "service_role";



GRANT ALL ON TABLE "public"."viaje" TO "anon";
GRANT ALL ON TABLE "public"."viaje" TO "authenticated";
GRANT ALL ON TABLE "public"."viaje" TO "service_role";



GRANT ALL ON TABLE "public"."viaje_ubicacion" TO "anon";
GRANT ALL ON TABLE "public"."viaje_ubicacion" TO "authenticated";
GRANT ALL ON TABLE "public"."viaje_ubicacion" TO "service_role";



GRANT ALL ON SEQUENCE "public"."viaje_ubicacion_id_seq" TO "anon";
GRANT ALL ON SEQUENCE "public"."viaje_ubicacion_id_seq" TO "authenticated";
GRANT ALL ON SEQUENCE "public"."viaje_ubicacion_id_seq" TO "service_role";



GRANT ALL ON TABLE "public"."zona" TO "anon";
GRANT ALL ON TABLE "public"."zona" TO "authenticated";
GRANT ALL ON TABLE "public"."zona" TO "service_role";



ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON SEQUENCES TO "postgres";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON SEQUENCES TO "anon";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON SEQUENCES TO "authenticated";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON SEQUENCES TO "service_role";






ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON FUNCTIONS TO "postgres";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON FUNCTIONS TO "anon";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON FUNCTIONS TO "authenticated";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON FUNCTIONS TO "service_role";






ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON TABLES TO "postgres";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON TABLES TO "anon";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON TABLES TO "authenticated";
ALTER DEFAULT PRIVILEGES FOR ROLE "postgres" IN SCHEMA "public" GRANT ALL ON TABLES TO "service_role";








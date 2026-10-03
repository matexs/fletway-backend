SET session_replication_role = replica;

--
-- PostgreSQL database dump
--

-- \restrict xnCPTjWgLNnXSksfD3B7hbDWp1kupuXZNpTvXLbTYXMoyGWB81n2bmzuxTKndp6

-- Dumped from database version 17.6
-- Dumped by pg_dump version 17.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: usuario; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: administrador; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: cliente; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: config_comision; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."config_comision" ("id", "porcentaje", "vigente_desde", "vigente_hasta") VALUES
	('7deb9f56-89a1-4b49-a3df-a95f9909399b', 15.00, '2026-08-22 22:01:10.169329+00', NULL);


--
-- Data for Name: config_costo_laboral; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."config_costo_laboral" ("id", "salario_basico_chofer", "salario_basico_ayudante", "adicionales_pct_chofer", "adicionales_pct_ayudante", "viaticos_diarios", "contribuciones_seg_social_pct", "obra_social_pct", "art_pct", "seguro_vida_mensual", "horas_mensuales", "horas_diarias", "vigente_desde", "vigente_hasta") VALUES
	('d0d7f790-62eb-4d62-9ec7-1ae8610e69de', 1037544.76, 963809.78, 18.00, 16.00, 24724.33, 18.00, 6.00, 10.00, 424.62, 192.00, 8.00, '2026-09-25 00:49:08.031062+00', NULL);


--
-- Data for Name: config_impuesto; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."config_impuesto" ("id", "iva_pct", "vigente_desde", "vigente_hasta") VALUES
	('d258fa5c-fb30-41dc-9c62-c691b5fa26d3', 21.00, '2026-09-25 00:49:08.031062+00', NULL);


--
-- Data for Name: config_operacion; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."config_operacion" ("id", "tiempo_base_operacion_min", "tiempo_espera_min", "tiempo_por_objeto_min", "tiempo_por_kg_min", "tiempo_por_m3_min", "tiempo_por_metro_min", "tiempo_por_escalera_min", "eficiencia_ayudante", "vigente_desde", "vigente_hasta") VALUES
	('2c86f2f7-0554-48f2-b77f-52ee123fe240', 10.00, 30.00, 2.000, 0.010, 8.000, 0.020, 5.000, 0.700, '2026-09-25 00:49:08.031062+00', NULL);


--
-- Data for Name: config_tarifa; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."config_tarifa" ("id", "tarifa_base", "valor_por_km", "valor_por_m3", "valor_por_hora", "recargo_escalera_por_piso", "recargo_por_ayudante", "vigente_desde", "vigente_hasta") VALUES
	('687b2c59-699a-486a-8e5e-192c07dd803f', 5000.00, 350.00, 1200.00, 4000.00, 800.00, 6000.00, '2026-08-22 22:01:10.169329+00', NULL);


--
-- Data for Name: estado_habilitacion_transportista; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."estado_habilitacion_transportista" ("codigo", "descripcion") VALUES
	('pendiente', 'Documentación cargada, esperando revisión del Administrador'),
	('habilitado', 'Documentación validada, puede operar'),
	('rechazado', 'Documentación rechazada, puede volver a presentarla (RF-01)');


--
-- Data for Name: tipo_documento; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."tipo_documento" ("codigo", "descripcion") VALUES
	('dni', 'Documento Nacional de Identidad'),
	('seguro', 'Seguro del vehículo'),
	('registro', 'Registro de conducir habilitante'),
	('vtv', 'Verificación Técnica Vehicular');


--
-- Data for Name: transportista; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: documento_transportista; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: estado_incidente; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."estado_incidente" ("codigo", "descripcion") VALUES
	('abierto', 'Reportado, sin revisar'),
	('en_revision', 'El Administrador lo está evaluando'),
	('resuelto', 'Resolución aplicada y registrada');


--
-- Data for Name: estado_oferta; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."estado_oferta" ("codigo", "descripcion") VALUES
	('pendiente', 'Postulación registrada, esperando decisión del Cliente'),
	('aceptada', 'Elegida por el Cliente, dio origen a un Viaje'),
	('no_seleccionada', 'El Cliente eligió otra oferta'),
	('retirada', 'El Transportista retiró su postulación');


--
-- Data for Name: estado_pago; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."estado_pago" ("codigo", "descripcion") VALUES
	('pendiente', 'Pago iniciado, sin confirmar por la pasarela'),
	('retenido', 'Cobrado al Cliente, en espera de liberación al Transportista'),
	('liberado', 'Liberado al Transportista, descontada la comisión'),
	('reembolsado_total', 'Devuelto en su totalidad al Cliente'),
	('reembolsado_parcial', 'Devuelto parcialmente al Cliente');


--
-- Data for Name: estado_solicitud; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."estado_solicitud" ("codigo", "descripcion") VALUES
	('publicada', 'Visible para transportistas compatibles, esperando postulaciones'),
	('asignada', 'El Cliente eligió una oferta, viaje confirmado'),
	('cancelada', 'Cancelada por el Cliente antes de asignar'),
	('expirada', 'Sin postulaciones ni asignación, cerrada por el sistema');


--
-- Data for Name: estado_viaje; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."estado_viaje" ("codigo", "descripcion") VALUES
	('confirmado', 'Oferta aceptada, viaje aún no iniciado'),
	('en_curso', 'PIN de inicio validado, servicio en ejecución'),
	('finalizado', 'PIN de fin validado, habilita calificación (RF-12)'),
	('cancelado_cliente', 'Cancelado por el Cliente (RF-08)'),
	('cancelado_transportista', 'Cancelado por el Transportista (RF-19)');


--
-- Data for Name: zona; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: solicitud; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: tipo_vehiculo; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."tipo_vehiculo" ("id", "nombre", "volumen_estandar_m3", "peso_maximo_estandar_kg") VALUES
	('9d64a5e6-01a4-4f71-b403-7b18ceb13e44', 'Utilitario chico', 3.00, 650.00),
	('1f94f02c-5ac3-467e-9199-68b5c00034e3', 'Furgón mediano', 8.00, 1200.00),
	('1e9e50b1-ca9a-467c-944a-1d222fe4fdb8', 'Camión chico', 20.00, 3500.00),
	('8e097d07-21ed-481b-81ad-2c2f3dcf8ccb', 'Camión mediano', 40.00, 7000.00);


--
-- Data for Name: vehiculo; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: oferta; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: tipo_incidente; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."tipo_incidente" ("codigo", "descripcion") VALUES
	('fraude', 'Sospecha de fraude o estafa'),
	('dano_carga', 'Daño a la carga transportada'),
	('incumplimiento', 'Incumplimiento del servicio acordado'),
	('disputa_pago', 'Disputa sobre el monto cobrado o liberado'),
	('otro', 'Otro tipo de problema no categorizado');


--
-- Data for Name: viaje; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: incidente; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: tipo_resolucion_incidente; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."tipo_resolucion_incidente" ("codigo", "descripcion") VALUES
	('reembolso_total', 'Reembolso total del monto al Cliente'),
	('reembolso_parcial', 'Reembolso parcial del monto al Cliente'),
	('liberacion_total', 'Liberación total del monto al Transportista'),
	('liberacion_parcial', 'Liberación parcial del monto al Transportista'),
	('veto', 'Bloqueo de cuenta del Cliente o Transportista involucrado'),
	('cierre_sin_accion', 'Cierre del incidente sin acción');


--
-- Data for Name: veto; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: incidente_resolucion; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: mensaje; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: tipo_notificacion; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."tipo_notificacion" ("codigo", "descripcion") VALUES
	('solicitud_compatible', 'Nueva solicitud publicada compatible con zona/vehículo del Transportista'),
	('postulacion_recibida', 'Nueva postulación recibida para una Solicitud del Cliente'),
	('viaje_confirmado', 'Oferta aceptada, viaje confirmado'),
	('viaje_cancelado', 'Viaje cancelado por Cliente o Transportista'),
	('cambio_estado_viaje', 'Cambio de estado del viaje (en curso, finalizado, etc.)'),
	('pago_procesado', 'Pago procesado o liberado'),
	('incidente_resuelto', 'Incidente reportado fue resuelto'),
	('documentacion_revisada', 'Documentación de Transportista aprobada o rechazada'),
	('veto_aplicado', 'Cuenta vetada temporal o definitivamente');


--
-- Data for Name: notificacion; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: objeto; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO "public"."objeto" ("id", "nombre", "peso_estimado_kg", "volumen_estimado_m3", "largo_m", "ancho_m", "alto_m", "rotacion_horizontal", "rotacion_vertical", "apilable") VALUES
	('9303b17b-3955-40bd-a388-95f9b51959b2', 'Sofá 2 cuerpos', 40.00, 1.800, NULL, NULL, NULL, true, true, true),
	('70dd56ec-6dc8-4463-a322-173cdc0f1328', 'Heladera', 70.00, 1.200, NULL, NULL, NULL, true, true, true),
	('8bafee01-c7ae-4667-8405-95443b0d7a11', 'Cama matrimonial (colchón + base)', 50.00, 2.000, NULL, NULL, NULL, true, true, true),
	('b66e9a75-a7ea-4f02-aadf-bc7dc4fa4a08', 'Caja mudanza estándar', 15.00, 0.100, NULL, NULL, NULL, true, true, true),
	('394be710-c024-4f11-964b-40cde2b9eeb1', 'Mesa de comedor', 25.00, 0.900, NULL, NULL, NULL, true, true, true);


--
-- Data for Name: pago; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: pago_movimiento; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: resena; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: solicitud_objeto; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: transportista_zona; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: vehiculo_costo; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Data for Name: viaje_ubicacion; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- Name: viaje_ubicacion_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('"public"."viaje_ubicacion_id_seq"', 1, false);


--
-- PostgreSQL database dump complete
--

-- \unrestrict xnCPTjWgLNnXSksfD3B7hbDWp1kupuXZNpTvXLbTYXMoyGWB81n2bmzuxTKndp6

RESET ALL;

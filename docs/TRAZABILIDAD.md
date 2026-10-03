# Trazabilidad ERS → implementación — fletway-backend

> Matriz que conecta cada requisito de la ERS con su implementación real en el backend.
> La skill `trace-requirement` la usa/actualiza antes de dar por cerrado un endpoint.
> Fuente de requisitos: `docs/ERS_Fletway.docx`.

**Estados:** `NO` no empezado · `EN CURSO` en progreso · `OK` implementado + trazado · `N/A` fuera del backend

**Última actualización:** 2026-10-01. Definiciones para la construcción (D-15 a D-31, `docs/PLAN_CONSTRUCCION.md`) y enmiendas a la ERS (sección final). Todo sigue en `NO` salvo lo marcado `N/A`.

> (nuevo) = tabla o columna agregada por las migraciones `migrations/0001`–`0007` (**aplicadas el 2026-09-24**).

---

## Requisitos funcionales — Administrador

| RF | Título | Estado | Endpoint(s) / servicio | Tablas | Tests |
|----|--------|--------|------------------------|--------|-------|
| RF-01 | Validar documentación de Transportistas | `EN CURSO` | `GET /api/admin/transportistas`, `POST /api/admin/documentos/{id}/aprobar` y `/rechazar` (`internal/feature/habilitacion`); estado derivado en la base (`0012`, D-33); notificación por `Notificador` (D-22). Falta la app | `documento_transportista`, `transportista`, `estado_habilitacion_transportista`, `notificacion` | `habilitacion_test.go` (flujo D-33, errores, Administrador, seguridad en la base) |
| RF-02 | Ver reportes de incidentes | `NO` | — | `incidente` | — |
| RF-03 | Resolver incidentes | `NO` | — | `incidente_resolucion`, `tipo_resolucion_incidente`, `pago_movimiento`, `veto` | — |
| RF-04 | Vetar Clientes o Transportistas | `NO` | — | `veto` | — |

## Requisitos funcionales — Cliente

| RF | Título | Estado | Endpoint(s) / servicio | Tablas | Tests |
|----|--------|--------|------------------------|--------|-------|
| RF-05 | Registro de Cliente | `EN CURSO` | `trg_alta_usuario` (`0011`) + `POST /api/auth/registro/cliente`, `GET /api/me` (`internal/feature/identidad`). Falta la pantalla de la app | `usuario`, `cliente` | `identidad_test.go` (registro, errores, trigger, RLS) |
| RF-06 | Publicar necesidad de servicio | `NO` | — (**sin cotización estimada**: el endpoint no calcula ni devuelve monto) | `solicitud` (acceso origen/destino (nuevo)), `solicitud_objeto` (dimensiones y flags (nuevo), copiados desde `objeto`), `objeto` | — |
| RF-07 | Elegir entre ofertas de Transportistas | `NO` | — (al aceptar, el `viaje` **copia** el desglose de costo de la `oferta` a `*_snapshot` (nuevo); no recalcula) | `oferta`, `viaje`, `estado_oferta`, `estado_viaje` | — |
| RF-08 | Cancelar servicio | `NO` | — | `viaje`, `estado_viaje`, `pago`, `pago_movimiento` | — |
| RF-09 | Recibir notificaciones de cambio de estado | `NO` | — | `notificacion`, `tipo_notificacion` | — |
| RF-10 | Chat con el Transportista | `N/A` | Supabase Realtime directo (D-10); backend solo habilita el `viaje` | `mensaje` | — |
| RF-11 | Ver perfiles de Transportistas | `NO` | — | `transportista`, `resena`, `vehiculo` | — |
| RF-12 | Calificar el servicio | `NO` | — | `resena` | — |
| RF-13 | Reportar problemas | `NO` | — | `incidente`, `tipo_incidente` | — |
| RF-14 | Ver historial de viajes | `NO` | — | `viaje` | — |
| RF-15 | Ver ubicación en tiempo real del Transportista | `N/A` | Supabase Realtime directo (D-10) sobre `viaje_ubicacion` | `viaje_ubicacion` | — |

## Requisitos funcionales — Transportista

| RF | Título | Estado | Endpoint(s) / servicio | Tablas | Tests |
|----|--------|--------|------------------------|--------|-------|
| RF-16 | Registro de Transportista | `EN CURSO` | Alta de cuenta (módulo 2) y documentación: `POST/GET /api/transportista/documentos`, bucket `documentos-transportista` con policies de Storage (`0012`). Falta la app | `usuario`, `transportista`, `documento_transportista` | `identidad_test.go`, `habilitacion_test.go` |
| RF-17 | Ver y ofertar sobre necesidades de servicio | `NO` | — (al ofertar corren `planificarViajes` + `calcularCostoOferta`; si no entra → error de validación) | `solicitud`, `solicitud_objeto`, `oferta` (desglose (nuevo)), `transportista_zona`, `vehiculo`, `vehiculo_costo` (nuevo), `config_costo_laboral` (nuevo), `config_operacion` (nuevo), `config_impuesto` (nuevo), `config_comision` | — |
| RF-18 | Registrar vehículo | `EN CURSO` | `GET /api/tipos-vehiculo`, `POST/GET /api/transportista/vehiculos`, `PUT .../{id}/activo`, `PUT/GET .../{id}/costos` (`internal/feature/vehiculo`); tipos con medidas estándar (`0013`, D-32). Falta la app | `vehiculo` (dimensiones útiles (nuevo)), `vehiculo_costo` (nuevo), `tipo_vehiculo` | `vehiculo_test.go` (alta, errores, activo, costos, RLS de `vehiculo_costo`) |
| RF-19 | Cancelar viajes | `NO` | — | `viaje`, `transportista` (tasa_cumplimiento) | — |
| RF-20 | Chat con el Cliente | `N/A` | Supabase Realtime directo (D-10) | `mensaje` | — |
| RF-21 | Ver información del viaje | `NO` | — | `viaje` | — |
| RF-22 | Realizar el viaje (PIN inicio/fin) | `NO` | — | `viaje`, `viaje_ubicacion` | — |
| RF-23 | Reportar problemas | `NO` | — | `incidente` | — |
| RF-24 | Ver historial de viajes | `NO` | — | `viaje` | — |

---

## Reglas de negocio

| RN | Título | Estado | Dónde se implementa | Verificación |
|----|--------|--------|---------------------|--------------|
| RN-01 | Cálculo automático del precio | `NO` | servicio de cotización, **sólo al crear la `oferta`** (RF-17). Diseño: `docs/ALGORITMO_COTIZACION.md` | test con `config_costo_laboral`/`config_operacion`/`config_impuesto`/`config_comision` (nuevo) fijas + `vehiculo_costo` (nuevo) conocido → `precio_calculado` esperado; el Cliente nunca manda precio; `solicitud` no expone monto. Fuera de alcance: término de demanda. Abierto: ubicación de `margen_pct` |
| RN-02 | Cálculo automático de cantidad de viajes / cubicaje | `NO` | `planificarViajes` con boxpacker3 **v2** (greedy, estimativo), sobre el **`vehiculo` real** de la oferta, no sobre `tipo_vehiculo`. Diseño: `docs/ALGORITMO_VIAJES_EMPAQUETADO.md` | test: carga conocida → `oferta.cantidad_viajes`; objeto que no entra → `ErrNoFactible` con motivo; `rotacion_vertical`/`apilable` respetados. **Atención:** Difiere de la ERS ("por tipo de vehículo"): ver §7 del doc |
| RN-03 | Comisión de la plataforma | `NO` | servicio de pagos | `%` desde `config_comision` vigente, congelado en `viaje.porcentaje_comision_snapshot` |
| RN-04 | Matchmaking por localidad | `EN CURSO` | zonas de trabajo: `GET /api/zonas`, `GET/PUT /api/transportista/zonas` (`internal/feature/geografia`); la query de matching es del módulo 7 | zona del transportista ∈ {origen, destino} de la solicitud; sin ensanchamiento |
| RN-05 | Postulación pull + aviso + top 3 por score | `NO` | job async (aviso **in-app** en esta etapa, D-22) + endpoint de ofertas (top 3) | score = f(precio, calificación, tasa de cumplimiento), **sin cercanía** (D-25, `docs/ALGORITMO_SCORE.md`) |
| RN-06 | Verificación por PIN + geolocalización | `NO` | endpoints de ejecución de viaje | 2 PIN (inicio/fin) + ping GPS; `resena` habilitada tras PIN de fin válido; el Transportista nunca ve el PIN, lo dicta el Cliente (D-26) |
| RN-07 | Cancelación con costo | `NO` | endpoints de cancelación | sin cargo si no salió; cargo de resarcimiento si salió; transportista no paga penalización; "salió" = `viaje.salio_en`, cargo del 20 % (D-27) |
| RN-08 | Catálogo de objetos comunes | `EN CURSO` | `GET /api/catalogo/objetos` (`internal/feature/catalogo`, `catalogo_test.go`); falta la copia a `solicitud_objeto` (módulo 6) y su uso en la cotización (módulo 8) | `solicitud_objeto` admite `objeto_id` o carga manual; en ambos casos guarda su copia de peso, largo/ancho/alto (nuevo) y flags de rotación/apilado (nuevo) |

---

## Requisitos no funcionales

| RNF | Estado | Cómo se satisface |
|-----|--------|-------------------|
| RNF-01 — JWT en todos los endpoints | `EN CURSO` | middleware `auth` obligatorio en el mux, con verificación ES256 contra el JWKS de Supabase (D-04, módulo 1, 2026-10-02). Desde el módulo 2 lo usan `GET /api/me` y los registros. Pasa a `OK` cuando la app consuma la API con el JWT |
| RNF-02 — Async y concurrente | `EN CURSO` | `internal/platform/async` (pool de workers); handlers no bloquean en side-effects |
| RNF-03 — 3FN + snapshot en `viaje` | `OK` (en DB) | ya en el esquema; el backend debe **escribir** los `*_snapshot` al confirmar viaje. Los snapshots de tarifa plana quedaron DEPRECATED y se reemplazaron por el desglose de costo (migración 0006, aplicada) |
| RNF-04 — Mínima decisión del Cliente | `EN CURSO` | precio automático (RN-01), top 3 (RN-05) — se respeta en el diseño de endpoints |

## Requisitos de interfaces

| RI | Estado | Nota |
|----|--------|------|
| RI-01 — Web admin Angular | `N/A` | Fuera de alcance de esta etapa. |
| RI-02 — App móvil Flutter | `N/A` | Repo `fletway-mobile`. |
| RI-03 — Pasarela de pagos | `NO` | Mercado Pago (D-09), con retención, split y webhook (D-28). Módulo 12. |
| RI-04 — Geolocalización en tiempo real | `N/A` | Supabase Realtime directo (D-10). |
| RI-05 — Chat condicionado a viaje aceptado | `EN CURSO` | La condición "existe `viaje`" la garantiza el backend; el transporte de mensajes es Realtime. |

---

## Enmiendas a la ERS

Desviaciones decididas respecto de `docs/ERS_Fletway.docx`. Son decisiones deliberadas, no omisiones. Desde el 2026-10-01 constan también en la propia ERS, sección "5. Anexo — Registro de cambios" (D-25).

| Requisito | Qué dice la ERS | Qué se implementa | Decisión |
|---|---|---|---|
| RN-01 / RF-06 | Cotización estimada al publicar la solicitud | Sin cotización estimada: el único precio es el de cada oferta | D-13 |
| RN-01 | La "demanda" es factor del precio | Sin término de demanda en esta versión | `ALGORITMO_COTIZACION.md` §0.3 |
| RN-02 | Viajes y cubicaje por tipo de vehículo, al publicar | Por vehículo real, al ofertar | D-14 |
| RN-05 | Score con precio, calificación, **cercanía** y cumplimiento | Score **sin cercanía** | D-25 |
| RN-05 | Notificación **push** proactiva | Aviso **in-app** en esta etapa; push en una etapa posterior | D-17, D-22 |
| RF-21 / RN-06 | El Transportista ve "el PIN correspondiente" | El Transportista ve el campo para cargarlo; el valor lo dicta el Cliente | D-26 |
| RI-01 | Web de administración en Angular | Fuera de esta etapa: el Administrador opera con endpoints desde Postman | D-17 |
| RI-02 | App Flutter para Cliente y Transportista | Sólo Android en esta etapa | D-17 |
| RI-03 / RNF-01 | JWT en todos los endpoints | Excepción: webhook de Mercado Pago, protegido por firma | D-28 |
| — | (no especificado) | Facturación fiscal fuera de alcance; IVA parametrizado | D-17 |


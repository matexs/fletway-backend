# Trazabilidad ERS → implementación — fletway-backend

> Matriz que conecta cada requisito de la ERS con su implementación real en el backend.
> La skill `trace-requirement` la usa/actualiza antes de dar por cerrado un endpoint.
> Fuente de requisitos: `docs/ERS_Fletway.pdf`.

**Estados:** `NO` no empezado · `EN CURSO` en progreso · `OK` implementado + trazado · `N/A` fuera del backend

**Última actualización:** 2026-09-24. Se rediseñaron RN-01, RN-02 y RN-08 según `docs/ALGORITMO_COTIZACION.md` y `docs/ALGORITMO_VIAJES_EMPAQUETADO.md`. Todo sigue en `NO` salvo lo marcado `N/A`.

> (nuevo) = tabla o columna agregada por las migraciones `migrations/0001`–`0007` (**aplicadas el 2026-09-24**).

---

## Requisitos funcionales — Administrador

| RF | Título | Estado | Endpoint(s) / servicio | Tablas | Tests |
|----|--------|--------|------------------------|--------|-------|
| RF-01 | Validar documentación de Transportistas | `NO` | — | `documento_transportista`, `transportista`, `estado_habilitacion_transportista` | — |
| RF-02 | Ver reportes de incidentes | `NO` | — | `incidente` | — |
| RF-03 | Resolver incidentes | `NO` | — | `incidente_resolucion`, `tipo_resolucion_incidente`, `pago_movimiento`, `veto` | — |
| RF-04 | Vetar Clientes o Transportistas | `NO` | — | `veto` | — |

## Requisitos funcionales — Cliente

| RF | Título | Estado | Endpoint(s) / servicio | Tablas | Tests |
|----|--------|--------|------------------------|--------|-------|
| RF-05 | Registro de Cliente | `NO` | — | `usuario`, `cliente` | — |
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
| RF-16 | Registro de Transportista | `NO` | — | `usuario`, `transportista`, `documento_transportista` | — |
| RF-17 | Ver y ofertar sobre necesidades de servicio | `NO` | — (al ofertar corren `planificarViajes` + `calcularCostoOferta`; si no entra → error de validación) | `solicitud`, `solicitud_objeto`, `oferta` (desglose (nuevo)), `transportista_zona`, `vehiculo`, `vehiculo_costo` (nuevo), `config_costo_laboral` (nuevo), `config_operacion` (nuevo), `config_impuesto` (nuevo), `config_comision` | — |
| RF-18 | Registrar vehículo | `NO` | — | `vehiculo` (dimensiones útiles (nuevo)), `vehiculo_costo` (nuevo), `tipo_vehiculo` | — |
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
| RN-04 | Matchmaking por localidad | `NO` | query de matching | zona del transportista ∈ {origen, destino} de la solicitud; sin ensanchamiento |
| RN-05 | Postulación pull + push + top 3 por score | `NO` | job async (push) + endpoint listado ofertas (top 3) | score = f(precio, calificación, cercanía, tasa_cumplimiento) |
| RN-06 | Verificación por PIN + geolocalización | `NO` | endpoints de ejecución de viaje | 2 PIN (inicio/fin) + ping GPS; `resena` habilitada tras PIN de fin válido |
| RN-07 | Cancelación con costo | `NO` | endpoints de cancelación | sin cargo si no salió; cargo de resarcimiento si salió; transportista no paga penalización |
| RN-08 | Catálogo de objetos comunes | `NO` | endpoint de catálogo + cotización | `solicitud_objeto` admite `objeto_id` o carga manual; en ambos casos guarda su copia de peso, largo/ancho/alto (nuevo) y flags de rotación/apilado (nuevo) |

---

## Requisitos no funcionales

| RNF | Estado | Cómo se satisface |
|-----|--------|-------------------|
| RNF-01 — JWT en todos los endpoints | `EN CURSO` | middleware `auth` obligatorio en el mux; ningún handler de negocio sin él |
| RNF-02 — Async y concurrente | `EN CURSO` | `internal/platform/async` (pool de workers); handlers no bloquean en side-effects |
| RNF-03 — 3FN + snapshot en `viaje` | `OK` (en DB) | ya en el esquema; el backend debe **escribir** los `*_snapshot` al confirmar viaje. Los snapshots de tarifa plana quedaron DEPRECATED y se reemplazaron por el desglose de costo (migración 0006, aplicada) |
| RNF-04 — Mínima decisión del Cliente | `EN CURSO` | precio automático (RN-01), top 3 (RN-05) — se respeta en el diseño de endpoints |

## Requisitos de interfaces

| RI | Estado | Nota |
|----|--------|------|
| RI-01 — Web admin Angular | `N/A` | Fuera de alcance de esta etapa. |
| RI-02 — App móvil Flutter | `N/A` | Repo `fletway-mobile`. |
| RI-03 — Pasarela de pagos | `NO` | Mercado Pago / Stripe sin decidir (D-09). |
| RI-04 — Geolocalización en tiempo real | `N/A` | Supabase Realtime directo (D-10). |
| RI-05 — Chat condicionado a viaje aceptado | `EN CURSO` | La condición "existe `viaje`" la garantiza el backend; el transporte de mensajes es Realtime. |

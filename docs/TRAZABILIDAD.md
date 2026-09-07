# Trazabilidad ERS → implementación — fletway-backend

> Matriz que conecta cada requisito de la ERS con su implementación real en el backend.
> La skill `trace-requirement` la usa/actualiza antes de dar por cerrado un endpoint.
> Fuente de requisitos: `docs/ERS_Fletway.pdf`.

**Estados:** ❌ no empezado · 🟡 en progreso · ✅ implementado + trazado · ➖ fuera del backend

**Última actualización:** 2026-09-07 (scaffolding — todo en ❌ salvo lo marcado ➖)

---

## Requisitos funcionales — Administrador

| RF | Título | Estado | Endpoint(s) / servicio | Tablas | Tests |
|----|--------|--------|------------------------|--------|-------|
| RF-01 | Validar documentación de Transportistas | ❌ | — | `documento_transportista`, `transportista`, `estado_habilitacion_transportista` | — |
| RF-02 | Ver reportes de incidentes | ❌ | — | `incidente` | — |
| RF-03 | Resolver incidentes | ❌ | — | `incidente_resolucion`, `tipo_resolucion_incidente`, `pago_movimiento`, `veto` | — |
| RF-04 | Vetar Clientes o Transportistas | ❌ | — | `veto` | — |

## Requisitos funcionales — Cliente

| RF | Título | Estado | Endpoint(s) / servicio | Tablas | Tests |
|----|--------|--------|------------------------|--------|-------|
| RF-05 | Registro de Cliente | ❌ | — | `usuario`, `cliente` | — |
| RF-06 | Publicar necesidad de servicio | ❌ | — | `solicitud`, `solicitud_objeto`, `objeto` | — |
| RF-07 | Elegir entre ofertas de Transportistas | ❌ | — | `oferta`, `viaje`, `estado_oferta`, `estado_viaje` | — |
| RF-08 | Cancelar servicio | ❌ | — | `viaje`, `estado_viaje`, `pago`, `pago_movimiento` | — |
| RF-09 | Recibir notificaciones de cambio de estado | ❌ | — | `notificacion`, `tipo_notificacion` | — |
| RF-10 | Chat con el Transportista | ➖ | Supabase Realtime directo (D-10); backend solo habilita el `viaje` | `mensaje` | — |
| RF-11 | Ver perfiles de Transportistas | ❌ | — | `transportista`, `resena`, `vehiculo` | — |
| RF-12 | Calificar el servicio | ❌ | — | `resena` | — |
| RF-13 | Reportar problemas | ❌ | — | `incidente`, `tipo_incidente` | — |
| RF-14 | Ver historial de viajes | ❌ | — | `viaje` | — |
| RF-15 | Ver ubicación en tiempo real del Transportista | ➖ | Supabase Realtime directo (D-10) sobre `viaje_ubicacion` | `viaje_ubicacion` | — |

## Requisitos funcionales — Transportista

| RF | Título | Estado | Endpoint(s) / servicio | Tablas | Tests |
|----|--------|--------|------------------------|--------|-------|
| RF-16 | Registro de Transportista | ❌ | — | `usuario`, `transportista`, `documento_transportista` | — |
| RF-17 | Ver y ofertar sobre necesidades de servicio | ❌ | — | `solicitud`, `oferta`, `transportista_zona`, `vehiculo` | — |
| RF-18 | Registrar vehículo | ❌ | — | `vehiculo`, `tipo_vehiculo` | — |
| RF-19 | Cancelar viajes | ❌ | — | `viaje`, `transportista` (tasa_cumplimiento) | — |
| RF-20 | Chat con el Cliente | ➖ | Supabase Realtime directo (D-10) | `mensaje` | — |
| RF-21 | Ver información del viaje | ❌ | — | `viaje` | — |
| RF-22 | Realizar el viaje (PIN inicio/fin) | ❌ | — | `viaje`, `viaje_ubicacion` | — |
| RF-23 | Reportar problemas | ❌ | — | `incidente` | — |
| RF-24 | Ver historial de viajes | ❌ | — | `viaje` | — |

---

## Reglas de negocio

| RN | Título | Estado | Dónde se implementa | Verificación |
|----|--------|--------|---------------------|--------------|
| RN-01 | Cálculo automático del precio | ❌ | servicio de cotización (cliente `solicitud` + `oferta`) | test con `config_tarifa` fija; el Cliente nunca manda precio |
| RN-02 | Cálculo automático de cantidad de viajes / cubicaje | ❌ | servicio de cotización, sobre `tipo_vehiculo` | test: N objetos → viajes por tipo de vehículo |
| RN-03 | Comisión de la plataforma | ❌ | servicio de pagos | `%` desde `config_comision` vigente, congelado en `viaje.porcentaje_comision_snapshot` |
| RN-04 | Matchmaking por localidad | ❌ | query de matching | zona del transportista ∈ {origen, destino} de la solicitud; sin ensanchamiento |
| RN-05 | Postulación pull + push + top 3 por score | ❌ | job async (push) + endpoint listado ofertas (top 3) | score = f(precio, calificación, cercanía, tasa_cumplimiento) |
| RN-06 | Verificación por PIN + geolocalización | ❌ | endpoints de ejecución de viaje | 2 PIN (inicio/fin) + ping GPS; `resena` habilitada tras PIN de fin válido |
| RN-07 | Cancelación con costo | ❌ | endpoints de cancelación | sin cargo si no salió; cargo de resarcimiento si salió; transportista no paga penalización |
| RN-08 | Catálogo de objetos comunes | ❌ | endpoint de catálogo + cotización | `solicitud_objeto` admite `objeto_id` o carga manual |

---

## Requisitos no funcionales

| RNF | Estado | Cómo se satisface |
|-----|--------|-------------------|
| RNF-01 — JWT en todos los endpoints | 🟡 | middleware `auth` obligatorio en el mux; ningún handler de negocio sin él |
| RNF-02 — Async y concurrente | 🟡 | `internal/platform/async` (pool de workers); handlers no bloquean en side-effects |
| RNF-03 — 3FN + snapshot en `viaje` | ✅ (en DB) | ya en el esquema; el backend debe **escribir** los `*_snapshot` al confirmar viaje |
| RNF-04 — Mínima decisión del Cliente | 🟡 | precio automático (RN-01), top 3 (RN-05) — se respeta en el diseño de endpoints |

## Requisitos de interfaces

| RI | Estado | Nota |
|----|--------|------|
| RI-01 — Web admin Angular | ➖ | Fuera de alcance de esta etapa. |
| RI-02 — App móvil Flutter | ➖ | Repo `fletway-mobile`. |
| RI-03 — Pasarela de pagos | ❌ | Mercado Pago / Stripe sin decidir (D-09). |
| RI-04 — Geolocalización en tiempo real | ➖ | Supabase Realtime directo (D-10). |
| RI-05 — Chat condicionado a viaje aceptado | 🟡 | La condición "existe `viaje`" la garantiza el backend; el transporte de mensajes es Realtime. |

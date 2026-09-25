# Decisiones técnicas — fletway-backend

> Registro de decisiones de arquitectura. Cada una tiene estado:
> **CONFIRMADA** (acordada con el humano) · **PROPUESTA A CONFIRMAR** (Claude la eligió
> por defecto para poder scaffoldear, falta ratificación) · **ABIERTA** (sin decidir).
>
> Regla: ninguna decisión no reversible se trata como cerrada mientras diga
> "PROPUESTA A CONFIRMAR".

**Última actualización:** 2026-09-24

---

## Índice

| # | Decisión | Estado |
|---|----------|--------|
| D-01 | Stack backend = Go | CONFIRMADA (restricción de la ERS, §2.4) |
| D-02 | Acceso a datos = `pgx` directo + RLS pass-through | CONFIRMADA (elegida por el humano en el prompt de scaffolding) |
| D-03 | Router HTTP = `net/http` `ServeMux` (Go 1.22+), sin framework | PROPUESTA A CONFIRMAR |
| D-04 | Verificación de JWT de Supabase (JWKS asimétrico + fallback HS256) | PROPUESTA A CONFIRMAR |
| D-05 | Path del módulo Go | PROPUESTA A CONFIRMAR |
| D-06 | Procesamiento async (RNF-02) = pool de workers in-process | PROPUESTA A CONFIRMAR |
| D-07 | Organización de paquetes = `platform/` + `feature/<ctx>/` | PROPUESTA A CONFIRMAR |
| D-08 | Migraciones versionadas en `migrations/` + `apply_migration` del MCP | CONFIRMADA (patrón del prompt) |
| D-09 | Pasarela de pagos (Mercado Pago vs Stripe) | ABIERTA (RI-03) |
| D-10 | Realtime (chat RF-10/20, GPS RI-04) lo sirve Supabase directo a la app | CONFIRMADA (modelo híbrido elegido por el humano) |
| D-11 | Formato de contrato de API = REST/JSON con envelope de error único | PROPUESTA A CONFIRMAR |
| D-12 | Testing = `testing` stdlib + `testcontainers`/Supabase local para integración | PROPUESTA A CONFIRMAR |
| D-13 | Modelo de precio (RN-01): sin cotización estimada; precio real por oferta a partir del costo operativo | CONFIRMADA (2026-09-24) |
| D-14 | Cálculo de viajes (RN-02) con `bavix/boxpacker3/v2`, greedy de una pasada, como estimativo | CONFIRMADA (2026-09-24) |
| D-15 | Ubicación de `margen_pct` (config de plataforma vs. por Transportista) | ABIERTA |

---

## D-01 — Stack backend = Go · CONFIRMADA

Restricción dura de la ERS (§2.4, RNF-02). Sin alternativas a evaluar.

---

## D-02 — Acceso a datos: `pgx` directo a Postgres + RLS pass-through · CONFIRMADA

**Decisión:** el backend conecta **directo al Postgres de Supabase** con
`jackc/pgx` + `pgxpool`. En cada request que toca la base:

1. Se abre una transacción.
2. Se ejecuta `SET LOCAL role authenticated`.
3. Se ejecuta `SET LOCAL request.jwt.claims = '<json de claims del JWT del usuario>'`.
4. Se corren las queries del handler.
5. Commit/rollback.

Así, **las políticas RLS ya desplegadas (127 en 39 tablas al 2026-09-24) siguen siendo la capa de seguridad efectiva**
(igual que si las queries pasaran por PostgREST). El backend Go agrega:

- Lógica de negocio y orquestación (cotización RN-01/02, score RN-05, snapshots RNF-03).
- Jobs async (RNF-02).
- Validación de entrada y contratos de API estables para la app.

**Descartado:** conectar con `service_role` y bypassear RLS (duplicaría todas las reglas
en Go, desaprovecha la inversión en RLS). **Descartado:** actuar como proxy fino sobre
PostgREST (menos "backend real", más acoplado).

**Implicancia no reversible:** el diseño de queries y del pool asume que RLS filtra.
Si alguna vez se cambia a `service_role`, hay que reauditar **toda** consulta.

**Pendiente de implementación:** helper en `internal/platform/database/rls.go`. Confirmar
que el rol `authenticated` y el GUC `request.jwt.claims` se comportan igual vía conexión
directa que vía PostgREST (probar contra la base real).

---

## D-03 — Router = `net/http` `ServeMux` · PROPUESTA A CONFIRMAR

Go 1.22 trae routing por método y path params en `ServeMux` (`GET /viajes/{id}`), lo
suficiente para este proyecto sin sumar dependencia. Alternativas razonables: `chi`
(middleware chain más ergonómico), `echo`/`gin` (más pesados).

**Reversible:** cambiar de router es contenido, no arquitectura. Se deja `ServeMux` como
default. Confirmar o pedir `chi`.

---

## D-04 — Verificación de JWT de Supabase · PROPUESTA A CONFIRMAR

Supabase Auth (GoTrue) emite JWT. Proyectos nuevos usan **claves asimétricas (JWKS)**;
proyectos viejos, **HS256 con el JWT secret del proyecto**. Propuesta: verificar contra
el **JWKS endpoint** (`/auth/v1/.well-known/jwks.json`) con cache, y **fallback a HS256**
con `SUPABASE_JWT_SECRET` si el proyecto está en el esquema legacy.

Claims relevantes: `sub` = `usuario.id` (= `auth.uid()`), `role`, `email`, `exp`.

**A confirmar:** ¿el proyecto `dbFletway` usa JWKS o HS256? Verificar vía MCP / panel.

---

## D-05 — Path del módulo Go · PROPUESTA A CONFIRMAR

Se scaffoldeó como **`github.com/fletway/fletway-backend`** (placeholder). Cambiar con
`go mod edit -module <path-real>` cuando se sepa el owner/org del repo remoto.

---

## D-06 — Async (RNF-02) = pool de workers in-process · PROPUESTA A CONFIRMAR

Para "tareas en segundo plano sin bloquear el flujo principal": un pool de goroutines
con cola con buffer en `internal/platform/async`, arrancado en `main` y apagado con
graceful shutdown. Cubre: notificar Transportistas al publicar solicitud (RN-05),
recalcular tasas, side-effects de pagos.

**Alternativas si crece:** `pg` como cola (`SELECT ... FOR UPDATE SKIP LOCKED`),
o Supabase Edge Functions + `pg_cron`. Se evita broker externo (Redis/NATS) por alcance.

**Reversible** mientras los jobs sean idempotentes y estén detrás de una interfaz
`async.Enqueuer`.

---

## D-07 — Organización de paquetes · PROPUESTA A CONFIRMAR

```
cmd/api/                 entrypoint
internal/platform/       infra transversal (config, database, auth, httpx, async, middleware)
internal/server/         wiring: construye el http.Server y registra rutas de cada feature
internal/feature/<ctx>/  un paquete por contexto de negocio:
    routes.go     registra los handlers en el mux
    handler.go    HTTP (decode, validar, responder)
    service.go    lógica de negocio + orquestación async
    repository.go queries pgx (scoped por RLS)
    dto.go        request/response structs (fuente para docs/ENDPOINTS.md)
pkg/                     solo si algo es realmente importable desde afuera (hoy vacío)
```

Contextos previstos (según dominios de la doc DB): `identidad` (usuario/cliente/
transportista/documento), `geografia` (zona), `vehiculo`, `solicitud`, `oferta`, `viaje`,
`pago`, `resena`, `incidente`, `notificacion`, `mensaje`, `catalogo` (tablas de referencia).

**Reversible** con esfuerzo. No crear los 12 paquetes de una: se crean con
`scaffold-endpoint` a medida que se implementan.

---

## D-08 — Migraciones · CONFIRMADA

`.sql` numerados en `migrations/`, aplicados vía `apply_migration` del MCP **con
confirmación humana**. Patrones obligatorios en `.claude/skills/supabase-migration/`.

---

## D-09 — Pasarela de pagos · ABIERTA

RI-03: Mercado Pago **o** Stripe. Sin decidir. Impacta `internal/feature/pago`. El
contrato interno se diseña detrás de una interfaz `pago.Gateway` para no atarse.

---

## D-10 — Realtime lo sirve Supabase directo · CONFIRMADA

Modelo híbrido (elegido por el humano): la app Flutter usa `supabase_flutter` para
**Auth (GoTrue)** y **Realtime** (chat RF-10/RF-20 sobre `mensaje`; GPS en vivo
RI-04/RF-15 sobre `viaje_ubicacion`), apoyándose en RLS. El backend Go **no** implementa
websockets propios para eso. El backend sí valida/procesa: creación del `viaje` que
habilita el chat, validación de PIN, cierre de viaje, etc.

**Implicancia:** las policies RLS de `mensaje` y `viaje_ubicacion` tienen que ser
correctas por sí solas — la app escribe ahí sin pasar por Go. Auditar con
`rls-policy-review` antes de habilitar esas features en la app.

---

## D-11 — Contrato de API · PROPUESTA A CONFIRMAR

REST sobre JSON. Envelope de error único:

```json
{ "error": { "code": "solicitud_no_encontrada", "message": "…", "details": {} } }
```

Respuestas OK devuelven el recurso directo (sin envelope `data`). Paginación por
`?limit=&cursor=`. Todo documentado en `docs/ENDPOINTS.md`, que es la fuente que consume
la skill `sync-api-models` del repo `fletway-mobile`.

---

## D-12 — Testing · PROPUESTA A CONFIRMAR

`testing` de stdlib + `testify` para asserts. Integración contra Postgres real
(Supabase local vía CLI, o `testcontainers-go`). Los tests de RLS son parte de la
suite: cada feature testea que un usuario no puede ver/tocar lo ajeno.

---

## D-13 — Modelo de precio (RN-01) · CONFIRMADA (2026-09-24)

- **No hay cotización estimada** al publicar la `solicitud`. El único precio que ve el
  Cliente es `oferta.precio_calculado`, calculado al ofertar (RF-17) con el `vehiculo` y
  la cantidad de ayudantes reales del Transportista.
- Precio = costo operativo (laboral + vehículo + adicionales) × (1 + margen) /
  (1 − comisión) × (1 + IVA).
- La distancia del Transportista al origen **no** se calcula ni se cobra.
- Costos del vehículo en la tabla privada `vehiculo_costo`, porque `vehiculo` es de
  lectura pública. `vehiculo.peso_maximo_kg` = carga útil.
- Parámetros versionados en `config_costo_laboral`, `config_operacion`,
  `config_impuesto` y `config_comision`. `config_tarifa` queda deprecada.
- El desglose se guarda en `oferta` y el `viaje` lo copia (el `viaje` lo crea la sesión
  del Cliente, que no puede leer `vehiculo_costo`).
- Columnas obsoletas: se deprecan ahora y se borran (DROP) en una migración posterior.

Detalle: `docs/ALGORITMO_COTIZACION.md`. Migraciones `0001`–`0007`.

---

## D-14 — Cálculo de viajes (RN-02) con boxpacker3 v2 · CONFIRMADA (2026-09-24)

Objetivo: un **estimativo** de la cantidad de viajes para el precio, no el mínimo
garantizado. `github.com/bavix/boxpacker3/v2@v2.0.0` con `NewGreedy(OrderDecreasing,
SelectFirstFit)`, `WithFinishers()` vacío y `MinSupportRatio: 0.6`. Aplica de verdad
`rotacion_vertical` y `apilable`. Se descartó la v1.3.2 porque no soporta esas restricciones y
estimaba cerca del doble de viajes. El cálculo corre sobre el `vehiculo` real al ofertar, no
sobre `tipo_vehiculo` como plantea la ERS.

Detalle y mediciones: `docs/ALGORITMO_VIAJES_EMPAQUETADO.md`.

---

## D-15 — Ubicación de `margen_pct` · ABIERTA

¿Parámetro de plataforma o configurable por cada Transportista? Si es por Transportista,
el margen pasa a ser otra variable por la que compiten las ofertas. Hoy el schema sólo
guarda el **valor aplicado** (`oferta.margen_pct`, `viaje.margen_pct_snapshot`), que sirve
para cualquiera de las dos opciones. No se crea ninguna columna de configuración hasta
decidirlo.

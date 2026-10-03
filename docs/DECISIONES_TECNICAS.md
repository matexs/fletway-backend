# Decisiones técnicas — fletway-backend

> Registro de decisiones de arquitectura. Cada una tiene estado:
> **CONFIRMADA** (acordada con el humano) · **PROPUESTA A CONFIRMAR** (Claude la eligió
> por defecto para poder scaffoldear, falta ratificación) · **ABIERTA** (sin decidir).
>
> Regla: ninguna decisión no reversible se trata como cerrada mientras diga
> "PROPUESTA A CONFIRMAR".

**Última actualización:** 2026-10-01 (definiciones para empezar la construcción; ver `docs/PLAN_CONSTRUCCION.md`)

---

## Índice

| # | Decisión | Estado |
|---|----------|--------|
| D-01 | Stack backend = Go | CONFIRMADA (restricción de la ERS, §2.4) |
| D-02 | Acceso a datos = `pgx` directo + RLS pass-through | CONFIRMADA (elegida por el humano en el prompt de scaffolding) |
| D-03 | Router HTTP = `net/http` `ServeMux` (Go 1.22+), sin framework | CONFIRMADA (2026-10-01) |
| D-04 | Autenticación con Supabase Auth; el backend verifica el JWT ES256 vía JWKS, sin fallback HS256 | CONFIRMADA (2026-10-01) |
| D-05 | Path del módulo Go = `github.com/matexs/fletway-backend` | CONFIRMADA (2026-10-01; se aplica en el módulo 1) |
| D-06 | Procesamiento async (RNF-02) = pool de workers in-process | CONFIRMADA (2026-10-01) |
| D-07 | Organización de paquetes = `platform/` + `feature/<ctx>/` | CONFIRMADA (2026-10-01) |
| D-08 | Migraciones versionadas en `migrations/` + `apply_migration` del MCP | CONFIRMADA (patrón del prompt) |
| D-09 | Pasarela de pagos = Mercado Pago (detrás de `pago.Gateway`) | CONFIRMADA (2026-10-01) |
| D-10 | Realtime (chat RF-10/20, GPS RI-04) lo sirve Supabase directo a la app | CONFIRMADA (modelo híbrido elegido por el humano) |
| D-11 | Contrato de API = REST/JSON bajo `/api`, envelope de error único, montos decimales con 2 posiciones | CONFIRMADA (2026-10-01) |
| D-12 | Testing = `testing` stdlib + `testify`, integración contra Supabase local (CLI) | CONFIRMADA (2026-10-01) |
| D-13 | Modelo de precio (RN-01): sin cotización estimada; precio real por oferta a partir del costo operativo | CONFIRMADA (2026-09-24) |
| D-14 | Cálculo de viajes (RN-02) con `bavix/boxpacker3/v2`, greedy de una pasada, como estimativo | CONFIRMADA (2026-09-24) |
| D-15 | `margen_pct` = parámetro de plataforma en `config_margen` (versionada) | CONFIRMADA (2026-10-01) |
| D-16 | Entornos: sólo local (Supabase CLI) y producción (`dbFletway` + Render) | CONFIRMADA (2026-10-01) |
| D-17 | Alcance de esta etapa (Android, sin push, sin web admin, es-AR, sin facturación) | CONFIRMADA (2026-10-01) |
| D-18 | Alta de usuario, rol y primer Administrador | CONFIRMADA (2026-10-01) |
| D-19 | Documentos y adjuntos en Supabase Storage (bucket privado) | CONFIRMADA (2026-10-01) |
| D-20 | Solicitud: fecha de servicio, zona por selector, sin edición, vencimiento y republicación | CONFIRMADA (2026-10-01) |
| D-21 | Matchmaking: qué es un Transportista compatible | CONFIRMADA (2026-10-01) |
| D-22 | Notificaciones: sólo in-app en esta etapa | CONFIRMADA (2026-10-01) |
| D-23 | Reglas y protección de la oferta (desglose en `oferta_costo`/`viaje_costo`) | CONFIRMADA (2026-10-01) |
| D-24 | Ajustes a la fórmula de precio y al cálculo de viajes | CONFIRMADA (2026-10-01) |
| D-25 | Score de recomendación sin cercanía (enmienda a RN-05) | CONFIRMADA (2026-10-01) |
| D-26 | Ejecución del viaje: PIN (`viaje_pin`), GPS, seguimiento, Realtime y chat | CONFIRMADA (2026-10-01) |
| D-27 | Cancelaciones | CONFIRMADA (2026-10-01) |
| D-28 | Pagos con Mercado Pago: retención, split y webhooks | CONFIRMADA (2026-10-01) |
| D-29 | Reseñas, incidentes y reputación | CONFIRMADA (2026-10-01) |
| D-30 | Aplicación del veto | CONFIRMADA (2026-10-01) |
| D-31 | Proceso: revisión de PRs, definición de terminado y casos de prueba | CONFIRMADA (2026-10-01) |
| D-32 | `tipo_vehiculo` con medidas estándar de referencia | CONFIRMADA (2026-10-01) |
| D-33 | Cierre de la revisión de documentación del Transportista | CONFIRMADA (2026-10-03) |

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

## D-03 — Router = `net/http` `ServeMux` · CONFIRMADA

Go 1.22 trae routing por método y path params en `ServeMux` (`GET /viajes/{id}`), lo
suficiente para este proyecto sin sumar dependencia. Alternativas razonables: `chi`
(middleware chain más ergonómico), `echo`/`gin` (más pesados).

**Reversible:** cambiar de router es contenido, no arquitectura.

**Confirmada 2026-10-01:** `ServeMux`, sin `chi` ni otro framework.

---

## D-04 — Autenticación con Supabase Auth y verificación del JWT · CONFIRMADA

Supabase Auth (GoTrue) emite JWT. Proyectos nuevos usan **claves asimétricas (JWKS)**;
proyectos viejos, **HS256 con el JWT secret del proyecto**. Propuesta: verificar contra
el **JWKS endpoint** (`/auth/v1/.well-known/jwks.json`) con cache, y **fallback a HS256**
con `SUPABASE_JWT_SECRET` si el proyecto está en el esquema legacy.

Claims relevantes: `sub` = `usuario.id` (= `auth.uid()`), `role`, `email`, `exp`.

**Confirmada 2026-10-01:** la autenticación es **Supabase Auth** (registro, login y refresh los
hace la app contra GoTrue). El backend sólo **verifica** el JWT: el proyecto firma con **ES256** y
publica sus claves en el JWKS (verificado el 2026-09-26). Se verifica contra el JWKS con cache y
**sin** fallback a HS256 (`SUPABASE_JWT_SECRET` deja de usarse). Se validan `exp`, `aud`
(`authenticated`) e `iss`.

---

## D-05 — Path del módulo Go · CONFIRMADA

Se scaffoldeó como **`github.com/fletway/fletway-backend`** (placeholder).

**Confirmada 2026-10-01:** el módulo pasa a **`github.com/matexs/fletway-backend`** (el repo real).
Se aplica como primera tarea del módulo 1 (`go mod edit -module` + actualizar los imports).

---

## D-06 — Async (RNF-02) = pool de workers in-process · CONFIRMADA

Para "tareas en segundo plano sin bloquear el flujo principal": un pool de goroutines
con cola con buffer en `internal/platform/async`, arrancado en `main` y apagado con
graceful shutdown. Cubre: notificar Transportistas al publicar solicitud (RN-05),
recalcular tasas, side-effects de pagos.

**Alternativas si crece:** `pg` como cola (`SELECT ... FOR UPDATE SKIP LOCKED`),
o Supabase Edge Functions + `pg_cron`. Se evita broker externo (Redis/NATS) por alcance.

**Reversible** mientras los jobs sean idempotentes y estén detrás de una interfaz
`async.Enqueuer`.

**Confirmada 2026-10-01** tal cual. Como el pool vive en memoria, nada que deba sobrevivir a un
reinicio se modela como job (vencimientos y vigencias se calculan al vuelo, D-20 y D-30).

---

## D-07 — Organización de paquetes · CONFIRMADA

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

**Confirmada 2026-10-01** tal cual (ya aplicada en el scaffolding).

---

## D-08 — Migraciones · CONFIRMADA

`.sql` numerados en `migrations/`, aplicados vía `apply_migration` del MCP **con
confirmación humana**. Patrones obligatorios en `.claude/skills/supabase-migration/`.

---

## D-09 — Pasarela de pagos = Mercado Pago · CONFIRMADA

RI-03: Mercado Pago **o** Stripe.

**Confirmada 2026-10-01: Mercado Pago.** Mejor cobertura de medios de pago argentinos (la ERS
§2.5 exige operar en ese mercado) y split de pagos nativo (Marketplace). Se implementa detrás de
la interfaz `pago.Gateway` para no atarse. En desarrollo se usa la cuenta sandbox con sus tarjetas
de prueba. Flujo detallado en D-28.

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

## D-11 — Contrato de API · CONFIRMADA

REST sobre JSON. Envelope de error único:

```json
{ "error": { "code": "solicitud_no_encontrada", "message": "…", "details": {} } }
```

Respuestas OK devuelven el recurso directo (sin envelope `data`). Paginación por
`?limit=&cursor=`. Todo documentado en `docs/ENDPOINTS.md`, que es la fuente que consume
la skill `sync-api-models` del repo `fletway-mobile`.

**Confirmada 2026-10-01**, con dos precisiones:
- Todas las rutas de negocio van bajo el prefijo **`/api`** (así las monta el servidor y así las
  espera la app). Los health checks quedan en la raíz.
- **Montos:** número JSON decimal con **2 posiciones** (no string, no centavos), igual que las
  columnas `numeric` y que la API de Mercado Pago. Redondeo half-up a 2 decimales **una sola vez**,
  en `precioFinal()`, antes de persistir; nunca en pasos intermedios.

---

## D-12 — Testing · CONFIRMADA

`testing` de stdlib + `testify` para asserts. Integración contra Postgres real
(Supabase local vía CLI, o `testcontainers-go`). Los tests de RLS son parte de la
suite: cada feature testea que un usuario no puede ver/tocar lo ajeno.

**Confirmada 2026-10-01:** `testing` + `testify`; integración contra **Supabase local**
(`supabase start`), nunca contra `dbFletway` (D-16).

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

## D-15 — `margen_pct` = parámetro de plataforma · CONFIRMADA (2026-10-01)

**Decisión:** parámetro **de plataforma**, en una tabla nueva `config_margen` versionada como
las demás `config_*` (`margen_pct` 0–100, `vigente_desde`/`vigente_hasta`, lectura para
autenticados y escritura sólo Administrador). No es configurable por Transportista.

**Por qué:** mismo patrón que las otras configuraciones y evita una carrera a la baja de márgenes
entre Transportistas, contraria a la "ganancia justa" que pide RN-01. Se puede revisar más
adelante como funcionalidad de negocio.

El schema ya guarda el valor aplicado (`oferta.margen_pct`, `viaje.margen_pct_snapshot`). La
tabla `config_margen` se crea en el módulo 8 (ver `docs/PLAN_CONSTRUCCION.md`).

---

## D-16 — Entornos: local y producción · CONFIRMADA (2026-10-01)

- **Local:** Supabase CLI (`supabase start`) para desarrollo y tests. Se resetea a voluntad y no
  depende de infraestructura paga.
- **Producción:** el proyecto `dbFletway` y el backend en **Render** (a priori; deploy desde
  GitHub). **El despliegue no es parte de esta etapa:** el objetivo actual es que todo funcione en
  local. Las credenciales y la URL de producción se definen cuando se despliegue.
- **No hay staging.** Ningún test ni desarrollo usa `dbFletway`.
- **Credenciales:** el `.env` real nunca va al repo; los valores se comparten por un gestor de
  contraseñas del equipo; en CI, como GitHub Actions Secrets. Las administra quien es owner del
  proyecto `dbFletway`.

---

## D-17 — Alcance de esta etapa · CONFIRMADA (2026-10-01)

Recortes deliberados (no olvidos). Constan también como enmiendas en `docs/TRAZABILIDAD.md`:

- **Sólo Android.** `ios/` queda generado pero no se mantiene (push y build de iOS requieren Mac y
  APNs).
- **Sin notificaciones push** en esta etapa: sólo notificaciones in-app (D-22).
- **Sin web admin (Angular):** el Administrador opera con endpoints del backend desde Postman
  (D-18).
- **Sólo español (Argentina)**, sin internacionalización.
- **Sin facturación fiscal:** el IVA queda parametrizado en `config_impuesto`; quién factura es un
  tema contable/legal fuera del proyecto.
- **Sin verificación de email ni de teléfono** en el alta (la verificación fuerte es RF-01 para
  Transportistas).
- **Un rol por cuenta:** quien quiera ser Cliente y Transportista usa dos cuentas.
- **Sin APIs de mapas, geocodificación ni ruteo conectadas todavía:** los componentes se construyen
  detrás de una interfaz y el proveedor (Google) se integra después (D-20, D-24).

---

## D-18 — Alta de usuario, rol y primer Administrador · CONFIRMADA (2026-10-01)

- **Alta:** la app hace `signUp` en Supabase Auth con `nombre_completo`, `telefono` y `rol` en la
  metadata. Un **trigger `AFTER INSERT` sobre `auth.users`** crea la fila `usuario`.
  **Regla de seguridad:** el trigger sólo acepta `rol` = `cliente` o `transportista`; cualquier otro
  valor (incluido `administrador`) se rechaza. La metadata la controla el propio usuario.
- Después, el endpoint de registro del backend (`POST /api/auth/registro/cliente` o
  `/transportista`) crea la fila del rol (`cliente` o `transportista`, este en estado `pendiente`).
- **Fuente del rol:** `GET /api/me` devuelve `usuario_id`, `email`, `nombre_completo`, `rol` y, si
  es Transportista, `estado_habilitacion`. La app usa `/me` al loguearse y al arrancar con sesión
  viva; **nunca** el rol de `user_metadata`.
- **Primer Administrador:** no se puede autoregistrar. Se crea una única vez por SQL (usuario en
  Supabase Auth + filas `usuario` y `administrador`), con confirmación humana, como paso de setup
  documentado en `docs/PLAN_CONSTRUCCION.md`. Los siguientes los crea otro Administrador.

---

## D-19 — Documentos y adjuntos en Supabase Storage · CONFIRMADA (2026-10-01)

- Bucket **privado** `documentos-transportista`. Prefijos: `transportista/{usuario_id}/...` para la
  documentación (RF-16) e `incidente/{incidente_id}/...` para los adjuntos de incidentes (D-29).
- Policies de Storage: el dueño hace `INSERT`/`SELECT` sobre su prefijo; el Administrador hace
  `SELECT` sobre todo; sin acceso público.
- Formatos: jpg, png y pdf. Tamaño máximo: 10 MB por archivo.
- `documento_transportista.url_archivo` guarda el **path** del objeto, no una URL. Las URLs se
  piden firmadas y bajo demanda.

---

## D-20 — Solicitud · CONFIRMADA (2026-10-01)

- **Fecha del servicio:** columnas nuevas `fecha_servicio_deseada` (date, obligatoria) y
  `franja_horaria_inicio` / `franja_horaria_fin` (time, opcionales; null = "lo antes posible").
- **Zona:** el Cliente elige la zona de origen y la de destino de un **selector** con el catálogo
  `zona` (el mismo que usa el Transportista). La zona no se deriva de coordenadas.
- **Dirección y coordenadas:** se construye el **componente sin API** (dirección escrita a mano y
  coordenadas provistas por una interfaz `Geocodificador`); el proveedor real (Google Geocoding y
  mapa) se integra más adelante (D-17). *Implementación (módulo 6):* `GEOCODIFICADOR_PROVEEDOR`
  elige el proveedor; en desarrollo, `aproximado` ubica la dirección cerca del centro de su zona
  piloto con un corrimiento determinístico de hasta 1,5 km. Con `APP_ENV=production` el backend no
  arranca si no es `google`, igual que el ruteo (A-6).
- **Sin edición:** para cambiar algo, el Cliente cancela la solicitud
  (`POST /api/solicitudes/{id}/cancelar`, sin costo porque no hay compromiso) y publica otra. Las
  ofertas pendientes de una solicitud cancelada pasan a `no_seleccionada`.
- **Vencimiento al vuelo:** una solicitud `publicada` cuya `fecha_servicio_deseada` ya pasó sin
  viaje confirmado se trata como vencida en las lecturas (no hay job ni `pg_cron`).
- **Republicar:** sobre una solicitud vencida, el Cliente puede **republicarla**
  (`POST /api/solicitudes/{id}/republicar` con una nueva fecha): se crea una solicitud nueva copiando
  los datos y los objetos; la vencida queda como registro.
- **Ayudantes:** `cantidad_ayudantes_solicitados` es **informativo**; no obliga a la oferta. La
  tarjeta de oferta muestra ambos números.

---

## D-21 — Matchmaking: Transportista compatible · CONFIRMADA (2026-10-01)

Una solicitud es compatible con un Transportista si se cumplen las tres condiciones:

1. **Zona:** `origen_zona_id` o `destino_zona_id` está entre sus zonas de trabajo (RN-04).
2. **Capacidad:** al menos un vehículo **activo** pasa la cota rápida de peso y volumen
   (`ALGORITMO_VIAJES_EMPAQUETADO.md` §4, paso 1), sin correr el empaquetado completo.
3. **Disponibilidad:** `transportista.disponible = true`, entendido como un interruptor manual
   ("estoy tomando trabajos"), no ligado a fechas.

Además, sólo Transportistas `habilitado` y sin veto vigente.

---

## D-22 — Notificaciones sólo in-app en esta etapa · CONFIRMADA (2026-10-01)

- No se implementa push (D-17). Toda notificación es una fila en `notificacion` que la app muestra
  (RF-09). El aviso de "solicitud compatible" (RN-05) también es in-app, y el Transportista además
  ve el listado de solicitudes compatibles.
- **Arquitectura preparada para push:** el envío pasa por una interfaz `Notificador` del backend
  (hoy, una implementación que sólo inserta en `notificacion`). Para sumar push más adelante (FCM)
  alcanza con otra implementación más una tabla de tokens de dispositivo.
- Sin preferencias de usuario sobre qué recibir.
- Textos de los 9 tipos de `tipo_notificacion`: en `docs/PLAN_CONSTRUCCION.md` §3.

---

## D-23 — Reglas y protección de la oferta · CONFIRMADA (2026-10-01)

- **No editable:** el Transportista la retira (`retirada`) y crea otra.
- **Sin vencimiento propio:** vive hasta que el Cliente elige o la solicitud vence.
- **Varias ofertas por Transportista** a una misma solicitud, una por vehículo (lo permite el
  `UNIQUE` actual).
- **Al aceptar una oferta**, las demás de esa solicitud pasan a `no_seleccionada` en la misma
  transacción que crea el `viaje`.
- **Máximo 3 ayudantes** por oferta (validación del endpoint, 400 si se piden más). Con ese tope la
  fórmula de eficiencia siempre mejora con cada ayudante.
- **Protección contra modificación:** trigger `trg_proteger_campos_oferta` (patrón de
  `trg_proteger_campos_viaje`) que impide a Cliente y Transportista hacer UPDATE de las columnas de
  costo y precio y de las FK estructurales. Sólo pueden cambiar el estado.
- **Ocultar el desglose al Cliente:** como Cliente y Transportista usan el mismo rol de Postgres
  (`authenticated`), un GRANT por columna no sirve. El desglose de costos vive en tablas propias:
  - **`oferta_costo`** (1 a 1 con `oferta`): distancia, duraciones, costo laboral, costo del
    vehículo, costos adicionales, costo operativo, margen, precio neto, comisión e IVA. RLS: sólo el
    Transportista dueño y el Administrador.
  - **`viaje_costo`** (1 a 1 con `viaje`): los snapshots de esos mismos valores. RLS: sólo el
    Transportista del viaje y el Administrador.
  - `oferta` y `viaje` conservan lo que ve el Cliente: precio final, cantidad de viajes y de
    ayudantes. Las columnas de costo agregadas en `0005` y `0006` se mueven a las tablas nuevas
    (están vacías).
  - **`fn_aceptar_oferta(oferta_id)`** (`SECURITY DEFINER`): verifica que quien llama sea el
    Cliente de la solicitud y, en una transacción, crea el `viaje`, copia `oferta_costo` a
    `viaje_costo`, genera los PIN (D-26), pasa las demás ofertas a `no_seleccionada` y la
    solicitud a `asignada`. Hace falta porque la sesión del Cliente no puede leer `oferta_costo`.

**Implementación (módulo 8, migración `0016`, 2026-10-03):** para que "se retira y se crea otra"
valga también con el mismo vehículo, el `UNIQUE (solicitud, transportista, vehículo)` pasó a un
índice único parcial que ignora las ofertas `retiradas`. Una oferta sin su fila en `oferta_costo` se
rechaza al hacer COMMIT (trigger diferido), así que sólo el backend, que calcula el precio, crea
ofertas. `POST /api/solicitudes/{id}/ofertas/cotizar` muestra el precio antes de ofertar, sin
guardar nada.

---

## D-24 — Ajustes a la fórmula de precio y al cálculo de viajes · CONFIRMADA (2026-10-01)

Cierra las ambigüedades de `ALGORITMO_COTIZACION.md` §4.3/§4.4 y de
`ALGORITMO_VIAJES_EMPAQUETADO.md` §6:

- `tiempo_espera_min` **se suma una vez por viaje** dentro del tiempo de operación.
- **Horas de ida y vuelta:** `h = DuracionRutaH * 2 + operacionH` (antes sólo se cobraba la ida).
- **Costos adicionales una vez por oferta.**
- **ART 10 %** (valor de la seed; el 18 % del documento fuente corresponde a contribuciones de
  seguridad social).
- **Valores de configuración:** las filas vigentes de `config_*` son los valores de trabajo para
  desarrollar y probar. Reemplazarlos por cifras definitivas es un `UPDATE`, no un cambio de diseño.
- **Ruteo:** interfaz `Ruteador` con tres proveedores, elegidos con `RUTEO_PROVEEDOR`:
  - `google`: **Google Distance Matrix**, para producción, integrado más adelante (D-17);
  - `aproximado`: **sólo local**, distancia en línea recta × 1,3 y duración a 30 km/h, para probar
    el flujo de punta a punta sin API;
  - `fijo`: **sólo tests**, valores fijos para resultados reproducibles.

  Si `APP_ENV=production` y el proveedor no es `google`, el backend **no arranca**. En producción,
  si la ruta no se puede obtener, `ErrRutaNoDisponible` y la oferta se rechaza con error de
  validación: nunca se estima la distancia "a ojo".
- **Cálculo de viajes:** `maxViajes = 20`; `context.WithTimeout` de **5 s** alrededor de
  `planificarViajes`; `MinSupportRatio = 0.6` fijo en el código.

---

## D-25 — Score de recomendación sin cercanía · CONFIRMADA (2026-10-01)

- **Enmienda a RN-05:** la "cercanía" se **elimina** del score y de todo algoritmo. El
  Transportista no tiene ubicación guardada y ya se decidió no calcular su distancia al origen.
- Fórmula en `docs/ALGORITMO_SCORE.md`: precio, calificación y tasa de cumplimiento, con los pesos
  de la propuesta renormalizados al quitar la cercanía.
- **ERS corregida (2026-10-01):** `docs/ERS_Fletway.docx` ya no menciona la cercanía en RN-05 ni en
  el glosario, y suma la sección "5. Anexo — Registro de cambios" con todas las desviaciones
  decididas. El `.docx` reemplaza al PDF en el repo: es la fuente editable.

---

## D-26 — Ejecución del viaje · CONFIRMADA (2026-10-01)

- **PIN (RN-06):** el **Transportista nunca puede ver el PIN**. Se lo pide al Cliente antes de
  empezar el servicio (PIN de inicio) y al finalizar (PIN de fin), como define la ERS. El Cliente ve
  ambos PIN desde que se confirma el viaje. En RF-21, "el PIN correspondiente" se interpreta como el
  campo para cargarlo, no su valor.
  - 4 dígitos numéricos, generados al confirmar el viaje.
  - 5 intentos fallidos: se bloquea la carga y se abre un incidente automático para el Administrador.
  - **Dónde vive:** tabla **`viaje_pin`** (1 a 1 con `viaje`) con `pin_inicio`, `pin_fin` y los
    contadores de intentos. RLS: sólo la leen el Cliente del viaje y el Administrador; nadie la
    escribe directamente. Se eliminan `viaje.pin_inicio` y `viaje.pin_fin` (tabla vacía).
  - **Generación:** la hace `fn_aceptar_oferta` (D-23) al confirmar el viaje.
  - **Validación:** **`fn_validar_pin(viaje_id, tipo, pin)`** (`SECURITY DEFINER`) verifica que
    quien llama sea el Transportista del viaje, compara, registra el intento y, si acierta, pasa el
    viaje a `en_curso` (inicio) o `finalizado` (fin). Al quinto fallo bloquea la carga y abre el
    incidente. Sólo devuelve "correcto" o "incorrecto", nunca el valor. Hace falta porque con RLS
    pass-through el backend corre con la sesión del Transportista.
- **GPS:** radio de tolerancia **150 m**, precisión mínima **≤ 50 m**. Si no se cumple, el PIN **no**
  se rechaza: se registra la discrepancia.
- **Salida:** columna nueva `viaje.salio_en`, seteada por `POST /api/viajes/{id}/salida` cuando el
  Transportista sale hacia el origen. No se agrega un estado nuevo.
- **Seguimiento:** pings cada 15 a 30 s desde el PIN de inicio hasta el PIN de fin o la cancelación;
  retención de 30 días desde `finalizado_en` (limpieza manual mientras no haya `pg_cron`). Android
  pide ubicación en segundo plano con notificación de servicio en primer plano.
- **Realtime:** `ALTER PUBLICATION supabase_realtime ADD TABLE mensaje, viaje_ubicacion`, después
  de correr `rls-policy-review` y sumar la condición de veto (D-30).
- **Chat:** no se cierra al terminar el viaje; sin adjuntos; el Administrador lo puede leer ante un
  incidente.
- **Sin conexión:** la validación del PIN requiere red (la app reintenta); los pings de GPS se
  encolan en el dispositivo y se reenvían al volver la señal.

---

## D-27 — Cancelaciones · CONFIRMADA (2026-10-01)

- **"Salió"** = `viaje.salio_en IS NOT NULL` (D-26), no se infiere del GPS.
- **Cancelación del Cliente:** sin cargo si no salió; si salió, cargo de resarcimiento =
  **20 % de `precio_calculado`**. El porcentaje se informa al Cliente antes de confirmar.
- **Cancelación del Transportista:** sin penalización económica; baja la tasa de cumplimiento
  (D-29); la `solicitud` vuelve a `publicada`; reembolso total al Cliente.
- **Columnas nuevas en `viaje`:** `cancelado_en`, `cancelado_por_usuario_id`,
  `motivo_cancelacion` (opcional) y `cargo_resarcimiento_monto` (nullable).

---

## D-28 — Pagos con Mercado Pago · CONFIRMADA (2026-10-01)

- **Retención al aceptar la oferta** (captura y `retenido`) y **liberación al validar el PIN de
  fin** (`liberado`), salvo que haya un incidente abierto (D-29).
- **Split (Marketplace):** el Transportista vincula su cuenta de Mercado Pago (OAuth) antes de
  poder ofertar; se guarda su identificador de cuenta (columna o tabla nueva). La comisión se
  descuenta con `marketplace_fee`, usando `config_comision` vigente al aceptar.
- **Webhook:** `POST /api/webhooks/pagos/mercadopago`, **excepción explícita a RNF-01** (un sistema
  externo no envía JWT de usuario), protegido por validación de firma (`x-signature`) e
  idempotencia (índice único sobre el id de evento externo).
- **Reembolsos y liberaciones parciales:** sólo desde la resolución de incidentes (RF-03), nunca
  directo por Cliente o Transportista. El cargo de cancelación se modela como reembolso del 80 %
  reteniendo el 20 %.

---

## D-29 — Reseñas, incidentes y reputación · CONFIRMADA (2026-10-01)

- **Reseña:** plazo de **14 días** desde `finalizado_en` (error `plazo_resena_vencido`);
  inmutable.
- **Incidentes:** admiten adjuntos (fotos, bucket de D-19); sin plazo para reportar; un incidente
  `abierto` o `en_revision` **congela la liberación** del pago hasta que el Administrador lo
  resuelva.
- **`calificacion_promedio`:** promedio simple de `resena.calificacion`, recalculado por trigger
  `AFTER INSERT ON resena`. Inicial: `NULL` ("sin reseñas").
- **`tasa_cumplimiento`:** `finalizados / (finalizados + cancelados_por_transportista) * 100` sobre
  todo el histórico, recalculada por trigger cuando un viaje pasa a `finalizado` o
  `cancelado_transportista`. Inicial: **100**.

---

## D-30 — Aplicación del veto · CONFIRMADA (2026-10-01)

- **Backend:** un middleware, en cada request autenticado, verifica si hay un veto vigente y
  responde 403 con code `cuenta_vetada`.
- **Canales directos de la app** (`mensaje`, `viaje_ubicacion`): condición de veto en sus policies
  RLS, agregada antes de habilitar Realtime.
- **Vigencia al vuelo:** veto definitivo, o temporal con `fecha_fin > now()`. No se guarda un flag
  que necesite un job para vencer.

---

## D-31 — Proceso · CONFIRMADA (2026-10-01)

- **Revisión de PRs:** 1 aprobación obligatoria del otro integrante antes de mergear a `main`,
  configurada como *branch protection rule* en GitHub.
- **Definición de terminado:** un RF/RN pasa a `OK` en `docs/TRAZABILIDAD.md` sólo con (1) tests
  unitarios de los cálculos puros con casos límite, (2) tests del endpoint (camino feliz,
  validación y RLS) y (3) su caso de prueba en `qa/casos-prueba/`.
- **Casos de prueba:** en Markdown o CSV dentro del repo (RF/RN, precondición, pasos, resultado
  esperado), escritos por quien implementa, en el mismo PR.

---

## D-32 — `tipo_vehiculo` con medidas estándar · CONFIRMADA (2026-10-01)

- `tipo_vehiculo` suma `largo_estandar_m`, `ancho_estandar_m` y `alto_estandar_m`;
  `volumen_estandar_m3` queda deprecada.
- Las 4 filas actuales se reemplazan por 6 tipos: Utilitario, Furgón chico, Furgón grande,
  Camión chico, Camión mediano y Camión grande (valores en `docs/PLAN_CONSTRUCCION.md` §2.3).
- **Son medidas de referencia:** la app las propone al registrar un vehículo y el Transportista
  las corrige. El cálculo de viajes (D-14) y el matchmaking (D-21) usan siempre las medidas del
  **vehículo real** (`vehiculo.largo_util_m`, `ancho_util_m`, `alto_util_m`).
- Se aplica en el módulo 4.

---

## D-33 — Cierre de la revisión de documentación · CONFIRMADA (2026-10-03)

RF-01 dice que el Administrador aprueba o rechaza la habilitación; el plan lo resuelve por
documento. Esta decisión fija cuándo cambia el estado de la cuenta.

- **Documentos obligatorios:** los cuatro de `tipo_documento` (DNI, seguro, registro y VTV), los
  que enumera la ERS. Cuenta el **último** documento cargado de cada tipo.
- **Rechazo inmediato:** apenas el Administrador rechaza un documento, el Transportista pasa a
  `rechazado` y recibe la notificación `documentacion_revisada` con el motivo. No se espera a
  revisar los demás.
- **Nueva carga:** si estaba `rechazado` y vuelve a cargar el documento rechazado, pasa a
  `pendiente`. Sin límite de reintentos (RF-01).
- **Habilitación:** cuando el último documento de los cuatro tipos está aprobado, pasa a
  `habilitado` y se le notifica. Aprobar documentos sueltos no notifica.
- **Renovación:** un Transportista `habilitado` que carga un documento nuevo sigue habilitado
  mientras espera la revisión; si se lo rechazan, pasa a `rechazado`.
- **Dónde vive:** la base deriva el estado de los documentos (`trg_recalcular_habilitacion`,
  migración `0012`), así que ni la API ni PostgREST pueden saltearlo.
- **Archivos para el Administrador:** el listado de revisión trae una URL firmada por documento,
  que vence a los 10 minutos (D-19), pedida a Storage con el JWT del Administrador.

# CLAUDE.md — fletway-backend

> Contexto persistente del repo. Cualquier sesión de Claude Code debe poder arrancar
> leyendo **solo este archivo** + `docs/ESTADO_PROYECTO.md` + `docs/DECISIONES_TECNICAS.md`,
> sin releer la ERS entera.

---

## 1. Rol del agente en este repo

Sos el agente de desarrollo del **backend Go** de Fletway (proyecto final de Ingeniería
en Sistemas de Información, UTN FRD). Este repo es el **dueño de la lógica de negocio
central y de la fuente de verdad del esquema de datos** (la doc de la base vive acá, en
`docs/DOCUMENTACION_BASE_DE_DATOS.md`).

Responsabilidades:

- Implementar los endpoints REST que consumen la app móvil (Flutter) y —más adelante,
  fuera de alcance hoy— la web admin (Angular).
- Mantener la trazabilidad **RF/RN de la ERS → endpoint/servicio Go** (`docs/TRAZABILIDAD.md`).
- Generar y versionar migraciones de Supabase siguiendo los patrones validados
  (skill `supabase-migration`), **nunca aplicándolas sin confirmación humana explícita**.
- Mantener actualizados `docs/ESTADO_PROYECTO.md` y `docs/DECISIONES_TECNICAS.md`.

Lo que **no** hacés acá:

- No tocás `admin-web` / Angular: está **fuera de alcance** hasta que la app móvil funcione.
- No ejecutás migraciones, `ALTER`, `DROP` ni cambios de política RLS sobre Supabase sin
  que un humano lo confirme en el momento (ver `.claude/mcp-notes.md`).
- No inventás RF, RN, RNF, RI ni tablas que no estén en la ERS o en la doc de base de datos.

---

## 2. El negocio en 6 líneas

Fletway es un **marketplace two-sided de fletes y mudanzas**. El **Cliente** publica una
`solicitud` de traslado (**sin cotización estimada**: no se muestra ningún monto al publicar)
y el sistema notifica a los **Transportistas** compatibles por zona/vehículo/disponibilidad.
Cada Transportista habilitado se **postula** voluntariamente con una `oferta` (modelo *pull*,
sin asignación forzada), cuyo precio calcula el sistema con su vehículo y ayudantes reales.
El Cliente ve un **top 3 por score de recomendación** y elige una: se confirma un `viaje`
con **snapshot histórico** del desglose de precio y datos. El viaje se ejecuta con
**verificación por PIN (inicio y fin) + GPS**, se cobra un `pago` con **comisión de
plataforma**, y el Cliente deja una `resena`. Cualquier parte puede abrir un `incidente`
que un **Administrador** resuelve (reembolso/liberación total o parcial, veto, o cierre
sin acción). El chat entre Cliente y Transportista se habilita **solo tras confirmar el viaje**.

---

## 3. Reglas de negocio críticas — NO violar

Fuente: `docs/ERS_Fletway.pdf` §3.2. Todo endpoint que las toque debe respetarlas y
declararlo en `docs/TRAZABILIDAD.md`.

| RN | Qué exige | Dónde impacta en el backend |
|----|-----------|------------------------------|
| **RN-01** | Precio **automático** del servicio en función de distancia, tiempo, cantidad de viajes, peso/volumen, demanda, escalera/altura, ayudantes y tarifa base. Diseño que priorice **ganancia justa** para el Transportista. **Decisión de producto:** sólo se calcula el precio de cada oferta (RF-17); **no hay cotización estimada** en RF-06. | Servicio de cotización al crear la `oferta` (`docs/ALGORITMO_COTIZACION.md`). El Cliente **no** ingresa precio. Costo operativo real (laboral + vehículo) + margen + comisión + IVA. `config_tarifa` queda **deprecada**. Sin demanda ni tramo de acercamiento. Ubicación de `margen_pct`: **abierta**. |
| **RN-02** | Cantidad de viajes y cubicaje calculados **automáticamente** por el sistema. La ERS lo plantea por `tipo_vehiculo` al publicar; **en el diseño actual se calcula sobre el `vehiculo` real al ofertar** (ver §7 de `docs/ALGORITMO_VIAJES_EMPAQUETADO.md`). | `planificarViajes` con boxpacker3 **v2** (`github.com/bavix/boxpacker3/v2`), greedy de una pasada: **estimativo para el precio, no el mínimo de viajes**. Si la carga no entra → error de validación, sin oferta. |
| **RN-03** | La plataforma cobra **comisión sobre cada pago** procesado entre Cliente y Transportista. | Servicio de pagos. `config_comision` **vigente**; el % se congela en `viaje.porcentaje_comision_snapshot`. |
| **RN-04** | Matchmaking **por localidad**: emparejar `solicitud` con Transportistas cuya zona de trabajo coincida con **origen o destino**. | Query de matching sobre `transportista_zona` + `solicitud.origen_zona_id`/`destino_zona_id`. Sin ensanchamiento de radio ni ventanas de tiempo (descartado, ver doc DB §6). |
| **RN-05** | Postulación **pull** con **notificación push proactiva** al publicar; el Transportista se postula si quiere (sin obligación). Al Cliente se le muestra un **top 3 por score** (precio + calificación + cercanía + tasa de cumplimiento), con opción "ver más". | Job async de notificación (RNF-02) + endpoint de listado de ofertas que devuelve top 3 ordenado por score. |
| **RN-06** | Verificación por **PIN + geolocalización**: se genera un PIN por viaje; el Transportista carga PIN al **llegar a cargar** y un **segundo PIN al finalizar**; se registra su ubicación GPS. Con **PIN de fin válido** se habilita la `resena` del Cliente (RF-12). | Endpoints de ejecución de viaje (RF-22). `resena` solo si `viaje` en estado que la ERS/doc DB fija como habilitante. |
| **RN-07** | Cancelación con costo: Cliente cancela **sin cargo** si el Transportista **no salió**; **con cargo de resarcimiento** si ya salió (RF-08). El Transportista que cancela (RF-19) **no** paga penalización económica, pero baja su `tasa_cumplimiento`. | Endpoints de cancelación (RF-08, RF-19). |
| **RN-08** | Catálogo de **objetos comunes** (`objeto`) con peso/volumen estimados, seleccionables al publicar la solicitud (RF-06) y usados en la cotización (RN-01, RN-02). También se admite carga manual con dimensiones propias. | `solicitud_objeto` soporta `objeto_id` (catálogo) **o** `nombre_personalizado`; en ambos casos guarda peso + largo/ancho/alto + flags de rotación/apilado. |

> **Schema de cotización (aplicado el 2026-09-24, `migrations/0001`–`0007`):** dimensiones en
> `objeto`/`solicitud_objeto`/`vehiculo`, `vehiculo_costo` (privada), `config_costo_laboral`,
> `config_operacion`, `config_impuesto`, desglose de costo en `oferta` y sus `*_snapshot` en `viaje`.
> Columnas y tablas deprecadas, riesgos de RLS y decisiones abiertas: `docs/ALGORITMO_COTIZACION.md` §8.

Requisitos no funcionales que condicionan **toda** decisión de arquitectura:

- **RNF-01** — JWT obligatorio en **todos** los endpoints (Cliente y Transportista).
- **RNF-02** — Backend **asíncrono y concurrente**: tareas en segundo plano (notificaciones
  push, recálculos, side-effects de pagos) **no** bloquean el request principal.
- **RNF-03** — 3FN estricta en tablas maestras; **snapshot histórico** intencional en `viaje`.
- **RNF-04** — Mínima carga de decisión para el Cliente (precio automático, top 3).

---

## 4. Arquitectura (resumen — detalle en `docs/DECISIONES_TECNICAS.md`)

- **Go** `net/http` (`ServeMux` de Go 1.22+), sin framework web. Módulo: ver `go.mod`.
- **Acceso a datos:** `pgx`/`pgxpool` **directo a Postgres de Supabase**, con
  **RLS pass-through**: en cada request se setea `SET LOCAL role authenticated` y
  `SET LOCAL request.jwt.claims = '<claims del JWT del usuario>'` dentro de la transacción,
  de modo que **las políticas RLS ya desplegadas (127 en 39 tablas al 2026-09-24) siguen
  siendo la capa de seguridad efectiva**. El backend agrega lógica de negocio y orquestación encima, no la reemplaza.
- **Auth:** se verifica el JWT emitido por Supabase Auth (GoTrue). `auth.uid()` = `usuario.id`.
- **Async (RNF-02):** pool de workers en proceso (`internal/platform/async`) para jobs
  disparados desde los handlers (ej. notificar Transportistas al publicar solicitud).
- **Capas por feature:** `internal/feature/<ctx>/` con `routes.go` + `handler.go` +
  `service.go` + `repository.go`. Infra compartida en `internal/platform/`.

> **Atención:** Varias de estas decisiones están marcadas **"propuesta a confirmar"** en
> `docs/DECISIONES_TECNICAS.md`. No las trates como cerradas hasta que el humano las confirme.

---

## 5. Convenciones

### Commits (Conventional Commits + ref a requisito)

```
<tipo>(<scope>): <resumen imperativo>   [RF-XX | RN-XX]

<cuerpo opcional>
```

- `tipo` ∈ `feat` `fix` `refactor` `test` `docs` `chore` `build` `ci` `perf`.
- `scope` = feature o área (`solicitud`, `oferta`, `viaje`, `pago`, `auth`, `db`, `mcp`…).
- Cuando el commit implementa o modifica un requisito, **citá el identificador** entre
  corchetes al final del subject (`[RF-06]`, `[RN-01]`, admite varios: `[RF-06, RN-02]`).
- Ejemplo: `feat(oferta): cálculo de precio y viajes al ofertar [RF-17, RN-01, RN-02]`
- Subject en imperativo, en español, con minúscula inicial, sin punto final y de hasta ~72
  caracteres. El cuerpo explica **por qué** se hizo el cambio, no qué líneas cambiaron.
- Un commit = una unidad lógica. No mezclar refactor, feature y formato en el mismo commit.
- **Sin emojis** en el subject ni en el cuerpo (ver "Prohibición de emojis").

### Branches

- `main` — protegida, siempre desplegable.
- `feat/RF-06-publicar-solicitud` · `feat/RN-01-cotizacion` · `fix/RF-22-validacion-pin`
- Formato: `<tipo>/<RF|RN>-<nn>-<slug-corto>`. Trabajo sin requisito asociado:
  `chore/<slug>` o `docs/<slug>`.

### Estilo y formato

- Base: [Effective Go](https://go.dev/doc/effective_go) y
  [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). Ante la duda, gana lo
  que diga Effective Go.
- **`gofmt` + `goimports` obligatorios.** El CI rechaza archivos sin formatear. Formatear con
  `make fmt`; instalar el hook con `make hooks`.
- **`golangci-lint` v2** con la config de `.golangci.yml`. Linters activos, además del set por
  defecto (`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`):

  | Linter | Qué exige |
  |---|---|
  | `bodyclose` | cerrar todo `http.Response.Body` (JWKS, pasarela de pagos) |
  | `contextcheck` | propagar el `context.Context` recibido, no crear uno nuevo |
  | `errorlint` | envolver con `%w` y comparar con `errors.Is` / `errors.As` |
  | `gosec` | chequeos básicos de seguridad |
  | `revive` | estilo general (la regla `exported` está desactivada, ver "Documentación godoc") |
  | `unconvert` | sin conversiones de tipo innecesarias |
  | `whitespace` | sin líneas en blanco sobrantes al inicio o fin de bloques |

- Un `//nolint` sólo con el linter explícito y el motivo: `//nolint:gosec // <por qué>`.
  Nunca `//nolint` a secas ni desactivar un linter en `.golangci.yml` para esquivar un hallazgo.
- Antes de cada commit: `make check` (fmt + vet + lint + test). Es lo mismo que corre el CI.
- Contexto: **todo** método que hace I/O recibe `context.Context` como primer parámetro.

### Estructura de paquetes y capas

```
cmd/api/                 solo wiring: config, logger, pool, server, graceful shutdown
internal/platform/       infra transversal (config, database, auth, httpx, async, middleware)
internal/server/         arma el http.Server y registra las rutas de cada feature
internal/feature/<ctx>/  un paquete por contexto de negocio (dominios de la doc de base de datos)
    routes.go     Register(mux, deps...): monta las rutas
    handler.go    HTTP: decodifica, valida forma, llama al service, responde
    service.go    reglas de negocio y orquestación; encola jobs async
    repository.go queries SQL, siempre dentro de db.WithinTx (RLS pass-through, D-02)
    dto.go        structs de request/response (fuente de docs/ENDPOINTS.md)
pkg/                     solo lo que deba importarse desde fuera del módulo (hoy vacío)
```

Dependencias permitidas, siempre en un solo sentido:
`handler -> service -> repository -> platform/database`.

Lo que **no** se mezcla:

- **Handler:** no ejecuta SQL, no importa `pgx` y no contiene reglas de negocio. Sólo traduce
  HTTP a una llamada al service y la respuesta a JSON.
- **Service:** no recibe ni escribe `*http.Request` / `http.ResponseWriter`. Devuelve errores
  de negocio como `*httpx.APIError` o como errores sentinel del paquete.
- **Repository:** sólo acceso a datos. Sin reglas de negocio, sin armar respuestas HTTP y sin
  queries fuera de `db.WithinTx`.
- **`internal/platform/`:** no importa nada de `internal/feature/`.
- **Features:** una feature no importa el repository de otra. Si necesita algo, lo pide a
  través del service de la otra feature.
- Los cálculos de negocio (cotización RN-01, empaquetado RN-02, score RN-05) son **funciones
  puras sin I/O**, en su propio archivo dentro de la feature (ej. `oferta/cotizacion.go`), para
  poder testearlas sin base. El service lee los datos y se los pasa.

### Manejo de errores y logging

- Al cruzar una capa, envolver con `fmt.Errorf("<operación>: %w", err)`. La operación va en
  minúscula, en español y sin punto final: `fmt.Errorf("crear oferta: %w", err)`.
- Comparar errores con `errors.Is` / `errors.As`. Nunca comparar el texto de `err.Error()`.
- Errores de dominio esperables como variables sentinel `ErrXxx` en el paquete que los produce
  (ej. `ErrNoFactible`). Lo que llega al cliente se traduce a `*httpx.APIError` con un `code`
  estable en snake_case español (`solicitud_no_encontrada`, `pin_invalido`).
- **Dónde se loguea:** una sola vez, en el borde.
  - `httpx.Error` loguea los errores no tipados (que terminan en 500).
  - El pool async loguea los jobs que fallan.
  - `cmd/api` loguea los errores de arranque y apagado.
- **Dónde se propaga:** services y repositories **no** loguean; devuelven el error envuelto.
  Nunca loguear un error y además devolverlo (queda duplicado).
- Nunca `panic` en el path de un request (el middleware `recover` lo captura, pero no es la vía
  normal). `os.Exit` / `log.Fatal` sólo en `main`, al arrancar.
- Logging estructurado con `log/slog`: mensaje corto en español y en minúscula, datos como pares
  clave-valor (`"job", job.Name, "err", err`). Nunca loguear secretos ni datos sensibles: JWT,
  PIN, tokens, credenciales ni datos de pago.
- Al cliente nunca se le devuelve el texto de un error interno. `httpx.Error` ya responde un 500
  genérico para cualquier error que no sea `*httpx.APIError`.

### Documentación godoc

- **Obligatorio** en todo paquete (comentario `// Package xxx ...`) y en todo tipo, función,
  método, constante y variable **exportados**.
- Idioma español, en formato godoc: la primera oración empieza con el nombre del identificador y
  dice qué hace o qué representa (`// Register monta GET /healthz y GET /readyz ...`).
- Lo mínimo que tiene que explicar cada comentario:
  - **Tipos:** qué representa y, si aplica, los invariantes o el valor cero útil.
  - **Funciones y métodos:** qué hace; los parámetros que no sean obvios; qué devuelve y en qué
    casos devuelve error (nombrar los sentinel, ej. `ErrNoFactible`); efectos secundarios
    (escribe en la base, encola un job, abre una transacción); y qué necesita del contexto
    (identidad para RLS).
  - **Reglas de negocio:** si implementa un RF/RN o una decisión, citarlo (`RN-01`, `D-14`).
- Grupos de declaraciones: cada identificador exportado lleva su propio comentario. Un comentario
  sobre el grupo no alcanza.
- Los comentarios de código no exportado explican el **por qué**, no el qué.
- Ejemplos de código dentro del comentario: indentados con tab, como en
  `internal/platform/database`.
- **Control:** la regla `exported` de `revive` está desactivada en `.golangci.yml`, así que el
  lint no lo verifica. Se controla en code review: un PR con exportados sin documentar no se
  aprueba. Deuda conocida: en `internal/platform/httpx`, `ErrorDetail` y los constructores de
  `APIError` todavía no tienen comentario propio.

### Nombres

- **Paquetes:** minúscula, una sola palabra, sin guiones bajos ni plurales (`oferta`, `httpx`,
  `async`). El paquete de una feature se llama como su dominio.
- **Archivos:** snake_case en minúscula (`routes.go`, `cotizacion.go`, `oferta_test.go`).
- **Identificadores:** `MixedCaps` / `mixedCaps`, sin guiones bajos. Siglas en mayúscula
  consistente: `ID`, `URL`, `JSON`, `HTTP`, `JWT`, `RLS`, `PIN`.
- **Idioma:** los conceptos de negocio en español, sin traducir (`Solicitud`, `Oferta`,
  `precioCalculado`, `planificarViajes`). Los términos técnicos de Go y de infraestructura pueden
  quedar en inglés (`Handler`, `Repo`, `Register`, `ctx`, `err`).
- **Base de datos:** nombres de tablas/columnas **exactamente** como en
  `docs/DOCUMENTACION_BASE_DE_DATOS.md` (español, snake_case). No traducir identificadores de la
  base al inglés.
- **Receptores:** una o dos letras, iguales en todos los métodos del tipo (`h`, `s`, `r`).
- **Constructores:** `NewXxx`. **Getters:** sin prefijo `Get` (`Nombre()`, no `GetNombre()`).
- **Interfaces:** por comportamiento (`Enqueuer`). **Errores:** sentinel `ErrXxx`, tipos
  `XxxError`.
- **Códigos de error de la API:** snake_case en español (`transportista_no_habilitado`).

### Testing

- `testing` de la stdlib; asserts e integración según D-12. Los tests viven junto al código, en
  archivos `*_test.go` del mismo paquete.
- **Table-driven** para toda función con más de un caso: un slice de structs con un campo
  `name`, cada caso en un subtest `t.Run(tc.name, ...)`, y `t.Parallel()` cuando los casos no
  comparten estado.
- **Test obligatorio** para:
  - toda función de cálculo de negocio (cotización RN-01, empaquetado RN-02, score RN-05,
    cancelación con costo RN-07), con casos límite y casos de error;
  - todo endpoint: camino feliz, validación (400 con el `code` correcto) y RLS (un usuario de
    otro rol o dueño de otro recurso no ve ni modifica nada);
  - todo `Validate()` de un DTO;
  - todo bug corregido: un test que lo reproduce, antes del fix.
- Se corren con `go test -race -count=1 ./...` (`make test`, igual que el CI).
- Ningún test usa la base de producción `dbFletway`.
- Un RF/RN no se marca `OK` en `docs/TRAZABILIDAD.md` sin sus tests, incluido el de RLS.

### Prohibición de emojis

- **Prohibido usar emojis** en código Go, comentarios, strings (mensajes, códigos de error),
  logs, mensajes de commit, nombres de branch y archivos de documentación del repo.
- Para estados y marcas se usa texto: `OK`, `NO`, `EN CURSO`, `N/A` (leyenda de
  `docs/TRAZABILIDAD.md`), "Atención:", "(nuevo)".
- Los caracteres tipográficos que no son emojis (flechas, `≥`, `·`, `—`, líneas de diagramas)
  están permitidos.

---

## 6. Skills disponibles (`.claude/skills/`)

| Skill | Cuándo usarla |
|-------|---------------|
| `supabase-migration` | Generar/preparar una migración de Supabase con los patrones validados (RLS, `(select auth.uid())`, split de `FOR ALL`). **No aplica nada sin confirmación.** |
| `trace-requirement` | Antes de dar por cerrado un endpoint: verificar que un RF/RN de la ERS está realmente implementado y trazado. |
| `scaffold-endpoint` | Crear un endpoint Go nuevo siguiendo la convención del repo (JWT, RLS pass-through, patrón async, capas por feature). |
| `rls-policy-review` | Auditar (solo lectura) que las policies RLS de una tabla siguen los patrones y no quedaron `FOR ALL` permisivos tras agregar `_select`. |
| `update-project-state` | Actualizar `docs/ESTADO_PROYECTO.md` y `docs/DECISIONES_TECNICAS.md` de forma consistente al cerrar un bloque de trabajo. |

---

## 7. MCP de Supabase

Reglas completas en **`.claude/mcp-notes.md`**. Resumen:

- **Día a día = solo lectura**: `list_tables`, `list_migrations`, `get_advisors`,
  inspección de columnas y policies para auditar que la doc coincide con el catálogo real.
- **Escritura (`apply_migration`, `execute_sql` con DDL/DML, cambios de RLS, borrados)**:
  **solo con confirmación humana explícita en el momento**. Pausá y pedila.
- Si el esquema real difiere de `docs/DOCUMENTACION_BASE_DE_DATOS.md`: **no lo "arregles"**,
  reportá la diferencia y esperá instrucción.

# fletway-backend

Backend en **Go** de **Fletway**, marketplace two-sided de fletes y mudanzas.
Proyecto final — Ingeniería en Sistemas de Información, UTN FRD.

Este repo es el **dueño de la lógica de negocio central** y de la **documentación del
esquema de datos**. Expone una API REST (JWT en todos los endpoints) que consume la app
móvil Flutter (`fletway-mobile`). La web admin en Angular está **fuera de alcance** en
esta etapa.

## Stack

| Pieza | Elección |
|-------|----------|
| Lenguaje | Go 1.27+ |
| HTTP | `net/http` `ServeMux` (Go 1.22+ routing) — *propuesta a confirmar* |
| Base de datos | Postgres 17 en **Supabase** (`dbFletway`), 39 tablas, RLS activo en todas |
| Acceso a datos | `pgx` / `pgxpool` directo, con **RLS pass-through** (el JWT del usuario viaja a la sesión de Postgres) |
| Auth | JWT de Supabase Auth (GoTrue) |
| Async | pool de workers in-process (RNF-02) |
| Pagos | Mercado Pago / Stripe — *sin decidir* (RI-03) |

Detalle y estado de cada decisión: [`docs/DECISIONES_TECNICAS.md`](docs/DECISIONES_TECNICAS.md).

## Arranque rápido

```bash
cp .env.example .env      # completar DATABASE_URL, SUPABASE_*, etc.
go mod tidy
make run                  # o: go run ./cmd/api
curl localhost:8080/healthz
```

## Estructura

```
cmd/api/                  entrypoint del servicio HTTP
internal/
  platform/               infra transversal: config, database, auth, httpx, async, middleware
  server/                 construye el http.Server y registra las rutas de cada feature
  feature/<contexto>/     un paquete por contexto de negocio (routes/handler/service/repository/dto)
migrations/               migraciones .sql de Supabase (se aplican vía MCP, con confirmación)
qa/                       colecciones Postman + casos de prueba
docs/                     ERS, doc de base de datos, estado, decisiones, trazabilidad, contrato de API
.claude/                  contexto y skills de Claude Code + reglas del MCP de Supabase
```

## Documentación

| Documento | Contenido |
|-----------|-----------|
| [`docs/ERS_Fletway.pdf`](docs/ERS_Fletway.pdf) | Especificación de requisitos (24 RF, 8 RN, 4 RNF, 5 RI). |
| [`docs/DOCUMENTACION_BASE_DE_DATOS.md`](docs/DOCUMENTACION_BASE_DE_DATOS.md) | Esquema real desplegado, dominio por dominio, con RLS. |
| [`docs/ESTADO_PROYECTO.md`](docs/ESTADO_PROYECTO.md) | Foto viva del avance. |
| [`docs/DECISIONES_TECNICAS.md`](docs/DECISIONES_TECNICAS.md) | Decisiones de arquitectura y su estado. |
| [`docs/TRAZABILIDAD.md`](docs/TRAZABILIDAD.md) | Matriz RF/RN → implementación. |
| [`docs/ENDPOINTS.md`](docs/ENDPOINTS.md) | Contrato de API (fuente para los modelos del cliente Flutter). |
| [`CLAUDE.md`](CLAUDE.md) | Contexto para agentes de Claude Code. |

## Convenciones

- **Commits:** Conventional Commits + ref al requisito: `feat(solicitud): … [RF-06, RN-01]`.
- **Branches:** `<tipo>/<RF|RN>-<nn>-<slug>`, ej. `feat/RF-06-publicar-solicitud`.
- **Código:** `gofmt` + `goimports` + `golangci-lint`. Identificadores de la base en
  español, sin traducir.

Ver [`CLAUDE.md`](CLAUDE.md) §5 para el detalle.

## Reglas de negocio que no se violan

RN-01 a RN-08 (precio automático, cantidad de viajes, comisión, matchmaking por zona,
pull + top 3, PIN + GPS, cancelación con costo, catálogo de objetos). Tabla completa en
[`CLAUDE.md`](CLAUDE.md) §3.

## MCP de Supabase

Solo lectura para explorar/auditar el esquema en el día a día. Escritura (migraciones,
RLS, borrados) **solo con confirmación humana explícita**. Reglas en
[`.claude/mcp-notes.md`](.claude/mcp-notes.md).

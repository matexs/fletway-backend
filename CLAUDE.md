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
`solicitud` de traslado; el sistema calcula una cotización estimada y notifica a los
**Transportistas** compatibles por zona/vehículo/disponibilidad. Cada Transportista
habilitado se **postula** voluntariamente con una `oferta` (modelo *pull*, sin asignación
forzada). El Cliente ve un **top 3 por score de recomendación** y elige una: se confirma
un `viaje` con **snapshot histórico** de tarifa y datos. El viaje se ejecuta con
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
| **RN-01** | Precio **automático** del servicio en función de distancia, tiempo, cantidad de viajes, peso/volumen, demanda, escalera/altura, ayudantes y tarifa base. Aplica tanto a la cotización estimada (RF-06) como al precio de cada oferta (RF-17). Diseño que priorice **ganancia justa** para el Transportista. | Servicio de cotización. El Cliente **no** ingresa precio. Usa `config_tarifa` **vigente**. |
| **RN-02** | Cantidad de viajes y cubicaje calculados **automáticamente** por el sistema, para **cada tipo de vehículo posible** (usa `tipo_vehiculo`, aún sin vehículo real asignado). | Servicio de cotización al publicar `solicitud` (RF-06). |
| **RN-03** | La plataforma cobra **comisión sobre cada pago** procesado entre Cliente y Transportista. | Servicio de pagos. `config_comision` **vigente**; el % se congela en `viaje.porcentaje_comision_snapshot`. |
| **RN-04** | Matchmaking **por localidad**: emparejar `solicitud` con Transportistas cuya zona de trabajo coincida con **origen o destino**. | Query de matching sobre `transportista_zona` + `solicitud.origen_zona_id`/`destino_zona_id`. Sin ensanchamiento de radio ni ventanas de tiempo (descartado, ver doc DB §6). |
| **RN-05** | Postulación **pull** con **notificación push proactiva** al publicar; el Transportista se postula si quiere (sin obligación). Al Cliente se le muestra un **top 3 por score** (precio + calificación + cercanía + tasa de cumplimiento), con opción "ver más". | Job async de notificación (RNF-02) + endpoint de listado de ofertas que devuelve top 3 ordenado por score. |
| **RN-06** | Verificación por **PIN + geolocalización**: se genera un PIN por viaje; el Transportista carga PIN al **llegar a cargar** y un **segundo PIN al finalizar**; se registra su ubicación GPS. Con **PIN de fin válido** se habilita la `resena` del Cliente (RF-12). | Endpoints de ejecución de viaje (RF-22). `resena` solo si `viaje` en estado que la ERS/doc DB fija como habilitante. |
| **RN-07** | Cancelación con costo: Cliente cancela **sin cargo** si el Transportista **no salió**; **con cargo de resarcimiento** si ya salió (RF-08). El Transportista que cancela (RF-19) **no** paga penalización económica, pero baja su `tasa_cumplimiento`. | Endpoints de cancelación (RF-08, RF-19). |
| **RN-08** | Catálogo de **objetos comunes** (`objeto`) con peso/volumen estimados, seleccionables al publicar la solicitud (RF-06) y usados en la cotización (RN-01, RN-02). También se admite carga manual con dimensiones propias. | `solicitud_objeto` soporta `objeto_id` (catálogo) **o** `nombre_personalizado` + peso/volumen. |

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
  de modo que **las 34 políticas RLS ya desplegadas siguen siendo la capa de seguridad
  efectiva**. El backend agrega lógica de negocio y orquestación encima, no la reemplaza.
- **Auth:** se verifica el JWT emitido por Supabase Auth (GoTrue). `auth.uid()` = `usuario.id`.
- **Async (RNF-02):** pool de workers en proceso (`internal/platform/async`) para jobs
  disparados desde los handlers (ej. notificar Transportistas al publicar solicitud).
- **Capas por feature:** `internal/feature/<ctx>/` con `routes.go` + `handler.go` +
  `service.go` + `repository.go`. Infra compartida en `internal/platform/`.

> ⚠️ Varias de estas decisiones están marcadas **"propuesta a confirmar"** en
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
- Ejemplo: `feat(solicitud): endpoint de publicación con cotización estimada [RF-06, RN-01, RN-02]`

### Branches

- `main` — protegida, siempre desplegable.
- `feat/RF-06-publicar-solicitud` · `feat/RN-01-cotizacion` · `fix/RF-22-validacion-pin`
- Formato: `<tipo>/<RF|RN>-<nn>-<slug-corto>`. Trabajo sin requisito asociado:
  `chore/<slug>` o `docs/<slug>`.

### Código Go

- `gofmt` + `goimports` obligatorio; lint con `golangci-lint` (config en `.golangci.yml`).
- Errores: envolver con `fmt.Errorf("...: %w", err)`. Nunca `panic` en path de request
  (el middleware `recover` lo captura, pero no es la vía normal).
- Contexto: **todo** método que hace I/O recibe `context.Context` como primer parámetro.
- Nombres de tablas/columnas: **exactamente** como en `docs/DOCUMENTACION_BASE_DE_DATOS.md`
  (español, snake_case). No traducir identificadores de la base al inglés.

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

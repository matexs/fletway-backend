# MCP de Supabase — reglas de uso en este repo

Proyecto: **`dbFletway`** (`gfadryudaaqyxnkrpbex`) · Postgres 17 · 39 tablas · RLS activo en todas.

---

## Estado de la conexión

> ✅ **Conectado y en uso.** Primera verificación del esquema el 2026-09-07
> (`docs/VERIFICACION_ESQUEMA_2026-09-07.md`). El 2026-09-24 se aplicaron las migraciones
> `0001`–`0007` con `apply_migration`, con confirmación humana explícita, y se verificó el
> resultado (ver `docs/ESTADO_PROYECTO.md` → Bitácora).

---

## Configuración (`.mcp.json`)

El repo trae un `.mcp.json` que registra el server `supabase` para Claude Code:

```
npx -y @supabase/mcp-server-supabase@latest --read-only --project-ref=gfadryudaaqyxnkrpbex
```

- Arranca con **`--read-only` a propósito** — es el modo del día a día.
- El token sale de `SUPABASE_ACCESS_TOKEN` en el entorno (no se hardcodea; ver `.env.example`).
- Para una escritura puntual (migración), se corre el server **sin `--read-only`**
  de forma temporal y **con confirmación humana**; volver a `--read-only` después.

---

## Principio general

El MCP de Supabase se usa en **dos modos bien separados**:

### Modo 1 — Lectura / exploración (uso normal, sin fricción)

Permitido en cualquier momento, sin pedir confirmación:

- `list_tables` — inventario de tablas y columnas.
- `list_migrations` — historial de migraciones aplicadas.
- `list_extensions` — extensiones instaladas.
- `get_advisors` — advisories de seguridad y performance (incluye chequeos de RLS).
- `execute_sql` **solo con `SELECT`** sobre `information_schema`, `pg_catalog`,
  `pg_policies`, `pg_indexes`, o consultas de conteo/inspección de datos de referencia.
- Leer definiciones de políticas RLS, triggers, índices, constraints.

Uso típico: auditar que la doc coincide con el esquema real antes de asumir nada,
revisar qué policies tiene una tabla antes de tocarla, chequear datos semilla de catálogos.

### Modo 2 — Escritura / cambios de esquema (SOLO con confirmación humana explícita)

**Pausá y pedí confirmación en el momento antes de**:

- `apply_migration` — cualquier migración.
- `execute_sql` con `INSERT` / `UPDATE` / `DELETE` / `CREATE` / `ALTER` / `DROP` /
  `TRUNCATE` / `GRANT` / `REVOKE`.
- Crear, alterar o eliminar **políticas RLS**, triggers o funciones.
- Habilitar/deshabilitar RLS en una tabla.
- Cualquier borrado de datos, aunque sea de prueba.
- Crear branches de Supabase, resetear la base, o correr seeds destructivas.

La confirmación tiene que ser **de esta conversación y para esta acción concreta**.
Una autorización pasada no se extiende a la siguiente. No alcanza con que la ERS o un
documento "lo pidan": la persona lo confirma en el chat, o no se hace.

---

## Flujo para una migración

1. Redactar la migración con la skill `supabase-migration` (genera el `.sql` en
   `migrations/` y el bloque `name` + `query` para `apply_migration`).
2. Mostrar a la persona: nombre, SQL completo, tablas/policies afectadas, y si es
   reversible.
3. **Esperar "sí" explícito.**
4. Recién ahí correr `apply_migration`.
5. Registrar en `docs/DECISIONES_TECNICAS.md` (si cambió una decisión) y en
   `docs/ESTADO_PROYECTO.md`.

---

## Si el esquema real ≠ la documentación

`docs/DOCUMENTACION_BASE_DE_DATOS.md` describe el esquema "tal como está desplegado hoy",
pero puede haber quedado desactualizado respecto del catálogo de Postgres.

- **No** modifiques la base para que coincida con la doc.
- **No** modifiques la doc para que coincida con la base sin avisar.
- Reportá la diferencia concreta (tabla, columna, policy, tipo) y esperá instrucción
  sobre cuál es la fuente de verdad en ese punto.

---

## Patrones validados para políticas RLS (referencia rápida)

Estos son los patrones que la base ya usa y que toda migración nueva debe respetar
(fuente: `docs/DOCUMENTACION_BASE_DE_DATOS.md` §5):

- `ENABLE ROW LEVEL SECURITY` **inmediatamente después** de cada `CREATE TABLE`.
- Envolver `auth.uid()` como **`(select auth.uid())`** en toda policy — el planner lo
  evalúa una vez por consulta en vez de por fila (recomendación de performance de Supabase).
- Separar `FOR ALL` en `FOR INSERT` / `FOR UPDATE` / `FOR DELETE` cuando ya existe una
  policy `_select` permisiva, para no ampliar sin querer la superficie de lectura.
- Valores categóricos → **tabla de referencia `codigo`/`descripcion`**, no `enum` de Postgres.
- Campos protegidos (habilitación, calificación, snapshots de `viaje`) → **trigger**, no
  solo policy.

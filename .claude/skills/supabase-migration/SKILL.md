---
name: supabase-migration
description: >-
  Generar y preparar una migración de la base de datos de Supabase (proyecto
  dbFletway) para Fletway, siguiendo los patrones ya validados en el esquema
  desplegado (RLS después de CREATE TABLE, (select auth.uid()), split de FOR ALL,
  tablas de referencia en vez de enums). Produce el archivo .sql en migrations/ y
  el bloque name+query para apply_migration, pero NUNCA aplica nada sin
  confirmación humana explícita. Usar cuando haya que crear una tabla, índice,
  columna, política RLS, trigger o función nueva, o modificar una existente.
---

# Skill: supabase-migration

## Cuándo usar

Cualquier cambio de esquema o de seguridad en la base `dbFletway`: nueva tabla,
columna, índice, constraint, política RLS, trigger, función, o modificación de
alguno de estos. También para dumps de baseline o seeds de catálogos.

## Regla de oro

**Esta skill genera y prepara. NO aplica.** `apply_migration` /
`execute_sql` con DDL/DML se corren **solo** después de que un humano vea el SQL
completo y diga que sí, en esta misma conversación. Ver `.claude/mcp-notes.md`.

## Procedimiento

### 1. Verificar el estado real (solo lectura)

Antes de escribir SQL, inspeccionar el catálogo real vía MCP:

- `list_tables` — ¿la tabla/columna ya existe? ¿con qué tipo?
- `list_migrations` — último número correlativo aplicado.
- `SELECT` sobre `pg_policies` para la tabla afectada — ¿qué policies tiene hoy?
- `get_advisors` (security + performance) — ¿hay advisories abiertos que esta
  migración deba respetar o resolver?

Si el catálogo real no coincide con `docs/DOCUMENTACION_BASE_DE_DATOS.md`:
**parar**, reportar la diferencia, no continuar hasta saber cuál es la fuente de
verdad.

### 2. Redactar el `.sql`

Archivo: `migrations/NNNN_<verbo>_<objeto>.sql` (NNNN = correlativo + 1).

Encabezado obligatorio en el archivo:

```sql
-- NNNN_<verbo>_<objeto>.sql
-- Requisito: RF-XX / RN-XX (o "infra" si no aplica)
-- Reversible: sí | no  (si no, explicar por qué)
-- Afecta: <tablas / policies / triggers>
```

### 3. Aplicar los patrones validados

**P1 — RLS inmediato.** Después de `CREATE TABLE x (...)`:

```sql
ALTER TABLE x ENABLE ROW LEVEL SECURITY;
```

Nunca dejar una tabla nueva sin RLS (todas las tablas actuales lo tienen).

**P2 — `(select auth.uid())`, nunca `auth.uid()` pelado.**

```sql
-- MAL
CREATE POLICY p_select ON x FOR SELECT USING (usuario_id = auth.uid());
-- BIEN (el planner lo evalúa una vez por consulta, no por fila)
CREATE POLICY p_select ON x FOR SELECT USING (usuario_id = (select auth.uid()));
```

Lo mismo para `auth.jwt()`, `auth.role()` → envolver en `(select ...)`.

**P3 — Separar `FOR ALL` cuando ya hay `_select` permisiva.** Si la tabla ya
tiene (o va a tener) una policy `_select` de lectura amplia (ej. perfiles
públicos, catálogos), **no** agregar una `FOR ALL` para la escritura: se
solapan y una `FOR ALL` permisiva puede ampliar sin querer la lectura. En su
lugar, policies explícitas por operación:

```sql
CREATE POLICY x_insert ON x FOR INSERT WITH CHECK (usuario_id = (select auth.uid()));
CREATE POLICY x_update ON x FOR UPDATE USING (usuario_id = (select auth.uid()))
                                       WITH CHECK (usuario_id = (select auth.uid()));
CREATE POLICY x_delete ON x FOR DELETE USING (usuario_id = (select auth.uid()));
```

**P4 — Categóricos → tabla de referencia, no `enum`.** Un valor que podría
crecer o necesitar metadata va en una tabla `codigo text PRIMARY KEY,
descripcion text NOT NULL`, referenciada por FK — nunca `CREATE TYPE ... AS ENUM`.
Coherente con `estado_*` / `tipo_*` existentes.

**P5 — Campos protegidos → trigger.** Si hay columnas que el propio usuario no
debe poder modificar (habilitación, calificación, snapshots de `viaje`), la
protección es un trigger `BEFORE UPDATE` que revierte/rechaza el cambio, además
de la policy. Ver `trg_proteger_campos_transportista` y `trg_proteger_campos_viaje`.

**P6 — Snapshots en tablas de evento (RNF-03).** Si la migración toca `viaje` (u
otra tabla transaccional de evento), respetar el patrón `*_snapshot`: los datos
que podrían cambiar en tablas maestras se congelan en columnas de la fila de
evento. No agregar FKs a maestras "para leer el valor actual" donde el diseño
pide snapshot.

**P7 — Índices para las queries de RLS.** Toda columna usada en `USING` /
`WITH CHECK` de una policy (típicamente `usuario_id`, `*_id` de ownership)
necesita índice, o las consultas con RLS escanean de más.

### 4. Preparar (no ejecutar) `apply_migration`

Presentar al humano:

```
Migración:  NNNN_<verbo>_<objeto>
Requisito:  RF-XX / RN-XX
Reversible: sí/no
Afecta:     <lista>

--- SQL ---
<contenido completo del .sql>

¿Aplico esta migración con apply_migration? (necesito un sí explícito)
```

### 5. Post-aplicación (solo si se confirmó y aplicó)

- `list_migrations` para confirmar que quedó registrada.
- `get_advisors` de nuevo — que no haya introducido advisories nuevos.
- Actualizar `docs/ESTADO_PROYECTO.md` (bitácora) y, si corresponde,
  `docs/DOCUMENTACION_BASE_DE_DATOS.md` y `docs/DECISIONES_TECNICAS.md`.

## Checklist

- [ ] Inspeccioné el catálogo real antes de escribir SQL.
- [ ] El `.sql` tiene encabezado (requisito, reversibilidad, afecta).
- [ ] Toda tabla nueva tiene `ENABLE ROW LEVEL SECURITY`.
- [ ] Cero `auth.uid()` pelado; todo `(select auth.uid())`.
- [ ] No hay `FOR ALL` solapada con una `_select` permisiva.
- [ ] Categóricos como tabla de referencia, no `enum`.
- [ ] Columnas de policy tienen índice.
- [ ] Mostré el SQL completo y pedí confirmación explícita.
- [ ] NO ejecuté nada sin ese "sí".

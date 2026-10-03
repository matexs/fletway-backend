# migrations/

Migraciones de la base de datos de Supabase (`dbFletway`).

## Estado

El esquema **ya está desplegado** (39 tablas, RLS activo en todas). El esquema inicial
(13 migraciones `01_identidad` … `09d_indices_fk_faltantes`, 2026-08-22/23) **no** está en
este directorio: vive sólo en el historial de Supabase. Acá se versionan los cambios
**posteriores**, generados con la skill `supabase-migration`.

| Migración | Estado | Tema |
|---|---|---|
| `0001`–`0007` | Aplicadas 2026-09-24 | Rediseño de cotización (RN-01) y cálculo de viajes (RN-02). Ver `docs/ALGORITMO_COTIZACION.md` §8 y el historial en `docs/DOCUMENTACION_BASE_DE_DATOS.md` §8. |

## Esquema base y entorno local (D-16)

- **`baseline/`** guarda un dump (sólo esquema, schema `public`) de `dbFletway` tomado con
  `supabase db dump`. Incluye las 13 migraciones iniciales **y** las `0001`–`0007`: es la foto
  completa a partir de la cual se reproduce la base en local. No se edita a mano.
- **Local:** `make db-local` (o `scripts/db-local.sh`) levanta Supabase local, resetea la base,
  aplica el esquema base y después cada `NNNN_*.sql` posterior a la `0007`, en orden.
- **Producción:** cada migración nueva se aplica en `dbFletway` con `apply_migration`, sólo con
  confirmación humana explícita (D-08), **después** de probarla en local.

## Convención de nombres

```
NNNN_<verbo>_<objeto>.sql
```

- `NNNN` — número correlativo de 4 dígitos (`0001`, `0002`, …).
- Ejemplos: `0001_add_index_solicitud_estado.sql`,
  `0002_split_for_all_policy_oferta.sql`.
- Cada `.sql` es **una** unidad lógica de cambio, idealmente reversible.

## Cómo se aplican

1. Se genera el `.sql` acá con la skill `supabase-migration`.
2. La skill también arma el bloque `name` + `query` para `apply_migration` del MCP.
3. **Se muestra a un humano**: nombre, SQL completo, tablas/policies afectadas,
   reversibilidad.
4. **Solo con "sí" explícito** se corre `apply_migration`.
5. Se registra en `docs/ESTADO_PROYECTO.md` (y en `docs/DECISIONES_TECNICAS.md` si
   cambió una decisión).

Nunca se aplican migraciones desde CI en esta etapa (ver `.claude/mcp-notes.md`).

## Patrones obligatorios

Ver `.claude/skills/supabase-migration/SKILL.md`. En resumen:

- `ENABLE ROW LEVEL SECURITY` después de cada `CREATE TABLE`.
- `(select auth.uid())` en vez de `auth.uid()` directo.
- Separar `FOR ALL` en `FOR INSERT` / `FOR UPDATE` / `FOR DELETE` si ya hay una
  policy `_select` permisiva.
- Categóricos → tabla de referencia `codigo`/`descripcion`, no `enum`.

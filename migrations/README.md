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
| `0008` | Aplicada 2026-10-02 | Seed de las 14 zonas piloto (módulo 0, RN-04). |
| `0009` | Aplicada 2026-10-02 | Catálogo de 28 objetos con medidas y `NOT NULL` en `largo_m`/`ancho_m`/`alto_m` (módulo 0, RN-08). |
| `0010` | Aplicada 2026-10-02 | Corta la recursión infinita entre `solicitud_select` y `oferta_select` con `fn_tiene_oferta_en_solicitud` (módulo 2). |
| `0011` | Aplicada 2026-10-02 | Trigger de alta `trg_alta_usuario` en `auth.users` (D-18), protección de `usuario.rol`/`email`/`activo`, alta de Transportista siempre `pendiente` e inserción de la fila de rol sólo si coincide con `usuario.rol` (módulo 2). |
| `0012` | Aplicada 2026-10-03 | Bucket privado `documentos-transportista` y sus policies de Storage (D-19); alta de documentos siempre pendiente y en el prefijo propio; estado de habilitación derivado de los documentos (D-33) (módulo 3). |
| `0013` | Aplicada 2026-10-03 | `tipo_vehiculo` con largo, ancho y alto estándar y 6 tipos nuevos (A-1, D-32); `volumen_estandar_m3` deprecada (módulo 4). |
| `0014` | Aplicada 2026-10-03 | Fecha y franja del servicio en `solicitud`; sin edición, objetos fijos y copia del catálogo por triggers (D-20, módulo 6). |
| `0015` | Sólo local (pendiente de confirmación para `dbFletway`) | Matchmaking: `fn_es_compatible` (D-21), listado de compatibles y aviso in-app idempotente (módulo 7). |

## Esquema base y entorno local (D-16)

- **`baseline/`** guarda el estado de `dbFletway` al 2026-10-02, tomado con `supabase db dump`:
  `00_extensiones.sql` (extensiones que el dump no incluye), `01_esquema_*.sql` (estructura del
  schema `public`) y `02_datos_catalogos_*.sql` (sólo catálogos y configuración; producción no
  tenía datos de personas). Incluye las 13 migraciones iniciales **y** las `0001`–`0007`: es la foto
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

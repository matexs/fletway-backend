---
name: rls-policy-review
description: >-
  Auditar (solo lectura) las políticas RLS de una o varias tablas de la base
  dbFletway de Fletway contra los patrones validados y contra la matriz de
  permisos documentada (doc de base de datos §5). Detecta auth.uid() sin
  envolver, FOR ALL solapadas con _select permisivas, tablas con RLS
  deshabilitado, columnas de policy sin índice, y desalineación con la matriz
  Cliente/Transportista/Administrador. No modifica nada. Usar antes de exponer
  una feature que escribe directo desde la app (mensaje, viaje_ubicacion), tras
  aplicar una migración, o cuando get_advisors marca algo de RLS.
---

# Skill: rls-policy-review

## Cuándo usar

- Antes de habilitar en la app una feature que **escribe directo a Supabase** sin
  pasar por el backend (chat `mensaje` RF-10/RF-20, GPS `viaje_ubicacion`
  RI-04/RF-15 — modelo híbrido D-10). Ahí las policies son la única defensa.
- Después de aplicar una migración que tocó policies.
- Cuando `get_advisors` (security) reporta algo de RLS.
- En revisión periódica de las 34 tablas.

## Es SOLO lectura

Esta skill no ejecuta `ALTER`, no crea ni dropea policies. Si encuentra algo para
corregir, produce hallazgos y (si se pide) una migración candidata con la skill
`supabase-migration`, que a su vez pide confirmación humana.

## Insumos (todo vía MCP, solo lectura)

- `SELECT * FROM pg_policies WHERE schemaname='public' AND tablename = ANY($1)`
- `SELECT relname, relrowsecurity, relforcerowsecurity FROM pg_class WHERE ...`
  (¿RLS habilitado? ¿forzado?)
- `pg_indexes` para la tabla.
- Definición de triggers (`pg_trigger`) si la tabla tiene campos protegidos.
- `docs/DOCUMENTACION_BASE_DE_DATOS.md` §5 — matriz de permisos esperada.

## Chequeos

### C1 — RLS habilitado

`relrowsecurity = true` para **toda** tabla de `public`. Una tabla sin RLS es un
hallazgo crítico.

### C2 — `auth.uid()` / `auth.jwt()` / `auth.role()` envueltos

Toda aparición en `qual` (USING) o `with_check` debe ser `(select auth.uid())`,
nunca `auth.uid()` directo. Grep sobre el texto de cada policy.

### C3 — `FOR ALL` solapada con `_select` permisiva

Si la tabla tiene una policy `cmd = 'ALL'` **y** otra de `SELECT` con `qual` más
permisiva (o `true`), la `FOR ALL` puede estar ampliando la lectura sin querer.
Hallazgo: proponer separar la `FOR ALL` en `INSERT` / `UPDATE` / `DELETE`
explícitas.

### C4 — Alineación con la matriz §5

Para cada tabla, cruzar las policies reales con lo que la doc dice que cada rol
puede hacer:

- **Lectura pública** (catálogos, perfiles de transportista, reseñas): debe haber
  `SELECT` con `qual` amplio.
- **Propiedad** (transaccional): `SELECT`/`UPDATE`/`DELETE` con `usuario_id =
  (select auth.uid())` o equivalente por join de ownership.
- **Administrador**: acceso total (típicamente vía chequeo de existencia en
  `administrador` o claim de rol).
- **Exclusivo Admin** (`config_tarifa`, `config_comision`, `veto`,
  `incidente_resolucion`, habilitación): que Cliente/Transportista **no** tengan
  policy de escritura.
- **`incidente`**: el reportante y el Admin ven; la **contraparte no**. Verificar
  que no haya una policy que filtre incidentes ajenos.
- **`viaje_ubicacion`**: solo el propio Transportista puede `INSERT`; Cliente y
  Transportista del viaje pueden `SELECT` los de sus viajes.
- **`mensaje`**: `INSERT`/`SELECT` solo para Cliente y Transportista del `viaje`
  correspondiente; no puede existir `mensaje` sin `viaje` (chat habilitado solo
  tras aceptar — RI-05).

### C5 — Campos protegidos por trigger

`transportista` (`estado_habilitacion_codigo`, `calificacion_promedio`,
`tasa_cumplimiento`) y `viaje` (columnas `*_snapshot` + FKs estructurales) deben
tener trigger `BEFORE UPDATE` que impida al usuario común cambiarlos. Verificar
que el trigger existe y está `enabled`.

### C6 — Índices para las policies

Cada columna referenciada en `qual`/`with_check` (ownership, estado, FKs de
join) debería tener índice. Sin índice, las lecturas con RLS escanean de más.

### C7 — `WITH CHECK` en INSERT/UPDATE

Policies de `INSERT` y `UPDATE` deben tener `with_check` (no solo `using`), o se
puede insertar/actualizar filas que luego no se pueden leer, o peor, escribir a
nombre de otro.

## Salida

Tabla de hallazgos:

| # | Tabla | Chequeo | Severidad | Detalle | Corrección propuesta |
|---|-------|---------|-----------|---------|----------------------|

Severidad: `crítico` (agujero de seguridad) / `alto` (desalineación con §5) /
`medio` (performance / falta índice) / `bajo` (estilo).

Si se pide corregir: derivar a `supabase-migration` para preparar el `.sql`
(que pedirá confirmación antes de aplicarse).

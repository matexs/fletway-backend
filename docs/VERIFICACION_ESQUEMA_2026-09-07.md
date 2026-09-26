# Informe de verificación de esquema — `dbFletway` vs `DOCUMENTACION_BASE_DE_DATOS.md`

**Fecha:** 2026-09-07 · **Proyecto Supabase:** `dbFletway` (`gfadryudaaqyxnkrpbex`) · **Postgres:** 17.6
**Método:** MCP de Supabase en **solo lectura** — `list_tables`, `list_migrations`, `list_extensions`,
`get_advisors` (security + performance) y consultas de solo lectura sobre `pg_class`, `pg_policies`,
`pg_trigger`, `pg_proc`, `pg_type`, `pg_index`.
**Resultado:** el esquema desplegado coincide con la doc en **todo lo estructural**. Hay **una (1)
diferencia real** (error de recuento de tablas) y algunas observaciones de advisors. **No se modificó nada.**

---

## 1. Checklist pedido — resultado

| # | Ítem a verificar | Esperado (doc) | Real (catálogo) | Estado |
|---|------------------|----------------|-----------------|--------|
| 1 | Cantidad de tablas | 34 | **35** | **DIFIERE** (ver §2) |
| 2 | RLS activo en todas las tablas | sí | 35/35 con `relrowsecurity = true`, 0 con `FORCE` | OK |
| 3 | Políticas con `(select auth.uid())` | sí, nunca `auth.uid()` desnudo | 109/109 políticas usan `( SELECT auth.uid() AS uid)` | OK |
| 4 | Trigger `trg_proteger_campos_transportista` | en `transportista` | `BEFORE UPDATE ... FOR EACH ROW EXECUTE fn_proteger_campos_transportista()` | OK |
| 5 | Trigger `trg_proteger_campos_viaje` | en `viaje` | `BEFORE UPDATE ... FOR EACH ROW EXECUTE fn_proteger_campos_viaje()` | OK |
| 6 | Sin enums de Postgres (todo `codigo`/`descripcion`) | sí | 0 enums en `public`; solo catálogos + CHECK inline | OK |
| 7 | Columnas `*_snapshot` en `viaje` | ubicación + financiero + identidad congelada | 14 columnas `*_snapshot` presentes (ver §3.4) | OK |

---

## 2. Diferencia encontrada — **NO corregida**

### D-1 · La doc dice «34 tablas», el esquema real tiene **35**

- `docs/DOCUMENTACION_BASE_DE_DATOS.md` afirma **34** en 3 lugares:
  - línea 3 (encabezado): «Postgres 17 · 34 tablas»
  - línea 28 (§2): «**RLS activo en las 34 tablas**»
  - línea 55 (§3, nota del diagrama): «omite las 13 tablas de referencia/catálogo»
- El catálogo real tiene **35 tablas base** en `public`. Coinciden las 3 fuentes consultadas:
  `mcp_list_tables` (35), `pg_tables` (35), `pg_class WHERE relkind='r'` (35).
- **No es drift de esquema.** Las 35 tablas están **todas descritas individualmente** en §4 de la
  propia doc. El recuento por dominio de §4 da **35**:

  | Dominio | Tablas | n |
  |---|---|---|
  | 1 · Identidad | usuario, cliente, administrador, transportista, estado_habilitacion_transportista, tipo_documento, documento_transportista | 7 |
  | 2 · Geografía | zona, transportista_zona | 2 |
  | 3 · Vehículos y objetos | tipo_vehiculo, vehiculo, objeto | 3 |
  | 4 · Solicitudes y matchmaking | estado_solicitud, solicitud, solicitud_objeto, estado_oferta, oferta | 5 |
  | 5 · Viajes | estado_viaje, viaje, viaje_ubicacion | 3 |
  | 6 · Pagos | config_tarifa, config_comision, estado_pago, pago, pago_movimiento | 5 |
  | 7 · Reseñas e incidentes | resena, tipo_incidente, estado_incidente, incidente, veto, tipo_resolucion_incidente, incidente_resolucion | 7 |
  | 8 · Notificaciones y chat | tipo_notificacion, notificacion, mensaje | 3 |
  | **Total** | | **35** |

- **Diagnóstico:** es un **error de recuento en el resumen** de la doc, no una tabla inesperada ni
  faltante. Ninguna tabla del catálogo real está sin documentar; ninguna tabla documentada falta en
  la base.
- **Arrastre:** `CLAUDE.md` (§3, «`viaje` … snapshot») no cita número, pero `CLAUDE.md` §4 dice «las
  34 políticas RLS ya desplegadas» (ver D-2) y `docs/ESTADO_PROYECTO.md` línea 22 dice «34 tablas».
- **Impacto:** documental / cosmético. Cero impacto en seguridad o lógica.
- **Acción requerida (humano):** confirmar si se corrige el número a **35** en
  `DOCUMENTACION_BASE_DE_DATOS.md` (líneas 3 y 28) y su arrastre en `CLAUDE.md` /
  `ESTADO_PROYECTO.md`; o bien confirmar que se esperaban 34 y decidir qué hacer (a priori **no
  sobra ninguna tabla**: las 35 están documentadas y justificadas por un RF/RN).

### D-2 · `CLAUDE.md` dice «34 políticas RLS», el real tiene **109**

- `CLAUDE.md` §4: «las **34** políticas RLS ya desplegadas siguen siendo la capa de seguridad efectiva».
- Real: **109 políticas** en `public` (2 a 4 por tabla). Probable confusión «34 políticas» ↔ «34 tablas».
- No afecta a `DOCUMENTACION_BASE_DE_DATOS.md` (no cita un número de políticas). Se registra para
  alinear `CLAUDE.md` cuando se decida sobre D-1.

---

## 3. Detalle de lo verificado OK

### 3.1 Tablas y RLS
- 35 tablas base en `public`, **todas** con RLS habilitado. Ninguna con `FORCE ROW LEVEL SECURITY`
  (coherente con el modelo de RLS pass-through: el backend entra como `authenticated`, no como owner).
- Sin vistas en `public`.
- 13 migraciones aplicadas, coherentes con los 8 dominios + RLS + fixes:
  `01_identidad`, `01b_fix_rls_tablas_referencia`, `02_geografia`, `03_vehiculos_objetos`,
  `04_solicitudes_matchmaking`, `05_viajes`, `06_pagos`, `07_resenas_incidentes`,
  `08_notificaciones_chat`, `09_rls_policies`, `09b_fix_advisories_seguridad`,
  `09c_fix_performance_rls`, `09d_indices_fk_faltantes`.

### 3.2 Políticas RLS
- **109/109** políticas usan `( SELECT auth.uid() AS uid)` — **ninguna** usa `auth.uid()` desnudo.
- Los catálogos (`estado_*`, `tipo_*`, `zona`, `objeto`, `tipo_vehiculo`) tienen el patrón esperado:
  `_select USING (true)` + `_insert/_update/_delete` con `fn_es_administrador()`. No quedaron
  `FOR ALL` permisivos solapados (el split está aplicado).
- `fn_es_administrador()` y `fn_es_transportista_habilitado()`: `sql`, `STABLE`, `SECURITY DEFINER`,
  `search_path = public`, ambas evalúan `(select auth.uid())` internamente.
- Reglas de negocio reflejadas en policy y coincidentes con la doc:
  - `oferta_insert`: `transportista_id = uid AND fn_es_transportista_habilitado()` (solo habilitado).
  - `resena_insert`: exige `viaje.estado_codigo = 'finalizado'` (RN-06 vigente / doc §Dominio 7).
  - `viaje_ubicacion_insert`: solo el `transportista_id` del viaje puede insertar pings.
  - `solicitud_select`: cliente propio ∨ admin ∨ (`publicada` ∧ zona del transportista ∈
    {origen, destino}) ∨ ya postulado — matchmaking por localidad (RN-04) sin ensanchamiento.
  - `mensaje_*`: atado a `viaje_id` y a que el emisor sea cliente/transportista del viaje.

### 3.3 Triggers de protección
| Trigger | Tabla | Momento | Función | Campos que revierte si `NOT fn_es_administrador()` |
|---|---|---|---|---|
| `trg_proteger_campos_transportista` | `transportista` | `BEFORE UPDATE` FOR EACH ROW | `fn_proteger_campos_transportista()` (plpgsql, SEC DEF, `search_path=public`) | `estado_habilitacion_codigo`, `calificacion_promedio`, `tasa_cumplimiento` |
| `trg_proteger_campos_viaje` | `viaje` | `BEFORE UPDATE` FOR EACH ROW | `fn_proteger_campos_viaje()` (plpgsql, SEC DEF, `search_path=public`) | FKs `oferta_id`, `solicitud_id`, `cliente_id`, `transportista_id` + `monto_total_snapshot`, `tarifa_base_snapshot`, `valor_por_km_snapshot`, `valor_por_m3_snapshot`, `valor_por_hora_snapshot`, `recargo_escalera_snapshot`, `recargo_ayudante_snapshot`, `porcentaje_comision_snapshot` |

Coincide con la doc (§Dominio 1 y §Dominio 5). Nota menor: la función de `viaje` **no** revierte
`origen_*_snapshot` / `destino_*_snapshot` / `distancia_km_snapshot` / `transportista_nombre_snapshot`
/ `vehiculo_patente_snapshot` / `vehiculo_marca_modelo_snapshot` (sí revierte los financieros y las
FK). La doc dice «impide que … alteren estas columnas snapshot **o** las FK estructurales» — la
protección real cubre los snapshots **financieros** + FKs, no los snapshots de ubicación/identidad.
Es una imprecisión de redacción de la doc, no un riesgo estructural; se deja anotado, **sin corregir**.

### 3.4 Enums y columnas snapshot
- **Enums Postgres en `public`: 0.** Los únicos enums del cluster son de `auth` / `realtime` /
  `storage` (Supabase interno). Todo lo categórico del negocio es tabla `codigo`/`descripcion` o
  `CHECK` inline (`usuario.rol`, `documento_transportista.estado`, `pago_movimiento.tipo`,
  `veto.tipo`, `veto.estado`, `resena.calificacion`, `solicitud_objeto.cantidad`).
- **`viaje` — 14 columnas `*_snapshot`** (todas presentes): `origen_direccion_snapshot`,
  `destino_direccion_snapshot`, `origen_lat_snapshot`, `origen_lng_snapshot`, `destino_lat_snapshot`,
  `destino_lng_snapshot`, `distancia_km_snapshot`, `transportista_nombre_snapshot`,
  `vehiculo_patente_snapshot`, `vehiculo_marca_modelo_snapshot`, `tarifa_base_snapshot`,
  `valor_por_km_snapshot`, `valor_por_m3_snapshot`, `valor_por_hora_snapshot`,
  `recargo_escalera_snapshot`, `recargo_ayudante_snapshot`, `monto_total_snapshot`,
  `porcentaje_comision_snapshot`. (18 en total; la doc las agrupa en ubicación / identidad / financiero.)

### 3.5 Versionado de config
- `idx_config_tarifa_vigente` y `idx_config_comision_vigente`:
  `UNIQUE (vigente_hasta) WHERE vigente_hasta IS NULL` → garantiza **una sola fila vigente** por
  tabla. Coincide con doc §Dominio 6.

### 3.6 Otros
- `usuario.email`: `citext` `UNIQUE` (extensión `citext` instalada en schema `extensions`). (OK)
- PKs, UNIQUEs 1:1 (`viaje.oferta_id`, `pago.viaje_id`, `resena.viaje_id`,
  `incidente_resolucion.incidente_id`) presentes. (OK)
- Extensiones instaladas: `citext`, `pgcrypto`, `uuid-ossp`, `pg_stat_statements`, `supabase_vault`,
  `pg_graphql`*, `plpgsql`. Sin extensiones de negocio inesperadas (PostGIS / pg_cron / pg_net solo
  *disponibles*, no instaladas). La doc no enumera extensiones → sin diferencia.

---

## 4. Advisors (registro, no son diferencias con la doc)

### 4.1 Seguridad — 4 × WARN
`fn_es_administrador()` y `fn_es_transportista_habilitado()` son `SECURITY DEFINER` y **ejecutables
por `anon` y `authenticated`** vía `/rest/v1/rpc/<fn>` (2 funciones × 2 roles = 4 lints):
- `0028_anon_security_definer_function_executable`
- `0029_authenticated_security_definer_function_executable`

Son **helpers de RLS**; el `SECURITY DEFINER` es intencional (necesitan leer `administrador` /
`transportista` saltando RLS). El lint sugiere `REVOKE EXECUTE ... FROM anon, authenticated` o
sacarlas del schema expuesto. **No accionar sin confirmación humana** — anotarlo para el bloque de
hardening de seguridad.
Ref: <https://supabase.com/docs/guides/database/database-linter?lint=0028_anon_security_definer_function_executable>

### 4.2 Performance — 43 × INFO
Todos `0005_unused_index` (43 índices sobre FK / estado nunca usados). **Esperable**: la base está
vacía (0 filas) y sin tráfico. No accionar ahora; reevaluar tras carga real de datos. La doc no
lista índices → sin diferencia que registrar.

---

## 5. Conclusión

El esquema desplegado en `dbFletway` **coincide con `docs/DOCUMENTACION_BASE_DE_DATOS.md` en todo lo
estructural**: RLS activo en todas las tablas, 100 % de políticas con `(select auth.uid())`, ambos
triggers de protección con sus campos, ausencia total de enums de Postgres, columnas `*_snapshot` en
`viaje`, versionado de config con índice único parcial, y las reglas de negocio clave (RN-04, RN-06,
habilitación de transportista, chat atado al viaje) reflejadas en las policies.

**Única diferencia real:** el recuento de tablas — la doc dice **34**, el catálogo tiene **35** (D-1).
Es un error de recuento en el resumen, no drift de esquema (las 35 están documentadas en §4).

**Pendiente de decisión humana** antes de tocar la doc:
1. Corregir «34 tablas» → «35» en `DOCUMENTACION_BASE_DE_DATOS.md` (líneas 3 y 28) y el arrastre en
   `CLAUDE.md` / `ESTADO_PROYECTO.md`. (D-1)
2. Corregir «34 políticas RLS» → «109» (o reformular) en `CLAUDE.md` §4. (D-2)
3. Precisar en `DOCUMENTACION_BASE_DE_DATOS.md` §Dominio 5 que `trg_proteger_campos_viaje` protege
   los snapshots **financieros** + FKs, no los de ubicación/identidad. (§3.3)
4. Evaluar los 4 WARN de seguridad de advisors sobre las funciones helper `SECURITY DEFINER`. (§4.1)

Ninguna de estas acciones se ejecutó: este informe solo reporta.

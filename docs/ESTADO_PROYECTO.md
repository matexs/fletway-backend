# Estado del proyecto — fletway-backend

> Foto viva del avance. Actualizar al cerrar cada bloque de trabajo (skill
> `update-project-state`). Fechas en formato absoluto.

**Última actualización:** 2026-09-07 · **Etapa:** scaffolding inicial + esquema verificado vía MCP

---

## Resumen de una línea

Esquema de datos desplegado y documentado; ERS cerrada; **código Go todavía no empezado** —
este repo tiene la estructura, los archivos de contexto y las skills, pero cero endpoints
implementados.

---

## Qué está hecho

| Área | Estado | Notas |
|------|--------|-------|
| Base de datos (Supabase `dbFletway`) | ✅ Desplegada · ✅ Verificada vía MCP (2026-09-07) | **35** tablas (la doc dice 34 — ver D-1 en `docs/VERIFICACION_ESQUEMA_2026-09-07.md`), RLS activo en todas, 109 políticas 100 % con `(select auth.uid())`, triggers `trg_proteger_campos_*` OK, sin enums de Postgres. Documentada en `docs/DOCUMENTACION_BASE_DE_DATOS.md`. |
| ERS | ✅ Cerrada | 24 RF, 8 RN, 4 RNF, 5 RI. `docs/ERS_Fletway.pdf`. |
| Estructura del repo Go | ✅ Scaffolding | `cmd/`, `internal/platform/`, `internal/feature/`, `migrations/`, `qa/`, `docs/`. |
| Archivos de contexto | ✅ | `CLAUDE.md`, este archivo, `DECISIONES_TECNICAS.md`, `TRAZABILIDAD.md`, `ENDPOINTS.md`. |
| Skills de Claude Code | ✅ | `supabase-migration`, `trace-requirement`, `scaffold-endpoint`, `rls-policy-review`, `update-project-state`. |
| Config MCP Supabase | ✅ | `.claude/mcp-notes.md` (solo lectura por defecto; escritura con confirmación). |
| Endpoints de negocio | ❌ No empezado | Solo `GET /healthz` y `GET /readyz` de ejemplo. |
| Verificación de módulos Go | ⚠️ Pendiente | `go mod tidy` no corrido (falta resolver dependencias en red). |

---

## Pendientes / próximos pasos (ordenados)

1. ~~**Verificar esquema real vía MCP de Supabase.**~~ ✅ **Hecho 2026-09-07.** Corridos
   `list_tables`, `list_migrations`, `list_extensions`, `get_advisors` (security + performance)
   y consultas de solo lectura sobre `pg_class` / `pg_policies` / `pg_trigger` / `pg_proc` /
   `pg_type` / `pg_index`. Informe completo: **`docs/VERIFICACION_ESQUEMA_2026-09-07.md`**.
   Resultado: esquema **coincide con la doc en todo lo estructural**. Diferencias registradas
   (no corregidas): **D-1** la doc dice «34 tablas» y hay **35** (error de recuento, ninguna
   tabla sobra ni falta); **D-2** `CLAUDE.md` dice «34 políticas RLS» y hay **109**. Pendiente
   decisión humana sobre corregir la doc (ver §5 del informe). Advisors: 4 WARN de seguridad
   (funciones helper `SECURITY DEFINER` ejecutables por `anon`/`authenticated`) + 43 INFO
   `unused_index` (base vacía, esperable).
2. **Confirmar las decisiones marcadas "propuesta a confirmar"** en
   `docs/DECISIONES_TECNICAS.md` (acceso a datos, router, verificación de JWT, módulo Go,
   estrategia de jobs async, pasarela de pagos).
3. `go mod tidy` + fijar versiones de `pgx`, lib de JWT, etc.
4. Implementar `internal/platform/auth` (verificación real del JWT de Supabase) y
   `internal/platform/database` (pool + helper de RLS pass-through).
5. Primer endpoint de negocio real con la skill `scaffold-endpoint` — candidato:
   **RF-05 (registro de Cliente)** o **RF-16 (registro de Transportista)**.
6. Colección Postman base en `qa/postman/` y primeros casos de prueba en `qa/casos-prueba/`.

---

## Bloqueos conocidos

- ~~**MCP de Supabase no conectado**~~ → resuelto 2026-09-07; esquema verificado
  (`docs/VERIFICACION_ESQUEMA_2026-09-07.md`).
- **Doc de base de datos desalineada en 2 recuentos** (no bloquea, requiere decisión humana):
  `DOCUMENTACION_BASE_DE_DATOS.md` dice «34 tablas» (real: **35**) y `CLAUDE.md` §4 dice
  «34 políticas RLS» (real: **109**). Detalle y propuesta de corrección en §5 del informe.
  **No se modificó ninguna doc de esquema** — pendiente de confirmación.
- **4 WARN de seguridad (advisors)** sobre `fn_es_administrador()` /
  `fn_es_transportista_habilitado()` (`SECURITY DEFINER` ejecutables por `anon`/`authenticated`).
  Intencional como helpers de RLS; revisar en el bloque de hardening antes de exponer la app.
- **Toolchain:** `go` 1.27.1 disponible; `flutter` NO (no afecta a este repo).
- Faltan credenciales reales de la pasarela de pagos (Mercado Pago / Stripe) — RI-03.
- Valores de `config_tarifa` / `config_comision` en la base son **ilustrativos**; hay que
  cargar los reales antes de cualquier cálculo que se dé por válido (RN-01, RN-03).

---

## Bitácora

| Fecha | Hito |
|-------|------|
| 2026-09-07 | Scaffolding inicial del repo: estructura, contexto, skills, config MCP. Sin código de negocio. |
| 2026-09-07 | Verificación del esquema real de `dbFletway` vía MCP (solo lectura). Esquema OK salvo recuento de tablas (34 doc → 35 real) y de políticas en `CLAUDE.md`. Informe: `docs/VERIFICACION_ESQUEMA_2026-09-07.md`. No se modificó el esquema ni las docs de base de datos. |

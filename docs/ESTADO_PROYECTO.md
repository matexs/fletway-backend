# Estado del proyecto — fletway-backend

> Foto viva del avance. Actualizar al cerrar cada bloque de trabajo (skill
> `update-project-state`). Fechas en formato absoluto.

**Última actualización:** 2026-09-07 · **Etapa:** scaffolding inicial

---

## Resumen de una línea

Esquema de datos desplegado y documentado; ERS cerrada; **código Go todavía no empezado** —
este repo tiene la estructura, los archivos de contexto y las skills, pero cero endpoints
implementados.

---

## Qué está hecho

| Área | Estado | Notas |
|------|--------|-------|
| Base de datos (Supabase `dbFletway`) | ✅ Desplegada | 34 tablas, RLS activo en todas, catálogos con datos semilla. Documentada en `docs/DOCUMENTACION_BASE_DE_DATOS.md`. |
| ERS | ✅ Cerrada | 24 RF, 8 RN, 4 RNF, 5 RI. `docs/ERS_Fletway.pdf`. |
| Estructura del repo Go | ✅ Scaffolding | `cmd/`, `internal/platform/`, `internal/feature/`, `migrations/`, `qa/`, `docs/`. |
| Archivos de contexto | ✅ | `CLAUDE.md`, este archivo, `DECISIONES_TECNICAS.md`, `TRAZABILIDAD.md`, `ENDPOINTS.md`. |
| Skills de Claude Code | ✅ | `supabase-migration`, `trace-requirement`, `scaffold-endpoint`, `rls-policy-review`, `update-project-state`. |
| Config MCP Supabase | ✅ | `.claude/mcp-notes.md` (solo lectura por defecto; escritura con confirmación). |
| Endpoints de negocio | ❌ No empezado | Solo `GET /healthz` y `GET /readyz` de ejemplo. |
| Verificación de módulos Go | ⚠️ Pendiente | `go mod tidy` no corrido (falta resolver dependencias en red). |

---

## Pendientes / próximos pasos (ordenados)

1. **Verificar esquema real vía MCP de Supabase.** El MCP no estaba conectado en el
   scaffolding. Correr `list_tables`, `list_migrations`, `get_advisors` y diferenciar
   contra `docs/DOCUMENTACION_BASE_DE_DATOS.md`. Registrar diferencias, no "arreglarlas".
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

- **MCP de Supabase no conectado** en la sesión de scaffolding → verificación de esquema
  diferida (pendiente #1).
- **Toolchain:** `go` 1.27.1 disponible; `flutter` NO (no afecta a este repo).
- Faltan credenciales reales de la pasarela de pagos (Mercado Pago / Stripe) — RI-03.
- Valores de `config_tarifa` / `config_comision` en la base son **ilustrativos**; hay que
  cargar los reales antes de cualquier cálculo que se dé por válido (RN-01, RN-03).

---

## Bitácora

| Fecha | Hito |
|-------|------|
| 2026-09-07 | Scaffolding inicial del repo: estructura, contexto, skills, config MCP. Sin código de negocio. |

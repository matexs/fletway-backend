# Estado del proyecto — fletway-backend

> Foto viva del avance. Actualizar al cerrar cada bloque de trabajo (skill
> `update-project-state`). Fechas en formato absoluto.

**Última actualización:** 2026-09-26 · **Etapa:** diseño de cotización y cálculo de viajes cerrado + esquema migrado + convenciones de código definidas

---

## Resumen de una línea

Esquema de datos desplegado, migrado para el nuevo modelo de precio y documentado; ERS cerrada;
algoritmos de cotización y cálculo de viajes diseñados; convenciones de código definidas
(`CLAUDE.md` §5); **código Go todavía no empezado** —
este repo tiene la estructura, los archivos de contexto y las skills, pero cero endpoints
implementados.

---

## Qué está hecho

| Área | Estado | Notas |
|------|--------|-------|
| Base de datos (Supabase `dbFletway`) | Desplegada · Migrada (2026-09-24) · Verificada vía MCP | **39** tablas, RLS activo en todas, **127** políticas 100 % con `(select auth.uid())`, triggers `trg_proteger_campos_*` OK, sin enums de Postgres. 20 migraciones registradas (13 iniciales + `0001`–`0007`). Documentada en `docs/DOCUMENTACION_BASE_DE_DATOS.md` (actualizada 2026-09-24, incluye historial de cambios). |
| Diseño de cotización (RN-01) y cálculo de viajes (RN-02) | Documentado | `docs/ALGORITMO_COTIZACION.md` y `docs/ALGORITMO_VIAJES_EMPAQUETADO.md`. Sin cotización estimada al publicar: el único precio es el de cada oferta. Viajes estimados con `bavix/boxpacker3/v2` (greedy). Pseudocódigo de empaquetado compilado y probado contra la v2 en un prototipo descartable. |
| Convenciones de código | Definidas (2026-09-26) | `CLAUDE.md` §5: estilo y golangci-lint, estructura de paquetes y capas, errores y logging, godoc obligatorio, nombres, testing y prohibición de emojis. Sin emojis en ningún archivo versionado. |
| Versión de Go | Unificada en 1.27 (2026-09-26) | `go.mod` y CI en **1.27**; golangci-lint v2.13.2. |
| ERS | Cerrada | 24 RF, 8 RN, 4 RNF, 5 RI. `docs/ERS_Fletway.pdf`. |
| Estructura del repo Go | Scaffolding | `cmd/`, `internal/platform/`, `internal/feature/`, `migrations/`, `qa/`, `docs/`. |
| Archivos de contexto | Hecho | `CLAUDE.md`, este archivo, `DECISIONES_TECNICAS.md`, `TRAZABILIDAD.md`, `ENDPOINTS.md`, `ALGORITMO_COTIZACION.md`, `ALGORITMO_VIAJES_EMPAQUETADO.md`. |
| Skills de Claude Code | Hecho | `supabase-migration`, `trace-requirement`, `scaffold-endpoint`, `rls-policy-review`, `update-project-state`. |
| Config MCP Supabase | Hecho | `.claude/mcp-notes.md` (solo lectura por defecto; escritura con confirmación). |
| Endpoints de negocio | No empezado | Solo `GET /healthz` y `GET /readyz` de ejemplo. |
| Verificación de módulos Go | Pendiente | `go mod tidy` no corrido (falta resolver dependencias en red). |

---

## Pendientes / próximos pasos (ordenados)

1. ~~**Verificar esquema real vía MCP de Supabase.**~~ **Hecho 2026-09-07.** Corridos
   `list_tables`, `list_migrations`, `list_extensions`, `get_advisors` (security + performance)
   y consultas de solo lectura sobre `pg_class` / `pg_policies` / `pg_trigger` / `pg_proc` /
   `pg_type` / `pg_index`. Informe completo: **`docs/VERIFICACION_ESQUEMA_2026-09-07.md`**.
   Resultado: esquema **coincide con la doc en todo lo estructural**. Diferencias registradas
   (no corregidas): **D-1** la doc dice «34 tablas» y hay **35** (error de recuento, ninguna
   tabla sobra ni falta); **D-2** `CLAUDE.md` dice «34 políticas RLS» y hay **109**. Ambas
   **resueltas 2026-09-24** al actualizar la doc tras las migraciones `0001`–`0007` (39 tablas,
   127 políticas). Advisors: 4 WARN de seguridad
   (funciones helper `SECURITY DEFINER` ejecutables por `anon`/`authenticated`) + 43 INFO
   `unused_index` (base vacía, esperable).
2. ~~**Rediseñar cotización y cálculo de viajes y migrar el esquema.**~~ **Hecho 2026-09-24.**
   Migraciones `0001`–`0007` aplicadas con confirmación explícita (ver bitácora).
   Quedan abiertos (detalle en `docs/ALGORITMO_COTIZACION.md` §8):
   - Ubicación de `margen_pct` (config de plataforma vs. por Transportista).
   - Relación `solicitud.cantidad_ayudantes_solicitados` ↔ `oferta.cantidad_ayudantes`.
   - Ambigüedades de fórmula: `tiempo_espera_min`, eficiencia de ayudantes, costos adicionales
     por viaje u oferta, horas del viaje de vuelta, proveedor de ruteo.
   - Tope de unidades por solicitud y timeout del cálculo (empaquetado).
   - Cargar dimensiones en las 5 filas de `objeto` y después `SET NOT NULL`.
   - Riesgos de RLS en `oferta` (lectura del desglose por el Cliente; UPDATE sin trigger de protección).
   - DROP posterior de lo deprecado (`config_tarifa`, snapshots de tarifa plana, etc.).
3. **Confirmar las decisiones marcadas "propuesta a confirmar"** en
   `docs/DECISIONES_TECNICAS.md` (acceso a datos, router, verificación de JWT, módulo Go,
   estrategia de jobs async, pasarela de pagos).
4. `go mod tidy` + fijar versiones de `pgx`, lib de JWT, `github.com/bavix/boxpacker3/v2@v2.0.0`, etc.
5. Implementar `internal/platform/auth` (verificación real del JWT de Supabase) y
   `internal/platform/database` (pool + helper de RLS pass-through).
6. Primer endpoint de negocio real con la skill `scaffold-endpoint` — candidato:
   **RF-05 (registro de Cliente)** o **RF-16 (registro de Transportista)**.
7. Colección Postman base en `qa/postman/` y primeros casos de prueba en `qa/casos-prueba/`.
8. Completar el godoc del código existente que no lo tiene (deuda anotada en `CLAUDE.md` §5, por
   ejemplo `ErrorDetail` y los constructores de `internal/platform/httpx`). El lint no lo
   verifica; se controla en code review.
9. Avisar al equipo que el módulo volvió a Go 1.27. Quien tenga Go 1.24 instalado recibe el
   toolchain 1.27 automáticamente (`GOTOOLCHAIN=auto`).

---

## Bloqueos conocidos

- ~~**MCP de Supabase no conectado**~~ → resuelto 2026-09-07; esquema verificado
  (`docs/VERIFICACION_ESQUEMA_2026-09-07.md`).
- ~~**Recuento de tablas en la doc de base de datos (D-1)**~~ → resuelto 2026-09-24: la doc
  se actualizó con el esquema migrado y dice **39** tablas, verificado contra el catálogo.
- ~~**`CLAUDE.md` §4 decía «34 políticas RLS»** (D-2)~~ → resuelto 2026-09-24: corregido a
  127 políticas en 39 tablas, verificado contra el catálogo.
- **4 WARN de seguridad (advisors)** sobre `fn_es_administrador()` /
  `fn_es_transportista_habilitado()` (`SECURITY DEFINER` ejecutables por `anon`/`authenticated`).
  Intencional como helpers de RLS; revisar en el bloque de hardening antes de exponer la app.
- **Toolchain:** `go` 1.27.1 disponible; `flutter` NO (no afecta a este repo). El módulo y el CI
  usan **Go 1.27** (ver bitácora 2026-09-26).
- Faltan credenciales reales de la pasarela de pagos (Mercado Pago / Stripe) — RI-03.
- Valores de `config_costo_laboral`, `config_operacion`, `config_impuesto` y `config_comision`
  son **ilustrativos**; hay que cargar los reales antes de cualquier cálculo que se dé por válido
  (RN-01, RN-03). **Atención:** Confirmar la ART: el texto del documento fuente dice 18 %, la constante 10 %
  (la seed usa 10 %). `config_tarifa` está deprecada.
- Las 5 filas del catálogo `objeto` no tienen dimensiones: bloquea usar el catálogo al publicar
  solicitudes (RF-06) hasta que se carguen.
- ~~`fletway-mobile/docs/API_CONTRATOS.md` definía `POST /solicitudes` → `CotizacionEstimada`~~ →
  resuelto 2026-09-26 en `fletway-mobile` (PR #1): el contexto de la app ya no tiene cotización
  estimada.

---

## Bitácora

| Fecha | Hito |
|-------|------|
| 2026-09-07 | Scaffolding inicial del repo: estructura, contexto, skills, config MCP. Sin código de negocio. |
| 2026-09-07 | Verificación del esquema real de `dbFletway` vía MCP (solo lectura). Esquema OK salvo recuento de tablas (34 doc → 35 real) y de políticas en `CLAUDE.md`. Informe: `docs/VERIFICACION_ESQUEMA_2026-09-07.md`. No se modificó el esquema ni las docs de base de datos. |
| 2026-09-24 | Diseño de cotización y cálculo de viajes: `docs/ALGORITMO_COTIZACION.md` y `docs/ALGORITMO_VIAJES_EMPAQUETADO.md`, adaptados de los borradores contra el schema real. Decisiones: sin cotización estimada; sin tramo de acercamiento; costos de vehículo en tabla privada `vehiculo_costo`; `peso_maximo_kg` = carga útil; tablas `config_*` nuevas y `config_tarifa` deprecada; deprecar ahora y hacer DROP después. Empaquetado con boxpacker3 **v2** (greedy, sin finishers, 60 % de apoyo) tras medir la v1 y la v2. |
| 2026-09-24 | Migraciones `0001`–`0007` **aplicadas** en `dbFletway` con confirmación humana explícita. 35 → 39 tablas, 109 → 127 políticas. Verificado: RLS en todas, sin `auth.uid()` desnudo, trigger de `viaje` actualizado, seeds cargadas; advisors sin hallazgos nuevos (siguen los 4 WARN previos + 42 INFO `unused_index`). Actualizados `DOCUMENTACION_BASE_DE_DATOS.md`, `CLAUDE.md` (incluye D-2: 34 → 127 políticas) y `TRAZABILIDAD.md`. |
| 2026-09-24 | Revisión de consistencia de todo el repo contra el esquema migrado: `README.md`, `migrations/README.md`, `ENDPOINTS.md` (RF-06 sin cotización estimada; oferta y aceptación con desglose), `DECISIONES_TECNICAS.md` (D-02 actualizado; nuevas D-13 modelo de precio, D-14 boxpacker3 v2, D-15 `margen_pct` abierta), `.claude/mcp-notes.md` y skills `supabase-migration`, `rls-policy-review`, `trace-requirement`, `scaffold-endpoint`. |
| 2026-09-25 | PR #1 (rediseño de cotización y cálculo de viajes: migraciones `0001`–`0007`, algoritmos, doc de base de datos y contexto) mergeado a `main`. CI verde. |
| 2026-09-26 | Versión de Go unificada en **1.27** (`go.mod` y `.github/workflows/ci.yml`; estaban en 1.24 desde `4cb404b`, mientras el README decía 1.27+ y boxpacker3 v2 pide 1.25 o más). Causa de la baja original: la corrida de `454d68b` falló porque golangci-lint **v1.64.8** está compilado con Go 1.24 y no acepta un módulo 1.27. Ya no aplica: el CI usa golangci-lint **v2.13.2**, compilado con Go 1.27.0. Verificado localmente con ese binario exacto: 0 issues; `gofmt`, `go vet`, `go build` y `go test -race` OK. PR #2. |
| 2026-09-26 | Convenciones de código en `CLAUDE.md` §5 (estilo y lint, capas, errores y logging, godoc, nombres, testing, prohibición de emojis). Emojis reemplazados por texto en toda la documentación: los estados de `TRAZABILIDAD.md` pasan a `NO` / `EN CURSO` / `OK` / `N/A`, también en las skills. PR #2 (junto con Go 1.27) mergeado; CI de `main` verde. |
| 2026-09-26 | Emojis quitados de los comentarios de `migrations/0001`–`0007` y de la salida de `scripts/pre-commit` (sólo comentarios: el SQL aplicado en `dbFletway` no cambia; verificado que la base no tiene emojis). PR #3 mergeado; CI de `main` verde. No quedan ramas secundarias. |

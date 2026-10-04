# Estado del proyecto — fletway-backend

> Foto viva del avance. Actualizar al cerrar cada bloque de trabajo (skill
> `update-project-state`). Fechas en formato absoluto.

**Última actualización:** 2026-10-02 · **Etapa:** construcción; módulos 0 y 1 terminados

---

## Resumen de una línea

Módulos 0 a 7 de `docs/PLAN_CONSTRUCCION.md` construidos, probados y mergeados (identidad,
habilitación, vehículos, catálogo, solicitudes y matchmaking); módulo 8 (oferta, precio y cálculo
de viajes) con backend y app hechos y la migración `0016` aplicada en `dbFletway`.

---

## Qué está hecho

| Área | Estado | Notas |
|------|--------|-------|
| Base de datos (Supabase `dbFletway`) | Desplegada · Migrada (2026-09-24) · Verificada vía MCP | **39** tablas, RLS activo en todas, **127** políticas 100 % con `(select auth.uid())`, triggers `trg_proteger_campos_*` OK, sin enums de Postgres. 20 migraciones registradas (13 iniciales + `0001`–`0007`). Documentada en `docs/DOCUMENTACION_BASE_DE_DATOS.md` (actualizada 2026-09-24, incluye historial de cambios). |
| Diseño de cotización (RN-01) y cálculo de viajes (RN-02) | Documentado | `docs/ALGORITMO_COTIZACION.md` y `docs/ALGORITMO_VIAJES_EMPAQUETADO.md`. Sin cotización estimada al publicar: el único precio es el de cada oferta. Viajes estimados con `bavix/boxpacker3/v2` (greedy). Pseudocódigo de empaquetado compilado y probado contra la v2 en un prototipo descartable. |
| Convenciones de código | Definidas (2026-09-26) | `CLAUDE.md` §5: estilo y golangci-lint, estructura de paquetes y capas, errores y logging, godoc obligatorio, nombres, testing y prohibición de emojis. Sin emojis en ningún archivo versionado. |
| Versión de Go | Unificada en 1.27 (2026-09-26) | `go.mod` y CI en **1.27**; golangci-lint v2.13.2. |
| ERS | Cerrada | 24 RF, 8 RN, 4 RNF, 5 RI. `docs/ERS_Fletway.docx`. |
| Estructura del repo Go | Scaffolding | `cmd/`, `internal/platform/`, `internal/feature/`, `migrations/`, `qa/`, `docs/`. |
| Archivos de contexto | Hecho | `CLAUDE.md`, este archivo, `DECISIONES_TECNICAS.md`, `TRAZABILIDAD.md`, `ENDPOINTS.md`, `ALGORITMO_COTIZACION.md`, `ALGORITMO_VIAJES_EMPAQUETADO.md`. |
| Skills de Claude Code | Hecho | `supabase-migration`, `trace-requirement`, `scaffold-endpoint`, `rls-policy-review`, `update-project-state`. |
| Config MCP Supabase | Hecho | `.claude/mcp-notes.md` (solo lectura por defecto; escritura con confirmación). |
| Plataforma (módulo 1) | Hecho (2026-10-02) | Conexión `pgx` con RLS pass-through (`WithinTx`) y verificación del JWT ES256 contra el JWKS de Supabase. Tests unitarios y de integración contra Supabase local. |
| Entorno local | Hecho (2026-10-02) | Supabase CLI + Docker, firma ES256 local, `make db-local` (esquema base de `dbFletway` en `migrations/baseline/` más migraciones posteriores) y `make test-integracion`. |
| Datos semilla (módulo 0) | Hecho (2026-10-02) | 14 zonas piloto y catálogo de 28 objetos con medidas (`0008`, `0009`), aplicados en local y en `dbFletway`. Los tipos de vehículo se reemplazan en el módulo 4. |
| Identidad (módulo 2) | En curso (2026-10-02) | Backend hecho: `GET /api/me`, `POST /api/auth/registro/cliente` y `/transportista` con tests de integración. Migraciones `0010` (corrige recursión de RLS `solicitud`/`oferta`) y `0011` (trigger de alta y protecciones) probadas en local y aplicadas en `dbFletway` con confirmación explícita. Primer Administrador creado en `dbFletway` (bootstrap de D-18, 2026-10-02, con confirmación). La app (login, registro, rol desde `/me`) está en `fletway-mobile`; falta probarla en un emulador. |
| Habilitación (módulo 3) | En curso (2026-10-03) | Backend hecho: carga y listado de documentos del Transportista, revisión del Administrador con URLs firmadas de Storage, notificación `documentacion_revisada` por la interfaz `Notificador` (D-22). Migración `0012` probada en local y aplicada en `dbFletway` con confirmación explícita. App en `fletway-mobile`, probada en el emulador. Reglas de cierre en D-33. |
| Vehículos, costos y zonas (módulo 4) | En curso (2026-10-03) | Backend hecho: tipos de vehículo con medidas estándar, alta y listado de vehículos, activar o desactivar, costos en un segundo paso, catálogo de zonas, zonas del Transportista y disponibilidad. Montos y medidas como decimales exactos (paquete `decimal`, D-11). Migración `0013` probada en local y aplicada en `dbFletway` con confirmación explícita. App en `fletway-mobile`, probada en el emulador. |
| Catálogo de objetos (módulo 5) | Hecho (2026-10-03) | `GET /api/catalogo/objetos` y el selector de la app, probados en el emulador y mergeados. |
| Solicitud (módulo 6) | Hecho (2026-10-03) | Backend hecho: publicar (objetos del catálogo y manuales, coordenadas por `Geocodificador` aproximado), listar con vencimiento al leer, detalle, cancelar y republicar. Migración `0014` (fecha y franja; sin edición, objetos fijos y copia del catálogo por triggers) probada en local y aplicada en `dbFletway` con confirmación explícita (advisors sin hallazgos nuevos). App en `fletway-mobile`. |
| Matchmaking (módulo 7) | Hecho (2026-10-03) | Backend hecho: `GET /api/transportista/solicitudes` y aviso in-app asíncrono al publicar y republicar. La regla D-21 vive en la base (`fn_es_compatible`, migración `0015`, probada en local y aplicada en `dbFletway` con confirmación explícita; advisors: dos WARN esperables por las funciones expuestas a usuarios logueados). App en `fletway-mobile`, probada en el emulador. |
| Oferta, precio y viajes (módulo 8) | En curso (2026-10-03) | Backend hecho: cotizar sin guardar, ofertar, retirar y listar las propias. Costos del vehículo de referencia por tipo (`config_costo_vehiculo`, D-34, migración `0017` aplicada en `dbFletway` con confirmación explícita). Funciones puras de precio (`cotizacion.go`) y de viajes con boxpacker3 v2 (`viajes.go`), con tests table-driven; interfaz `Ruteador` (`aproximado` en local, `fijo` en tests; en producción exige `google`). Migración `0016` (`config_margen` con seed 0 %, `oferta_costo`, protección de la oferta) probada en local y aplicada en `dbFletway` con confirmación explícita (advisors sin hallazgos nuevos). App en `fletway-mobile`. |
| Top 3, perfil y aceptación (módulo 9) | En curso (2026-10-03) | Backend hecho: ofertas de una solicitud ordenadas por score (función pura, tests de `ALGORITMO_SCORE.md` §5), perfil público del Transportista y aceptación con `fn_aceptar_oferta`. Migración `0018` (`viaje_costo`, `viaje_pin`, el viaje sólo nace de la aceptación) probada en local y aplicada en `dbFletway` con confirmación explícita (advisors: un WARN esperable por `fn_aceptar_oferta` ejecutable por usuarios logueados; verifica que quien llama sea el Cliente). App en `fletway-mobile`. |
| Dependencias Go | Hecho (2026-10-03) | `pgx/v5`, `golang-jwt/jwt/v5`, `keyfunc/v3`, `testify`, `bavix/boxpacker3/v2`. Módulo `github.com/matexs/fletway-backend` (D-05). |

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
3. ~~**Definir lo que faltaba para construir.**~~ **Hecho 2026-10-01.** Las 61 preguntas de
   `PENDIENTES_ANTES_DE_CONSTRUIR` quedaron resueltas con las soluciones acordadas por el equipo:
   decisiones D-03 a D-31 confirmadas en `docs/DECISIONES_TECNICAS.md`, score en
   `docs/ALGORITMO_SCORE.md` y orden de trabajo en `docs/PLAN_CONSTRUCCION.md`. Los puntos de
   la lista anterior quedan cubiertos por el plan.
4. ~~**Resolver los puntos A-1 a A-7**~~ **Hecho 2026-10-01** (`docs/PLAN_CONSTRUCCION.md` §1.1).
5. **Setup** (`PLAN_CONSTRUCCION.md` §1.2): Supabase local con el schema base, credenciales
   locales, branch protection en GitHub, primer Administrador. El despliegue a producción no es
   parte de esta etapa.
6. **Construir los módulos 0 a 14** en el orden de `docs/PLAN_CONSTRUCCION.md` §4. Empieza por el
   módulo 0 (seeds) y el módulo 1 (plataforma: `pgx`, JWT, módulo `github.com/matexs/fletway-backend`).
7. Completar el godoc del código existente que no lo tiene (deuda de `CLAUDE.md` §5, por ejemplo
   `ErrorDetail` y los constructores de `internal/platform/httpx`).
8. Avisar al equipo que el módulo volvió a Go 1.27 (`GOTOOLCHAIN=auto` descarga el toolchain).

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
- Faltan las credenciales del sandbox de Mercado Pago (D-09) — se necesitan recién en el módulo 12.
- Los valores de `config_*` son los **valores de trabajo** para desarrollar y probar (D-24); cambiarlos
  por cifras definitivas es un `UPDATE`, no bloquea. ART adoptada: 10 %. `config_tarifa` está deprecada.
- Las 5 filas del catálogo `objeto` no tienen dimensiones: las completa la seed del módulo 0
  (`PLAN_CONSTRUCCION.md` §2.2), junto con 23 objetos y 14 zonas piloto.
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
| 2026-10-01 | Definiciones para empezar la construcción: se incorporaron las soluciones acordadas por el equipo a los 61 pendientes. `DECISIONES_TECNICAS.md`: D-03 a D-12 confirmadas, D-09 Mercado Pago, D-15 margen de plataforma (`config_margen`) y nuevas D-16 a D-31 (entornos local y producción, alcance de la etapa, alta y rol, Storage, solicitud con republicación, matchmaking, notificaciones in-app, oferta, ajustes de precio, score sin cercanía, PIN que sólo ve el Cliente, cancelaciones, pagos, reputación, veto y proceso). Nuevos `ALGORITMO_SCORE.md` y `PLAN_CONSTRUCCION.md`; actualizados `ALGORITMO_COTIZACION.md`, `ALGORITMO_VIAJES_EMPAQUETADO.md`, `ENDPOINTS.md` (rutas bajo `/api`), `TRAZABILIDAD.md` (enmiendas a la ERS), `CLAUDE.md` y `README.md`. Sin cambios en la base. |
| 2026-10-01 | Resueltos los puntos A-1 a A-7: `tipo_vehiculo` con 6 tipos y medidas estándar de referencia (D-32), color primario `#C36224` con secundarios grises y fondo blanco, Render a priori y esta etapa sólo local (D-16), desglose en `oferta_costo`/`viaje_costo` con `fn_aceptar_oferta` (D-23), PIN en `viaje_pin` con `fn_validar_pin` (D-26), proveedores de ruteo `google`/`aproximado`/`fijo` (D-24). ERS corregida: `docs/ERS_Fletway.docx` sin cercanía y con el anexo "Registro de cambios"; el `.docx` reemplaza al PDF en el repo. |
| 2026-10-02 | Setup y módulo 1. Protección de `main` en los dos repos (1 aprobación, bypass de admin; D-31). Entorno local con Docker y Supabase CLI, firma ES256 local y `make db-local`. Módulo 1 del backend: módulo Go renombrado, `pgx` con RLS pass-through y verificación del JWT ES256 sin HS256. Verificado de punta a punta contra Supabase local: tests de integración de `WithinTx`, 401 sin token o con token adulterado, token real aceptado. |
| 2026-10-02 | Módulo 0. Esquema base de `dbFletway` en `migrations/baseline/` (extensiones, estructura y datos de catálogo; producción sin datos de personas): la base local queda igual a producción (39 tablas, 127 policies). Seeds `0008` (14 zonas) y `0009` (28 objetos, medidas `NOT NULL`), probadas en local y aplicadas en `dbFletway` con confirmación explícita. Advisors sin hallazgos nuevos. |
| 2026-10-02 | Módulo 2 (backend). Se detectó que en `dbFletway` las policies `solicitud_select` y `oferta_select` se referencian entre sí (recursión infinita en toda lectura autenticada de `solicitud`, `oferta`, `usuario`, `cliente`, `solicitud_objeto`): migración `0010`. Además, un usuario podía cambiar su `usuario.rol` y darse de alta como Transportista `habilitado` vía PostgREST: migración `0011`, con el trigger de alta de D-18. Feature `identidad` (`/api/me`, registros). Probado en local; `0010` y `0011` aplicadas en `dbFletway` con confirmación. Advisors: un WARN nuevo del mismo tipo que los existentes (`fn_tiene_oferta_en_solicitud` ejecutable por RPC; sólo responde sobre las ofertas de quien llama). |
| 2026-10-02 | Bootstrap del primer Administrador en `dbFletway` (D-18): alta por signUp con rol `cliente` y, por SQL sin JWT y con confirmación explícita, `usuario.rol = 'administrador'` y fila en `administrador`. |
| 2026-10-03 | Módulo 2 probado en el emulador Android: tres arreglos de la app (atrás desde el registro, revalidación, mensaje sin perfil) y `compileSdk` 37. Módulo 3 (backend): D-33 confirmada; migración `0012` (Storage, protección de documentos y estado derivado); feature `habilitacion`, paquete `storage` y `Notificador` in-app. Probado en local, incluida la descarga con la URL firmada. |
| 2026-10-03 | `0012` aplicada en `dbFletway` con confirmación explícita. Advisors: sin hallazgos por la migración; aparece "Leaked password protection disabled" (configuración de Supabase Auth, pendiente de decidir). Módulo 3 probado en el emulador. |
| 2026-10-03 | Módulo 4 (backend): migración `0013` (tipos de vehículo, D-32), features `vehiculo` y `geografia`, disponibilidad en `identidad` (y `disponible` en `/me`). Test de terminado: sólo el dueño y el Administrador leen `vehiculo_costo`. Riesgo anotado para el módulo 9: la patente es visible para cualquier autenticado por RLS. |
| 2026-10-03 | `0013` aplicada en `dbFletway` con confirmación explícita (6 tipos de vehículo). Módulo 4 probado en el emulador: disponibilidad, alta con medidas propuestas, costos (valores exactos en la base), activar y desactivar, y zonas. |
| 2026-10-03 | Módulo 4 probado en el emulador (también el Transportista pendiente y la patente duplicada) y mergeado. Módulo 5 (backend): feature `catalogo`. |
| 2026-10-03 | Módulo 5 probado en el emulador (con GPU por software: la de hardware hacía caer a qemu) y mergeado. Se decide terminar primero los flujos y hacer la pasada de UI/UX en el módulo 15, con una decoración mínima por pantalla mientras tanto. Módulo 6 (backend): migración `0014`, feature `solicitud`, `Geocodificador`. Se cerraron por trigger: edición de solicitudes, cambios de objetos con ofertas y objetos de catálogo con medidas falsas. |
| 2026-10-03 | Módulo 6 aplicado (`0014`), probado en el emulador y mergeado. Módulo 7 (backend): migración `0015` y feature `matchmaking`; test de terminado con los cinco casos excluidos (sin zona, sin capacidad o vehículo inactivo, no disponible, no habilitado, vetado). |
| 2026-10-03 | Módulo 7 mergeado. Módulo 8 (backend): migración `0016` y feature `oferta`. Además de lo que pedía el plan: el desglose es obligatorio al crear la oferta (trigger diferido: sólo el backend crea ofertas, PostgREST no puede inventar un precio), una oferta vigente por vehículo para poder retirar y volver a ofertar (D-23), y `POST .../ofertas/cotizar` para ver el precio antes de ofertar. Test de terminado: el Cliente no lee el desglose y nadie modifica el precio. |
| 2026-10-03 | `0016` aplicada en `dbFletway` con confirmación explícita (0 ofertas y 0 viajes antes de mover las columnas). Advisors sin hallazgos nuevos. |
| 2026-10-03 | D-34: los costos del vehículo pasan a ser de referencia por tipo, definidos por la plataforma (el Transportista podía inflarlos). Migración `0017` (`config_costo_vehiculo` con valores de trabajo para los 6 tipos) probada en local; se retiran los endpoints de costos del vehículo y `vehiculo_costo` queda deprecada. Los tests de oferta pasan a zonas propias (Tigre, San Fernando, Escobar) para no cruzarse con los de matchmaking cuando corren en paralelo. |
| 2026-10-03 | `0017` aplicada en `dbFletway` con confirmación explícita (6 costos de referencia). Advisors sin hallazgos nuevos. |
| 2026-10-03 | Módulo 8 mergeado. Módulo 9 (backend): migración `0018`, score, listado para el Cliente, aceptación y feature `perfil`. Se cerró que el Cliente podía crear un viaje por PostgREST. Hallazgos anotados: `usuario_select` expone email y teléfono de los Transportistas a cualquier autenticado; `trg_proteger_campos_transportista` descarta los cambios de calificación y cumplimiento de quien no sea Administrador, así que el recálculo del módulo 13 tiene que hacerse desde un trigger o una función que lo contemple. |
| 2026-10-03 | `0018` aplicada en `dbFletway` con confirmación explícita (0 viajes antes de mover las columnas). Advisors: un WARN nuevo del tipo conocido (`fn_aceptar_oferta`). |
| 2026-10-04 | Módulo 9 mergeado. D-35: direcciones, ruteo y mapas con Geoapify y OpenStreetMap, gratis y sin tarjeta, en reemplazo de Google. Backend: cliente `geoapify`, proveedores `geoapify` de `Geocodificador` y `Ruteador` (con caché), autocompletado de direcciones y ruta de una solicitud para el mapa. El precio de las ofertas pasa a usar la distancia por calles real. |

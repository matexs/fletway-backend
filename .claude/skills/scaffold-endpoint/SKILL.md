---
name: scaffold-endpoint
description: >-
  Crear un endpoint HTTP nuevo en el backend Go de Fletway siguiendo la
  convención del repo: paquete por feature en internal/feature/<ctx>/ con
  routes/handler/service/repository/dto, middleware Auth obligatorio (RNF-01),
  acceso a datos vía db.WithinTx con RLS pass-through (D-02), y side-effects
  pesados delegados al pool async (RNF-02). Registra el endpoint en
  docs/ENDPOINTS.md y docs/TRAZABILIDAD.md. Usar al implementar cualquier RF que
  necesite un endpoint REST nuevo.
---

# Skill: scaffold-endpoint

## Cuándo usar

Al implementar un RF que expone un endpoint REST nuevo (o una operación nueva
sobre un recurso existente).

## Convención del repo (recordatorio)

```
internal/feature/<ctx>/
  routes.go      func Register(mux *http.ServeMux, deps...) — monta las rutas
  handler.go     HTTP: decodifica, valida forma, llama al service, responde
  service.go     lógica de negocio + orquestación; encola jobs async
  repository.go   queries pgx, SIEMPRE dentro de db.WithinTx(ctx, identity, ...)
  dto.go         request/response structs (fuente para docs/ENDPOINTS.md)
  <ctx>_test.go  tests: camino feliz + validación + RLS
```

`<ctx>` sale de los dominios de `docs/DOCUMENTACION_BASE_DE_DATOS.md`:
`identidad`, `geografia`, `vehiculo`, `solicitud`, `oferta`, `viaje`, `pago`,
`resena`, `incidente`, `notificacion`, `catalogo`.

## Procedimiento

### 1. Ubicar

- ¿Qué RF implementa? ¿Qué RN toca? (mirar `docs/TRAZABILIDAD.md`).
- ¿Qué contexto/paquete? ¿Existe `internal/feature/<ctx>/` o hay que crearlo?
- ¿Qué tablas? Leer sus columnas y policies reales (MCP, solo lectura) —
  identificadores en español, tal cual la base.
- Definir método + path según `docs/ENDPOINTS.md` (envelope de error único, D-11).

### 2. `dto.go`

Structs de request y response con tags `json`. Reglas:

- El cliente **no** envía campos calculados por el sistema: precio (RN-01),
  cantidad de viajes (RN-02), `%` comisión (RN-03), score (RN-05), PIN (RN-06),
  ni ningún `*_snapshot`.
- Validación de forma en un método `(req *XxxRequest) Validate() error` que
  devuelve `*httpx.APIError` (`BadRequest`).

### 3. `repository.go`

```go
func (r *Repo) Crear(ctx context.Context, id database.Identity, in CrearParams) (Row, error) {
    var out Row
    err := r.db.WithinTx(ctx, id, func(tx any) error {
        // t := tx.(pgx.Tx)  // cuando pgx esté cableado
        // queries acá — RLS ya está aplicado por WithinTx (SET LOCAL ...)
        return nil
    })
    return out, err
}
```

- **Toda** query pasa por `WithinTx` con la identidad del request. Nunca queries
  sueltas contra el pool sin contexto RLS.
- Nombres de tablas/columnas exactos (español, snake_case).
- Columnas de ownership/estado en `WHERE` → asumir que ya hay índice; si no,
  abrir una migración con `supabase-migration`.

### 4. `service.go`

- Orquesta: valida reglas de negocio que RLS no expresa, arma snapshots, calcula
  lo que la RN pida.
- Side-effects pesados o no críticos para la respuesta (notificar transportistas
  RN-05, recalcular tasas, llamar a la pasarela de pagos si aplica) → `jobs.Enqueue(...)`
  con job **idempotente**. No bloquear el request (RNF-02).
- Errores de negocio → `*httpx.APIError` con `code` estable en snake_case español
  (`solicitud_no_encontrada`, `transportista_no_habilitado`, `pin_invalido`).

### 5. `handler.go`

```go
func (h *Handler) crear(w http.ResponseWriter, r *http.Request) error {
    id, ok := auth.FromContext(r.Context())
    if !ok {
        return httpx.Unauthorized("no_autenticado", "…") // no debería pasar tras el middleware
    }
    var req CrearRequest
    if err := httpx.Decode(r, &req); err != nil { return err }
    if err := req.Validate(); err != nil { return err }

    out, err := h.svc.Crear(r.Context(), id, req.toParams())
    if err != nil { return err }

    httpx.JSON(w, http.StatusCreated, toResponse(out))
    return nil
}
```

### 6. `routes.go`

```go
func Register(mux *http.ServeMux, db *database.DB, jobs async.Enqueuer) {
    h := &Handler{svc: newService(db, jobs)}
    mux.HandleFunc("POST /solicitudes", httpx.Wrap(h.crear))
    mux.HandleFunc("GET /solicitudes/{id}", httpx.Wrap(h.obtener))
}
```

Luego agregar la llamada en `internal/server/server.go` → `registerAPI(apiMux)`.
El sub-mux de negocio ya está detrás del middleware `Auth` (RNF-01) — no hace
falta re-envolver.

### 7. Tests (`<ctx>_test.go`)

Mínimo:

- Camino feliz (201/200 + body esperado).
- Validación (400 con el `code` correcto ante body inválido).
- **RLS:** un usuario que no es dueño / de otro rol recibe 403/404 y **no** ve
  ni modifica el recurso.
- Si toca una RN: un test que la fija (ej. el `precio_calculado` de la oferta =
  cálculo esperado con `config_*` y `vehiculo_costo` de prueba; ver
  `docs/ALGORITMO_COTIZACION.md`).

### 8. Documentar

- `docs/ENDPOINTS.md`: método, path, auth, request, response, errores, RF/RN.
- `docs/TRAZABILIDAD.md`: pasar el RF a 🟡 o (tras `trace-requirement`) ✅.
- Commit: `feat(<ctx>): <resumen> [RF-XX, RN-YY]`.
  Branch: `feat/RF-XX-<slug>`.

## Checklist

- [ ] Endpoint detrás del middleware `Auth` (RNF-01).
- [ ] Todas las queries dentro de `db.WithinTx(ctx, identity, ...)`.
- [ ] El cliente no envía campos calculados por el sistema.
- [ ] Side-effects no bloqueantes → `jobs.Enqueue` (idempotente).
- [ ] Snapshots escritos donde la RNF-03 lo pide.
- [ ] Errores con `code` estable en español.
- [ ] Tests de camino feliz + validación + RLS.
- [ ] `docs/ENDPOINTS.md` y `docs/TRAZABILIDAD.md` actualizados.

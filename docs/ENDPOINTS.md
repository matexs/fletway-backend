# Contrato de API — fletway-backend

> **Fuente de verdad del contrato REST que consume `fletway-mobile`.**
> La skill `sync-api-models` del repo mobile lee este archivo para mantener los modelos
> Dart en sincronía. Cuando agregues o cambies un endpoint, actualizá acá **primero**.

**Base URL (dev):** `http://localhost:8080`
**Versión de contrato:** `v0` (draft — nada implementado todavía)
**Última actualización:** 2026-09-07

---

## Convenciones

- Todos los endpoints de negocio requieren header `Authorization: Bearer <JWT de Supabase>`
  (RNF-01). Excepción: `/healthz`, `/readyz`.
- Content-Type `application/json` en request y response.
- Identificadores: `uuid` string. Timestamps: ISO-8601 UTC (`2026-09-07T14:03:00Z`).
- Montos: número entero en centavos, o decimal string — **a definir en D-11**.
- Nombres de campos: se exponen tal como en la base (español, snake_case) salvo decisión
  explícita en contrario, para no introducir una capa de traducción propensa a errores.
- Error envelope:
  ```json
  { "error": { "code": "<slug>", "message": "<humano>", "details": {} } }
  ```
- Paginación: `?limit=<n>&cursor=<opaco>`; response incluye `next_cursor` (o `null`).

---

## Endpoints implementados

### `GET /healthz`
Liveness. Sin auth. `200 {"status":"ok"}`.

### `GET /readyz`
Readiness (incluye ping a la base). Sin auth. `200 {"status":"ready","db":"ok"}` /
`503` si la base no responde.

---

## Endpoints planificados (no implementados)

> Se completan a medida que se implementan, con la skill `scaffold-endpoint`.
> Mapa RF → endpoint (borrador, sujeto a D-11):

| RF/RN | Método + path (propuesto) | Rol | Notas |
|-------|---------------------------|-----|-------|
| RF-05 | `POST /auth/registro/cliente` | público→Cliente | crea `usuario` + `cliente`; delega credenciales a Supabase Auth |
| RF-16 | `POST /auth/registro/transportista` | público→Transportista | crea `usuario` + `transportista` (estado `pendiente`) + carga docs |
| RF-18 | `POST /transportista/vehiculos` | Transportista | alta de `vehiculo` |
| RF-06 / RN-01 / RN-02 | `POST /solicitudes` | Cliente | crea `solicitud` + `solicitud_objeto`; devuelve cotización estimada; dispara push async (RN-05) |
| RF-06 | `GET /catalogo/objetos` | autenticado | catálogo RN-08 |
| RF-17 / RN-04 | `GET /transportista/solicitudes` | Transportista | solicitudes compatibles por zona/vehículo |
| RF-17 / RN-01 | `POST /solicitudes/{id}/ofertas` | Transportista habilitado | crea `oferta` con precio calculado |
| RF-07 / RN-05 | `GET /solicitudes/{id}/ofertas` | Cliente | **top 3 por score**, `?ver_mas=true` para el resto |
| RF-07 | `POST /ofertas/{id}/aceptar` | Cliente | confirma `viaje` con snapshots (RNF-03); habilita chat (RI-05) |
| RF-08 / RN-07 | `POST /viajes/{id}/cancelar` (Cliente) | Cliente | sin cargo / con resarcimiento según estado |
| RF-19 | `POST /viajes/{id}/cancelar` (Transportista) | Transportista | baja `tasa_cumplimiento`, sin penalización económica |
| RF-21 | `GET /viajes/{id}` | Cliente/Transportista del viaje | incluye PIN para el Transportista |
| RF-22 / RN-06 | `POST /viajes/{id}/pin-inicio` · `POST /viajes/{id}/pin-fin` | Transportista | valida PIN + ubicación |
| RF-12 / RN-06 | `POST /viajes/{id}/resena` | Cliente | solo con PIN de fin validado |
| RF-11 | `GET /transportistas/{id}` | autenticado | perfil público + reseñas |
| RF-13 / RF-23 | `POST /incidentes` | Cliente/Transportista | alta de `incidente` |
| RF-14 / RF-24 | `GET /viajes?rol=cliente\|transportista` | autenticado | historial |
| RF-09 | `GET /notificaciones` · `POST /notificaciones/{id}/leida` | autenticado | — |
| RF-01 / RF-02 / RF-03 / RF-04 | endpoints de Administrador | Administrador | se detallan cuando arranque ese frente |

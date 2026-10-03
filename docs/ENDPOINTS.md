# Contrato de API — fletway-backend

> **Fuente de verdad del contrato REST que consume `fletway-mobile`.**
> La skill `sync-api-models` del repo mobile lee este archivo para mantener los modelos
> Dart en sincronía. Cuando agregues o cambies un endpoint, actualizá acá **primero**.

**Base URL:** local `http://localhost:8080`; producción, la URL de Render (D-16). Las rutas de negocio van bajo **`/api`** (D-11); los health checks, en la raíz.
**Versión de contrato:** `v0` (draft)
**Última actualización:** 2026-10-02

---

## Convenciones

- Todos los endpoints de negocio requieren header `Authorization: Bearer <JWT de Supabase>`
  (RNF-01). Excepciones: `/healthz`, `/readyz` y el webhook de Mercado Pago (D-28, protegido por
  firma).
- Content-Type `application/json` en request y response.
- Identificadores: `uuid` string. Timestamps: ISO-8601 UTC (`2026-09-07T14:03:00Z`).
- Montos: número JSON decimal con **2 posiciones** (no string, no centavos), redondeado half-up
  una sola vez al calcular el precio final (D-11).
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

### Alta de cuenta (D-18)

1. La app hace `signUp` en Supabase Auth con email, contraseña y la metadata
   `{"rol": "cliente" | "transportista", "nombre_completo": "...", "telefono": "..."}`.
   El trigger `trg_alta_usuario` crea la fila `usuario`. Si `rol` no es `cliente` ni
   `transportista`, o faltan `nombre_completo` o `telefono`, el `signUp` falla y no se crea la
   cuenta.
2. Con la sesión obtenida, la app llama al endpoint de registro de su rol.
3. Desde ahí, el rol y el estado de la cuenta salen siempre de `GET /api/me`, nunca de
   `user_metadata`.

#### Respuesta `Me` (común a los tres endpoints)

```json
{
  "usuario_id": "uuid",
  "email": "cliente@ejemplo.com",
  "nombre_completo": "Clara Cliente",
  "telefono": "1144440000",
  "rol": "cliente | transportista | administrador",
  "activo": true,
  "registro_completo": true,
  "estado_habilitacion": "pendiente | habilitado | rechazado | null"
}
```

- `registro_completo`: `false` mientras falte la fila del rol; la app manda al usuario a
  completar el registro.
- `estado_habilitacion`: sólo para Transportistas registrados; `null` en otro caso.

### `GET /api/me` (RNF-01, D-18)
Perfil del usuario autenticado. `200` con `Me`.

| Status | `code` | Cuándo |
|---|---|---|
| 401 | `no_autenticado` / `token_invalido` | sin JWT o JWT inválido |
| 404 | `usuario_no_encontrado` | el usuario de Auth no tiene fila `usuario` |

### `POST /api/auth/registro/cliente` (RF-05)
Crea la fila `cliente` del usuario autenticado. Sin body. `201` con `Me` si la creó; `200`
con `Me` si ya existía (se puede reintentar).

### `POST /api/auth/registro/transportista` (RF-16)
Crea la fila `transportista` en estado `pendiente` (la habilitación es el módulo 3). Sin body.
`201` / `200` como el de Cliente.

Errores de los dos registros:

| Status | `code` | Cuándo |
|---|---|---|
| 401 | `no_autenticado` / `token_invalido` | sin JWT o JWT inválido |
| 403 | `cuenta_inactiva` | `usuario.activo = false` |
| 404 | `usuario_no_encontrado` | el usuario de Auth no tiene fila `usuario` |
| 409 | `rol_no_corresponde` | el rol del endpoint no es el de `usuario.rol` (un rol por cuenta, D-17) |

---

## Endpoints planificados (no implementados)

> Se completan a medida que se implementan, con la skill `scaffold-endpoint`, siguiendo el orden
> de `docs/PLAN_CONSTRUCCION.md` (columna "Mód."). Todas las rutas llevan el prefijo `/api`.

| Mód. | RF/RN | Método + path | Rol | Notas |
|------|-------|---------------|-----|-------|
| 3 | RF-16 | `POST /api/transportista/documentos` · `GET /api/transportista/documentos` | Transportista | registra el path del archivo ya subido a Storage (D-19) |
| 3 | RF-01 | `GET /api/admin/transportistas?estado=pendiente` | Administrador | operado desde Postman (D-17) |
| 3 | RF-01 | `POST /api/admin/documentos/{id}/aprobar` · `/rechazar` | Administrador | rechazo con motivo; notificación `documentacion_revisada` |
| 4 | RF-18 | `POST /api/transportista/vehiculos` · `GET /api/transportista/vehiculos` | Transportista | medidas útiles + peso (carga útil) |
| 4 | RF-18 | `PUT /api/transportista/vehiculos/{id}/costos` | Transportista | fila de `vehiculo_costo`; sin ella no se puede ofertar con ese vehículo |
| 4 | RN-04 | `GET /api/zonas` · `PUT /api/transportista/zonas` | autenticado / Transportista | catálogo de zonas y selección múltiple |
| 4 | RN-05 | `PUT /api/transportista/disponibilidad` | Transportista | interruptor manual (D-21) |
| 5 | RN-08 | `GET /api/catalogo/objetos` | autenticado | catálogo con medidas y flags |
| 6 | RF-06 | `POST /api/solicitudes` | Cliente | copia peso, medidas y flags de cada objeto; **no calcula ni devuelve monto** (D-13, D-20) |
| 6 | RF-06 | `GET /api/solicitudes` · `GET /api/solicitudes/{id}` | Cliente | "vencida" calculada al vuelo (D-20) |
| 6 | RF-06 | `POST /api/solicitudes/{id}/cancelar` | Cliente | sin costo; ofertas pendientes a `no_seleccionada` |
| 6 | RF-06 | `POST /api/solicitudes/{id}/republicar` | Cliente | sólo si está vencida; crea una nueva con otra fecha |
| 7 | RF-17 / RN-04 | `GET /api/transportista/solicitudes` | Transportista | compatibles según D-21, sin vencidas |
| 8 | RF-17 / RN-01 / RN-02 | `POST /api/solicitudes/{id}/ofertas` | Transportista habilitado | request: `vehiculo_id` + `cantidad_ayudantes` (0..3). Calcula viajes y precio. Si la carga no entra, la ruta no se obtiene o el cálculo vence → error de validación con el motivo, sin oferta |
| 8 | RF-17 | `POST /api/ofertas/{id}/retirar` · `GET /api/transportista/ofertas` | Transportista | la oferta no se edita: se retira y se crea otra (D-23) |
| 9 | RF-07 / RN-05 | `GET /api/solicitudes/{id}/ofertas` | Cliente | **top 3 por score** (`ALGORITMO_SCORE.md`); `?ver_mas=true` pagina. Sólo `precio_calculado`, nunca el desglose |
| 9 | RF-11 | `GET /api/transportistas/{id}` | autenticado | perfil público + reseñas; sin datos financieros |
| 9 | RF-07 | `POST /api/ofertas/{id}/aceptar` | Cliente | vía `fn_aceptar_oferta`: crea el `viaje`, copia el desglose a `viaje_costo`, genera los PIN en `viaje_pin`, las demás ofertas pasan a `no_seleccionada`; habilita chat (RI-05) |
| 10 | RF-21 | `GET /api/viajes/{id}` | Cliente/Transportista del viaje | el Cliente ve los PIN (de `viaje_pin`); el Transportista **nunca** (D-26) |
| 10 | RN-07 | `POST /api/viajes/{id}/salida` | Transportista | setea `salio_en` |
| 10 | RF-22 / RN-06 | `POST /api/viajes/{id}/pin-inicio` · `/pin-fin` | Transportista | PIN dictado por el Cliente + lat/lng/precisión; tolerancia 150 m; 5 intentos fallidos abren incidente |
| 11 | RF-08 / RF-19 / RN-07 | `POST /api/viajes/{id}/cancelar` | Cliente / Transportista | Cliente: cargo del 20 % si ya salió; Transportista: sin cargo, solicitud vuelve a `publicada` (D-27) |
| 12 | RI-03 | `POST /api/transportista/cuenta-pago` | Transportista | vinculación de Mercado Pago (OAuth), requisito para ofertar |
| 12 | RI-03 | `POST /api/webhooks/pagos/mercadopago` | Mercado Pago | **sin JWT** (excepción a RNF-01), firma + idempotencia (D-28) |
| 13 | RF-12 / RN-06 | `POST /api/viajes/{id}/resena` | Cliente | viaje finalizado y dentro de 14 días (D-29) |
| 13 | RF-13 / RF-23 | `POST /api/incidentes` | Cliente/Transportista | con adjuntos opcionales; congela la liberación del pago |
| 13 | RF-02 / RF-03 | `GET /api/admin/incidentes` · `POST /api/admin/incidentes/{id}/resolver` | Administrador | reembolso o liberación total/parcial, veto, cierre sin acción |
| 13 | RF-04 | `POST /api/admin/vetos` | Administrador | temporal o definitivo |
| 13 | RF-09 | `GET /api/notificaciones` · `POST /api/notificaciones/{id}/leida` | autenticado | sólo in-app en esta etapa (D-22) |
| 14 | RF-14 / RF-24 | `GET /api/viajes?rol=cliente\|transportista` | autenticado | historial paginado |

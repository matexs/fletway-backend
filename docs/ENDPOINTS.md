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
  "estado_habilitacion": "pendiente | habilitado | rechazado | null",
  "disponible": "true | false | null"
}
```

- `registro_completo`: `false` mientras falte la fila del rol; la app manda al usuario a
  completar el registro.
- `estado_habilitacion`: sólo para Transportistas registrados; `null` en otro caso.
- `disponible`: interruptor "estoy tomando trabajos" del Transportista (D-21); `null` en otro caso.

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

### Documentación del Transportista (RF-16, RF-01, D-19, D-33)

La app sube cada archivo **directo a Supabase Storage**, bucket privado `documentos-transportista`,
con el path `transportista/{usuario_id}/{nombre}` (jpg, png o pdf de hasta 10 MB). Después lo
registra en el backend. El estado de habilitación lo calcula la base (D-33).

#### Respuesta `Documento`

```json
{
  "id": "uuid",
  "tipo_documento_codigo": "dni | registro | seguro | vtv",
  "estado": "pendiente | aprobado | rechazado",
  "motivo_rechazo": "texto | null",
  "cargado_en": "2026-10-03T01:09:43Z",
  "revisado_en": "2026-10-03T01:09:52Z | null",
  "url": "URL firmada, sólo para el Administrador (vence a los 10 minutos)"
}
```

#### `POST /api/transportista/documentos` (RF-16)
Request: `{"tipo_documento_codigo": "dni", "path": "transportista/{usuario_id}/dni-....pdf"}`.
`201` con `Documento` (siempre `pendiente`).

| Status | `code` | Cuándo |
|---|---|---|
| 400 | `datos_incompletos` | falta el tipo o el path |
| 400 | `tipo_documento_invalido` | el tipo no es dni, registro, seguro ni vtv |
| 400 | `path_invalido` | el path no está bajo `transportista/{tu usuario_id}/` |
| 400 | `archivo_no_encontrado` | no hay un objeto subido en ese path |
| 403 | `no_es_transportista` | la cuenta no es de Transportista registrado |

#### `GET /api/transportista/documentos` (RF-16, RF-01)
`200`:

```json
{
  "estado_habilitacion": "pendiente | habilitado | rechazado",
  "documentos": [
    { "tipo_documento_codigo": "dni", "descripcion": "Documento Nacional de Identidad", "ultimo": "Documento | null" }
  ]
}
```

Un elemento por tipo requerido, en orden fijo, con el último documento cargado. Error: `403
no_es_transportista`.

#### `GET /api/admin/transportistas` (RF-01, Administrador)
Sin parámetros: los Transportistas con algún documento `pendiente` (incluye renovaciones de
habilitados). Con `?estado=pendiente|habilitado|rechazado`: los de ese estado de habilitación.
`200` con una lista de:

```json
{
  "usuario_id": "uuid", "nombre_completo": "...", "email": "...", "telefono": "...",
  "estado_habilitacion": "pendiente",
  "documentos": [ "igual que en GET /api/transportista/documentos, con url en cada Documento" ]
}
```

Errores: `400 estado_invalido`, `403 requiere_administrador`.

#### `POST /api/admin/documentos/{id}/aprobar` y `/rechazar` (RF-01, Administrador)
`rechazar` lleva `{"motivo": "..."}` (1 a 500 caracteres). `200`:

```json
{ "documento": "Documento", "estado_habilitacion": "estado del Transportista después de revisar" }
```

Notifica al Transportista (`documentacion_revisada`, en segundo plano) cuando un rechazo lo deja
`rechazado` (con el motivo) o cuando queda `habilitado`.

| Status | `code` | Cuándo |
|---|---|---|
| 400 | `motivo_requerido` / `motivo_demasiado_largo` | rechazo sin motivo o con más de 500 caracteres |
| 403 | `requiere_administrador` | quien llama no es Administrador |
| 404 | `documento_no_encontrado` | el id no existe o está mal formado |
| 409 | `documento_ya_revisado` | el documento ya no está pendiente |

### Vehículos y costos (RF-18, RN-01, D-32)

Errores de validación de forma: `400 datos_invalidos` con `details` = `{ "<campo>": "<problema>" }`
(todos los campos con problema a la vez). Montos y medidas son números JSON con hasta 2 decimales
(D-11); un número mandado como texto es `400 json_invalido`.

#### `GET /api/tipos-vehiculo` (autenticado)
`200` con la lista de tipos, de menor a mayor capacidad. Las medidas son de referencia: la app las
propone al registrar un vehículo (D-32).

```json
[{ "id": "uuid", "nombre": "Furgón chico", "largo_estandar_m": 2.5, "ancho_estandar_m": 1.5,
   "alto_estandar_m": 1.2, "peso_maximo_estandar_kg": 700 }]
```

#### `POST /api/transportista/vehiculos` (Transportista)
Request (medidas útiles de la caja en metros, `peso_maximo_kg` = carga útil):

```json
{ "tipo_vehiculo_id": "uuid", "patente": "AB123CD", "marca": "Renault", "modelo": "Kangoo",
  "largo_util_m": 2.4, "ancho_util_m": 1.45, "alto_util_m": 1.15, "peso_maximo_kg": 650 }
```

La patente se normaliza (mayúsculas, sin espacios ni guiones) y tiene que ser `AAA999` o
`AA999AA`. Rangos: largo `(0, 20]`, ancho `(0, 3]`, alto `(0, 5]`, peso `(0, 40000]`. `marca` y
`modelo` son opcionales (hasta 60 caracteres). `201` con `Vehiculo`:

```json
{ "id": "uuid", "tipo_vehiculo_id": "uuid", "tipo_vehiculo_nombre": "Furgón chico",
  "patente": "AB123CD", "marca": "Renault", "modelo": "Kangoo",
  "largo_util_m": 2.4, "ancho_util_m": 1.45, "alto_util_m": 1.15, "peso_maximo_kg": 650,
  "activo": true, "tiene_costos": false, "creado_en": "2026-10-03T12:00:00Z" }
```

Errores: `400 datos_invalidos`, `400 tipo_vehiculo_invalido`, `403 no_es_transportista`,
`409 patente_duplicada`.

#### `GET /api/transportista/vehiculos` (Transportista)
`200` con la lista de `Vehiculo` propios (activos primero). Error: `403 no_es_transportista`.

#### `PUT /api/transportista/vehiculos/{id}/activo` (Transportista)
Request `{"activo": false}`. `200` con `Vehiculo`. Un vehículo inactivo no cuenta para el
matchmaking ni para ofertar (D-21). Errores: `400 datos_incompletos`, `404 vehiculo_no_encontrado`
(no existe o no es propio).

#### `PUT /api/transportista/vehiculos/{id}/costos` y `GET` (Transportista dueño)
Segundo paso del alta: los costos operativos que usa el precio de cada oferta (RN-01,
`ALGORITMO_COTIZACION.md` §4.2). El `PUT` crea o reemplaza. Sólo los leen el dueño y el
Administrador. Request y respuesta (la respuesta suma `actualizado_en`):

```json
{ "combustible_precio_l": 1350.5, "rendimiento_km_l": 9.5, "cantidad_neumaticos": 4,
  "costo_neumatico": 185000, "vida_neumatico_km": 50000, "costo_mantenimiento_km": 45.75,
  "valor_compra": 28000000, "valor_residual": 9000000, "vida_util_km": 400000,
  "seguro_mensual": 95000, "patente_mensual": 38000 }
```

Reglas: montos `>= 0` con hasta 2 decimales; `rendimiento_km_l > 0`; `cantidad_neumaticos` de 1
a 30; `vida_neumatico_km` y `vida_util_km` enteros `> 0`; `valor_residual <= valor_compra`.
Errores: `400 datos_invalidos`, `404 vehiculo_no_encontrado`, `404 costos_no_cargados` (sólo el
`GET`).

### Zonas y disponibilidad (RN-04, D-21)

#### `GET /api/zonas` (autenticado)
`200` con `[{ "id": "uuid", "nombre": "San Isidro", "provincia": "Buenos Aires" }]`, ordenado por
provincia y nombre.

#### `GET /api/transportista/zonas` y `PUT /api/transportista/zonas` (Transportista)
`{"zona_ids": ["uuid", ...]}` en los dos sentidos. El `PUT` reemplaza el conjunto completo (puede
ser vacío, hasta 50, sin repetidos). Errores: `400 datos_incompletos`, `400 datos_invalidos`,
`400 zona_invalida`, `403 no_es_transportista`.

#### `PUT /api/transportista/disponibilidad` (Transportista)
Request `{"disponible": true}`. `200` con `Me`. Errores: `400 datos_incompletos`,
`403 no_es_transportista`.

---

## Endpoints planificados (no implementados)

> Se completan a medida que se implementan, con la skill `scaffold-endpoint`, siguiendo el orden
> de `docs/PLAN_CONSTRUCCION.md` (columna "Mód."). Todas las rutas llevan el prefijo `/api`.

| Mód. | RF/RN | Método + path | Rol | Notas |
|------|-------|---------------|-----|-------|
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

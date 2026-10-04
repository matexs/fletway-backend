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

### Vehículos (RF-18, D-32, D-34)

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
  "activo": true, "creado_en": "2026-10-03T12:00:00Z" }
```

Errores: `400 datos_invalidos`, `400 tipo_vehiculo_invalido`, `403 no_es_transportista`,
`409 patente_duplicada`.

#### `GET /api/transportista/vehiculos` (Transportista)
`200` con la lista de `Vehiculo` propios (activos primero). Error: `403 no_es_transportista`.

#### `PUT /api/transportista/vehiculos/{id}/activo` (Transportista)
Request `{"activo": false}`. `200` con `Vehiculo`. Un vehículo inactivo no cuenta para el
matchmaking ni para ofertar (D-21). Errores: `400 datos_incompletos`, `404 vehiculo_no_encontrado`
(no existe o no es propio).

Los costos del vehículo que entran en el precio no los carga el Transportista: son de referencia
por tipo de vehículo (`config_costo_vehiculo`, D-34).

### Zonas y disponibilidad (RN-04, D-21)

#### `GET /api/zonas` (autenticado)
`200` con `[{ "id": "uuid", "nombre": "San Isidro", "provincia": "Buenos Aires" }]`, ordenado por
provincia y nombre.

#### `GET /api/direcciones/sugerencias?zona_id=&q=` (autenticado)
Autocompletado de direcciones dentro de la zona (D-35): hasta 5 sugerencias a menos de 20 km del
centro de la zona. `q` entre 3 y 100 caracteres. `200`:

```json
[{ "direccion": "Avenida Centenario 1200", "detalle": "La Calabria, 1642 San Isidro, Argentina",
   "lat": -34.4659, "lng": -58.5216 }]
```

Las coordenadas son orientativas: al publicar, el backend vuelve a ubicar la dirección. Con el
proveedor `aproximado` la única sugerencia es el texto escrito. Errores: `400 datos_invalidos`,
`400 zona_invalida`.

#### `GET /api/transportista/zonas` y `PUT /api/transportista/zonas` (Transportista)
`{"zona_ids": ["uuid", ...]}` en los dos sentidos. El `PUT` reemplaza el conjunto completo (puede
ser vacío, hasta 50, sin repetidos). Errores: `400 datos_incompletos`, `400 datos_invalidos`,
`400 zona_invalida`, `403 no_es_transportista`.

#### `PUT /api/transportista/disponibilidad` (Transportista)
Request `{"disponible": true}`. `200` con `Me`. Errores: `400 datos_incompletos`,
`403 no_es_transportista`.

### Catálogo de objetos (RN-08)

#### `GET /api/catalogo/objetos` (autenticado)
`200` con el catálogo completo (28 objetos), ordenado por nombre. El Cliente elige de acá al publicar
una solicitud (módulo 6) y los valores se copian a `solicitud_objeto`.

```json
[{ "id": "uuid", "nombre": "Heladera", "peso_estimado_kg": 70, "largo_m": 0.7, "ancho_m": 0.7,
   "alto_m": 1.8, "rotacion_horizontal": true, "rotacion_vertical": false, "apilable": false }]
```

`alto_m` es el eje vertical; `rotacion_vertical = false` indica que no se puede acostar y
`apilable = false` que no se le pone carga encima (RN-02).

### Solicitudes (RF-06, D-20)

Ningún endpoint de solicitud calcula ni devuelve un monto (RN-01): el único precio es el de cada
oferta. Una solicitud no se edita: se cancela y se publica otra (la base lo impone, `0014`).
"Hoy" y "vencida" se cuentan en hora argentina.

#### `POST /api/solicitudes` (Cliente)

```json
{
  "origen":  { "zona_id": "uuid", "direccion": "Av. Centenario 1200", "pisos": 2,
               "ascensor_utilizable": false, "distancia_vehiculo_m": 15.5 },
  "destino": { "zona_id": "uuid", "direccion": "Av. Corrientes 3500", "pisos": 5,
               "ascensor_utilizable": true, "distancia_vehiculo_m": 0 },
  "fecha_servicio_deseada": "2026-10-12",
  "franja_horaria_inicio": "09:00",
  "franja_horaria_fin": "13:00",
  "cantidad_ayudantes_solicitados": 2,
  "objetos": [
    { "objeto_id": "uuid", "cantidad": 1 },
    { "nombre_personalizado": "Piano vertical", "cantidad": 1, "peso_unitario_kg": 250,
      "largo_m": 1.5, "ancho_m": 0.6, "alto_m": 1.3,
      "rotacion_horizontal": true, "rotacion_vertical": false, "apilable": false }
  ]
}
```

- Un objeto del catálogo lleva **sólo** `objeto_id` y `cantidad`: peso, medidas y restricciones los
  copia la base desde `objeto` (RN-08). Uno manual lleva nombre, peso y las tres medidas; las
  restricciones son opcionales (por defecto `true`).
- La franja es opcional (sin franja = "lo antes posible"); si se carga, van las dos horas.
- Las coordenadas las calcula el backend con el `Geocodificador` (D-20); no se exponen.
- Reglas: fecha `>= hoy`; dirección de 5 a 200 caracteres; pisos de 0 a 60; distancia a pie de 0 a
  9999,9 m con 1 decimal; ayudantes de 0 a 3 (informativo, D-20); de 1 a 50 objetos, cantidad de 1 a
  100; objeto manual: nombre de 2 a 80 caracteres, peso hasta 2000 kg, medidas hasta 10 m, con hasta
  2 decimales.

`201` con `Solicitud`:

```json
{
  "id": "uuid", "estado": "publicada", "fecha_servicio_deseada": "2026-10-12",
  "franja_horaria_inicio": "09:00", "franja_horaria_fin": "13:00",
  "origen":  { "zona_id": "uuid", "zona_nombre": "San Isidro", "direccion": "...", "pisos": 2,
               "ascensor_utilizable": false, "distancia_vehiculo_m": 15.5 },
  "destino": { "...": "igual que origen" },
  "cantidad_ayudantes_solicitados": 2,
  "objetos": [
    { "id": "uuid", "objeto_id": "uuid | null", "nombre": "Heladera", "cantidad": 1,
      "peso_unitario_kg": 70, "largo_m": 0.7, "ancho_m": 0.7, "alto_m": 1.8,
      "rotacion_horizontal": true, "rotacion_vertical": false, "apilable": false }
  ],
  "creado_en": "2026-10-10T15:00:00Z"
}
```

`estado`: `publicada`, `vencida` (publicada con la fecha ya pasada, calculado al leer), `asignada`,
`cancelada` o `expirada`.

| Status | `code` | Cuándo |
|---|---|---|
| 400 | `datos_invalidos` | forma inválida (`details` por campo, por ejemplo `objetos[1].largo_m`) |
| 400 | `json_invalido` | campos desconocidos (por ejemplo, un monto) |
| 400 | `fecha_pasada` | la fecha es anterior a hoy |
| 400 | `zona_invalida` / `objeto_invalido` | zona u objeto del catálogo inexistente |
| 400 | `direccion_no_ubicable` | el `Geocodificador` no ubica la dirección en la zona |
| 403 | `no_es_cliente` | la cuenta no es de Cliente registrado |

#### `GET /api/solicitudes` (Cliente)
`200` con las solicitudes propias, las más nuevas primero:
`[{ "id", "estado", "fecha_servicio_deseada", "franja_horaria_inicio", "franja_horaria_fin",
"origen_zona_nombre", "destino_zona_nombre", "cantidad_objetos", "creado_en" }]`
(`cantidad_objetos` suma las cantidades). Error: `403 no_es_cliente`.

#### `GET /api/solicitudes/{id}`
`200` con `Solicitud` si quien llama la puede ver (RLS: el Cliente dueño, el Administrador y los
Transportistas de las zonas de origen o destino). Error: `404 solicitud_no_encontrada`.

#### `GET /api/solicitudes/{id}/ruta`
Recorrido para el mapa (D-35), visible para quien ve la solicitud: su Cliente y los Transportistas
que la pueden ofertar. Distancia y tiempo de manejo de ida y hasta 300 puntos del trayecto por
calles (con `aproximado`, la línea recta). `200`:

```json
{ "origen": { "lat": -34.4659, "lng": -58.5216 }, "destino": { "lat": -34.6042, "lng": -58.4144 },
  "distancia_km": 25.1, "duracion_min": 26, "trazado": [{ "lat": -34.4658, "lng": -58.5215 }] }
```

Errores: `404 solicitud_no_encontrada`, `400 ruta_no_disponible`.

#### `POST /api/solicitudes/{id}/cancelar` (Cliente dueño)
Sin costo: todavía no hay compromiso. Las ofertas pendientes pasan a `no_seleccionada`. `200` con
`Solicitud`. Errores: `404 solicitud_no_encontrada`, `409 solicitud_no_cancelable` (no está
publicada).

#### `POST /api/solicitudes/{id}/republicar` (Cliente dueño)
Request `{"fecha_servicio_deseada": "2026-10-20", "franja_horaria_inicio": null,
"franja_horaria_fin": null}`. Sólo sobre una solicitud vencida: crea una nueva copiando datos y
objetos; la vencida queda como registro. `201` con la `Solicitud` nueva. Errores:
`400 fecha_pasada`, `404 solicitud_no_encontrada`, `409 solicitud_no_vencida`.

### Matchmaking (RN-04, RN-05, D-21)

#### `GET /api/transportista/solicitudes` (Transportista)
Solicitudes que el Transportista puede ofertar, las de fecha más cercana primero. Una solicitud es
compatible (D-21) si está publicada y no vencida; el Transportista está habilitado, disponible y sin
veto vigente; la zona de origen o de destino está entre las suyas; y algún vehículo activo pasa la
cota rápida de peso y volumen (la carga entra en 20 viajes como máximo). Si no cumple las tres
condiciones del Transportista, la lista vuelve vacía. `200`:

```json
[{ "id": "uuid", "fecha_servicio_deseada": "2026-10-12", "franja_horaria_inicio": "09:00",
   "franja_horaria_fin": "13:00", "origen_zona_nombre": "San Isidro",
   "destino_zona_nombre": "Ciudad Autónoma de Buenos Aires", "cantidad_objetos": 4,
   "peso_total_kg": 400, "volumen_total_m3": 3.98, "cantidad_ayudantes_solicitados": 2,
   "creado_en": "2026-10-03T15:00:00Z" }]
```

El detalle (direcciones y objetos) sale de `GET /api/solicitudes/{id}`. Error: `403
no_es_transportista`.

**Aviso al publicar (RN-05, D-22):** `POST /api/solicitudes` y `.../republicar` encolan un job que
crea la notificación in-app `solicitud_compatible` ("Hay una nueva solicitud de flete en tu zona.
Revisala y postulate si te interesa.") para cada Transportista compatible en ese momento. Es
idempotente: una notificación por Transportista y solicitud.

### Ofertas (RF-17, RN-01, RN-02, D-23, D-24)

El Transportista elige vehículo y ayudantes; el sistema calcula la cantidad de viajes con el
vehículo real (`ALGORITMO_VIAJES_EMPAQUETADO.md`) y el precio (`ALGORITMO_COTIZACION.md`). Una oferta
no se edita: se retira y se crea otra. El desglose (`desglose`) sólo lo ven el Transportista dueño y
el Administrador; la base lo guarda en `oferta_costo` (`0016`).

#### `POST /api/solicitudes/{id}/ofertas/cotizar` (Transportista)
Mismo request y mismas validaciones que ofertar, pero **no guarda nada**: devuelve el precio y los
viajes que tendría la oferta, para decidir antes de ofertar. `200`:

```json
{ "cantidad_viajes": 1, "cantidad_ayudantes": 1, "precio_calculado": 98456.12,
  "desglose": { "distancia_km": 18.4, "duracion_ruta_h": 0.61, "duracion_operacion_h": 0.52,
    "costo_laboral": 42210.33, "costo_vehiculo": 9120.5, "costos_adicionales": 0,
    "costo_operativo": 51330.83, "margen_pct": 0, "precio_neto": 60389.21,
    "porcentaje_comision": 15, "iva_pct": 21 } }
```

#### `POST /api/solicitudes/{id}/ofertas` (Transportista habilitado)
Request: `{ "vehiculo_id": "uuid", "cantidad_ayudantes": 1 }` (0 a 3, D-23). La solicitud tiene que
estar entre las compatibles del Transportista (`GET /api/transportista/solicitudes`); el vehículo,
activo. `201` con `Oferta`:

```json
{ "id": "uuid", "estado": "pendiente", "solicitud_id": "uuid", "solicitud_estado": "publicada",
  "fecha_servicio_deseada": "2026-10-12", "origen_zona_nombre": "San Isidro",
  "destino_zona_nombre": "Ciudad Autónoma de Buenos Aires", "vehiculo_id": "uuid",
  "vehiculo_patente": "AB123CD", "vehiculo_tipo": "Furgón chico", "cantidad_viajes": 1,
  "cantidad_ayudantes": 1, "precio_calculado": 98456.12, "desglose": { "...": "igual que cotizar" },
  "creado_en": "2026-10-03T15:00:00Z" }
```

Errores (también de cotizar, salvo el último):

| Status | `code` | Cuándo |
|---|---|---|
| 400 | `datos_invalidos` | `vehiculo_id` mal formado o ayudantes fuera de 0..3 (`details` por campo) |
| 400 | `carga_no_factible` | la carga no entra en el vehículo; `details.motivos` explica por objeto ("Heladera: no entra en la posición en que tiene que viajar") o "la carga necesita más de 20 viajes con este vehículo" |
| 400 | `calculo_demorado` | el cálculo de viajes superó los 5 s (D-24) |
| 400 | `ruta_no_disponible` | no se pudo calcular el recorrido (nunca se estima "a ojo", D-24) |
| 403 | `no_es_transportista` · `transportista_no_habilitado` | |
| 404 | `solicitud_no_disponible` | no existe, ya no está publicada o no es compatible con el Transportista |
| 404 | `vehiculo_no_encontrado` | no existe o es de otro Transportista |
| 409 | `transportista_no_disponible` | tiene la disponibilidad apagada |
| 409 | `vehiculo_inactivo` | |
| 409 | `oferta_duplicada` | ya tiene una oferta vigente con ese vehículo para la solicitud |

#### `POST /api/ofertas/{id}/retirar` (Transportista dueño)
Pasa a `retirada` una oferta `pendiente`. `200` con `Oferta`. Después puede ofertar de nuevo con el
mismo vehículo. Errores: `404 oferta_no_encontrada`, `409 oferta_no_retirable`.

#### `GET /api/transportista/ofertas` (Transportista)
Sus ofertas, las más nuevas primero, con el desglose. `solicitud_estado` trae `vencida` calculado al
leer (D-20). `200` con `[Oferta]`. Error: `403 no_es_transportista`.

#### `GET /api/solicitudes/{id}/ofertas` (Cliente dueño)
Ofertas **pendientes** de la solicitud ordenadas por score (RN-05, `ALGORITMO_SCORE.md`): precio,
calificación (3,5 neutral sin reseñas) y tasa de cumplimiento, sin cercanía (D-25). Sin parámetros
trae las 3 primeras; `?ver_mas=true` sigue la misma lista desde `cursor` (por defecto 3) con `limit`
(por defecto 10, hasta 50). Nunca expone el desglose ni la patente. `200`:

```json
{ "cantidad_ayudantes_solicitados": 1, "total": 5, "siguiente_cursor": 3,
  "ofertas": [{ "id": "uuid", "transportista_id": "uuid", "transportista_nombre": "Tomás Pérez",
    "calificacion_promedio": 4.5, "cantidad_resenas": 12, "tasa_cumplimiento": 100,
    "vehiculo_tipo": "Furgón grande", "cantidad_viajes": 1, "cantidad_ayudantes": 1,
    "precio_calculado": 106900.15, "creado_en": "2026-10-03T15:00:00Z" }] }
```

`calificacion_promedio` es `null` sin reseñas; `siguiente_cursor` es `null` si no hay más. Errores:
`400 datos_invalidos` (`cursor` o `limit`), `404 solicitud_no_encontrada` (no existe o no es suya).

#### `POST /api/ofertas/{id}/aceptar` (Cliente dueño de la solicitud)
Llama a `fn_aceptar_oferta` (`0018`, D-23): en una transacción crea el viaje con sus snapshots, copia
el desglose a `viaje_costo`, genera los dos PIN en `viaje_pin`, acepta la oferta, pasa las demás a
`no_seleccionada` y la solicitud a `asignada`. `201` con el viaje confirmado (recién acá se ve la
patente):

```json
{ "id": "uuid", "estado": "confirmado", "solicitud_id": "uuid", "transportista_id": "uuid",
  "transportista_nombre": "Tomás Pérez", "vehiculo_patente": "AC456EF",
  "vehiculo_marca_modelo": "Iveco Daily", "monto_total": 106900.15, "cantidad_viajes": 1,
  "cantidad_ayudantes": 1, "fecha_servicio_deseada": "2026-10-08",
  "origen_direccion": "Av. Centenario 1200", "destino_direccion": "Corrientes 3500",
  "creado_en": "2026-10-03T15:00:00Z" }
```

Errores: `404 oferta_no_encontrada` (no existe o la solicitud no es suya), `409 oferta_no_disponible`
(retirada, ya aceptada o no seleccionada), `409 solicitud_no_asignable` (ya no publicada o vencida),
`409 transportista_inhabilitado` (el Transportista dejó de estar habilitado o está vetado).

### Perfil del Transportista (RF-11)

#### `GET /api/transportistas/{id}` (autenticado)
Perfil público de un Transportista habilitado. Sin email, teléfono, patentes ni datos financieros.
`200`:

```json
{ "id": "uuid", "nombre": "Tomás Pérez", "forma_trabajo": "Mudanzas chicas, con cuidado.",
  "calificacion_promedio": 4.5, "cantidad_resenas": 12, "tasa_cumplimiento": 100,
  "zonas": ["San Isidro", "Tigre"], "tipos_vehiculo": ["Furgón grande"],
  "resenas": [{ "calificacion": 5, "mensaje": "Impecable.", "cliente_nombre": "Clara",
    "creado_en": "2026-09-30T18:00:00Z" }],
  "en_fletway_desde": "2026-09-01T12:00:00Z" }
```

Trae las 20 reseñas más recientes y sólo los tipos de vehículos activos. Error: `404
transportista_no_encontrado` (no existe o no está habilitado).

---

## Endpoints planificados (no implementados)

> Se completan a medida que se implementan, con la skill `scaffold-endpoint`, siguiendo el orden
> de `docs/PLAN_CONSTRUCCION.md` (columna "Mód."). Todas las rutas llevan el prefijo `/api`.

| Mód. | RF/RN | Método + path | Rol | Notas |
|------|-------|---------------|-----|-------|
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

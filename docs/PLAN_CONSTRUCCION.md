# Fletway — Plan de construcción

**Fecha:** 2026-10-01 · **Alcance:** `fletway-backend` (Go) y `fletway-mobile` (Flutter).

Plan para construir la aplicación **módulo a módulo**. Cada módulo dice qué requisitos cubre, qué decisiones aplica, qué cambios de base necesita, qué endpoints y pantallas construir, y cuándo está terminado. Las decisiones están en `docs/DECISIONES_TECNICAS.md` (D-01 a D-31) y salen del relevamiento `PENDIENTES_ANTES_DE_CONSTRUIR` y de las soluciones acordadas por el equipo el 2026-10-01.

**Cómo usarlo con Claude Code:**
- Construir los módulos **en orden**. Cada módulo tiene su rama (`feat/<RF|RN>-<nn>-<slug>`) y uno o más PRs.
- Antes de cada módulo, leer este plan, las decisiones que cita y el `CLAUDE.md` de cada repo.
- Endpoints nuevos con la skill `scaffold-endpoint`; pantallas con `new-screen`; cambios de base con `supabase-migration` (siempre con confirmación humana antes de aplicar).
- Un RF/RN se marca `OK` en `docs/TRAZABILIDAD.md` sólo con la definición de terminado de D-31.

---

## 1. Antes de empezar

### 1.1 Puntos resueltos (2026-10-01)

Surgieron al incorporar las soluciones y quedaron resueltos con el equipo:

| # | Punto | Resolución |
|---|---|---|
| A-1 | `tipo_vehiculo` | Se reemplazan las 4 filas por **6 tipos con largo, ancho y alto estándar** (§2.3). Son medidas de referencia para proponer al registrar un vehículo; el cálculo de viajes y el matchmaking usan las medidas del **vehículo real** (`vehiculo.largo_util_m`, etc.). Módulo 4. |
| A-2 | Color de la app | Primario **naranja tostado `#C36224`**; secundarios discretos (grises); fondo blanco. Detalle en el `CLAUDE.md` §5 de mobile. Módulo 1. |
| A-3 | Hosting | **Render**, a priori. El despliegue a producción no es parte de esta etapa: el objetivo es que todo funcione **en local** (D-16). |
| A-4 | Desglose de la oferta | Tablas **`oferta_costo`** y **`viaje_costo`** (1 a 1) con RLS sólo para el Transportista dueño y el Administrador, y función **`fn_aceptar_oferta`** (`SECURITY DEFINER`) para la aceptación (D-23). Módulos 8 y 9. |
| A-5 | PIN | Tabla **`viaje_pin`** (sólo Cliente del viaje y Administrador) y función **`fn_validar_pin`** (`SECURITY DEFINER`); se eliminan `viaje.pin_inicio` y `viaje.pin_fin` (D-26). Módulos 9 y 10. |
| A-6 | Ruteo sin API | Interfaz `Ruteador` con proveedores `google` (producción), `aproximado` (sólo local) y `fijo` (sólo tests), elegidos con `RUTEO_PROVEEDOR`; en producción el backend no arranca si el proveedor no es `google` (D-24). Módulo 8. |
| A-7 | ERS | **Hecho:** `docs/ERS_Fletway.docx` sin cercanía en RN-05 y en el glosario, más la sección "5. Anexo — Registro de cambios". El `.docx` reemplaza al PDF en el repo (D-25). |

### 1.2 Tareas de setup (una sola vez)

1. **Supabase local (D-16):** requiere Docker y Supabase CLI. La configuración está en `supabase/config.toml` (con firma ES256 local, D-04) y el esquema base en `migrations/baseline/` (dump de `dbFletway`, sólo lectura, que ya incluye `0001`–`0007`). **`make db-local`** genera la clave local si falta, levanta el stack, resetea la base y aplica el esquema base más las migraciones posteriores. `make test-integracion` corre los tests contra esa base (`TEST_DATABASE_URL`).
2. **Credenciales (D-16):** `.env` de los dos repos para el entorno local, desde el gestor de contraseñas del equipo. Las de producción (Render) se definen cuando se haga el despliegue, fuera de esta etapa.
3. **Branch protection (D-31):** en GitHub, para `main` de los dos repos, "Require a pull request before merging" con 1 aprobación.
4. **Primer Administrador (D-18):** crear el usuario en Supabase Auth y sus filas `usuario` (rol `administrador`) y `administrador` por SQL, con confirmación humana, una única vez. Se hace al terminar el módulo 2 (necesita el trigger de alta). **Hecho el 2026-10-02:** como el trigger rechaza `rol=administrador` y el dashboard no carga metadata, el alta se hace con un signUp con `rol=cliente` y después el SQL cambia `usuario.rol` y crea la fila `administrador`.

---

## 2. Módulo 0 — Datos semilla

Una migración de seed versionada (skill `supabase-migration`), aplicada primero en local y después en `dbFletway` con confirmación.

### 2.1 Zonas piloto (D-20, D-21)

Granularidad partido/localidad. Área piloto: CABA y Zona Norte del GBA.

```sql
INSERT INTO zona (nombre, provincia) VALUES
  ('Ciudad Autónoma de Buenos Aires', 'CABA'),
  ('Campana', 'Buenos Aires'),
  ('Zárate', 'Buenos Aires'),
  ('Escobar', 'Buenos Aires'),
  ('Pilar', 'Buenos Aires'),
  ('Tigre', 'Buenos Aires'),
  ('San Fernando', 'Buenos Aires'),
  ('San Isidro', 'Buenos Aires'),
  ('Vicente López', 'Buenos Aires'),
  ('San Martín', 'Buenos Aires'),
  ('Tres de Febrero', 'Buenos Aires'),
  ('San Miguel', 'Buenos Aires'),
  ('Malvinas Argentinas', 'Buenos Aires'),
  ('José C. Paz', 'Buenos Aires');
```

Agregar zonas después es un ABM del Administrador.

### 2.2 Catálogo de objetos (RN-08)

Completa las 5 filas existentes y suma 23. `alto_m` es el eje vertical; `rotacion_vertical = false` en lo que no se puede acostar; `apilable = false` en lo frágil o pesado. `volumen_estimado_m3` no se ajusta: está marcado como redundante.

```sql
UPDATE objeto SET largo_m=1.40, ancho_m=0.80, alto_m=0.75, rotacion_horizontal=true,  rotacion_vertical=true,  apilable=false WHERE nombre='Mesa de comedor';
UPDATE objeto SET largo_m=0.70, ancho_m=0.70, alto_m=1.80, rotacion_horizontal=true,  rotacion_vertical=false, apilable=false WHERE nombre='Heladera';
UPDATE objeto SET largo_m=1.40, ancho_m=1.90, alto_m=0.55, rotacion_horizontal=true,  rotacion_vertical=true,  apilable=false WHERE nombre='Cama matrimonial (colchón + base)';
UPDATE objeto SET largo_m=1.60, ancho_m=0.90, alto_m=0.85, rotacion_horizontal=true,  rotacion_vertical=true,  apilable=false WHERE nombre='Sofá 2 cuerpos';
UPDATE objeto SET largo_m=0.50, ancho_m=0.40, alto_m=0.40, rotacion_horizontal=true,  rotacion_vertical=true,  apilable=true  WHERE nombre='Caja mudanza estándar';

INSERT INTO objeto (nombre, peso_estimado_kg, volumen_estimado_m3, largo_m, ancho_m, alto_m, rotacion_horizontal, rotacion_vertical, apilable) VALUES
  ('Caja mudanza chica', 10, 0.036, 0.40, 0.30, 0.30, true,  true,  true),
  ('Caja mudanza grande', 25, 0.150, 0.60, 0.50, 0.50, true,  true,  true),
  ('Cama 1 plaza (colchón + base)', 30, 0.855, 0.90, 1.90, 0.50, true,  true,  false),
  ('Cama 2 plazas (colchón + base)', 45, 1.463, 1.40, 1.90, 0.55, true,  true,  false),
  ('Ropero / placard 2 cuerpos', 60, 1.440, 1.20, 0.60, 2.00, true,  false, false),
  ('Cómoda / cajonera', 35, 0.450, 1.00, 0.50, 0.90, true,  false, true),
  ('Sofá 3 cuerpos', 55, 1.530, 2.00, 0.90, 0.85, true,  true,  false),
  ('Sillón individual', 20, 0.544, 0.80, 0.80, 0.85, true,  true,  false),
  ('Mesa ratona', 12, 0.225, 1.00, 0.50, 0.45, true,  true,  true),
  ('Silla (comedor)', 5, 0.182, 0.45, 0.45, 0.90, true,  true,  true),
  ('Mesa de luz', 8, 0.099, 0.45, 0.40, 0.55, true,  true,  true),
  ('Escritorio', 25, 0.540, 1.20, 0.60, 0.75, true,  true,  false),
  ('Biblioteca / estantería', 35, 0.486, 0.90, 0.30, 1.80, true,  false, false),
  ('TV (hasta 55", embalada)', 15, 0.156, 1.30, 0.15, 0.80, false, false, false),
  ('Microondas', 12, 0.060, 0.50, 0.40, 0.30, true,  false, true),
  ('Horno eléctrico / anafe', 15, 0.120, 0.60, 0.50, 0.40, true,  false, true),
  ('Lavarropas', 65, 0.306, 0.60, 0.60, 0.85, true,  false, false),
  ('Lavavajillas', 40, 0.306, 0.60, 0.60, 0.85, true,  false, false),
  ('Aire acondicionado split (embalado)', 25, 0.081, 0.90, 0.30, 0.30, true,  false, true),
  ('Estufa / calefactor portátil', 10, 0.112, 0.40, 0.40, 0.70, true,  false, true),
  ('Bicicleta', 15, 1.020, 1.70, 0.60, 1.00, true,  true,  false),
  ('Espejo / cuadro grande', 8, 0.048, 1.20, 0.05, 0.80, false, false, false),
  ('Bulto de ropa / valija', 10, 0.072, 0.60, 0.40, 0.30, true,  true,  true);
```

`volumen_estimado_m3` es NOT NULL en la tabla, así que en los `INSERT` se carga el producto de las tres medidas. La propuesta original no incluía esa columna; sin ella el `INSERT` falla.

Después de esta seed: migración con `SET NOT NULL` en `objeto.largo_m/ancho_m/alto_m`.

### 2.3 Tipos de vehículo (A-1, módulo 4)

Migración del módulo 4: agrega `largo_estandar_m`, `ancho_estandar_m` y `alto_estandar_m` a `tipo_vehiculo`, marca `volumen_estandar_m3` como deprecada y reemplaza las 4 filas actuales (no hay vehículos que las referencien). Las medidas dan el volumen propuesto para cada tipo.

```sql
DELETE FROM tipo_vehiculo;  -- seguro: vehiculo está vacía
INSERT INTO tipo_vehiculo (nombre, largo_estandar_m, ancho_estandar_m, alto_estandar_m, volumen_estandar_m3, peso_maximo_estandar_kg) VALUES
  ('Utilitario',     1.60, 1.20, 1.05,  2.02,  500),
  ('Furgón chico',   2.50, 1.50, 1.20,  4.50,  700),
  ('Furgón grande',  3.00, 1.70, 1.70,  8.67, 1200),
  ('Camión chico',   4.00, 2.00, 1.90, 15.20, 2500),
  ('Camión mediano', 5.00, 2.20, 2.20, 24.20, 3500),
  ('Camión grande',  7.00, 2.40, 2.40, 40.32, 7000);
```

Uso: al registrar un vehículo, la app propone las medidas del tipo elegido y el Transportista las corrige con las reales. El algoritmo de viajes siempre usa las del vehículo real.

### 2.4 Configuración

Las filas vigentes de `config_costo_laboral`, `config_operacion`, `config_impuesto` y `config_comision` (15 %) son los valores de trabajo (D-24). `config_margen` se crea en el módulo 8.

---

## 3. Textos de las notificaciones (D-22)

`{}` = dato a interpolar. Sólo in-app en esta etapa.

| `codigo` | Texto |
|---|---|
| `solicitud_compatible` | "Hay una nueva solicitud de flete en tu zona. Revisala y postulate si te interesa." |
| `postulacion_recibida` | "Recibiste una nueva oferta para tu solicitud de flete." |
| `viaje_confirmado` | "¡Listo! Tu viaje quedó confirmado con {nombre_contraparte}." |
| `cambio_estado_viaje` | "Tu viaje cambió de estado: ahora está {estado_viaje}." |
| `viaje_cancelado` | "El viaje fue cancelado por {el Cliente / el Transportista}." Si hubo cargo: " Se aplicó un cargo de ${monto}." |
| `pago_procesado` | "Se procesó un pago de ${monto} por tu viaje." |
| `documentacion_revisada` | "Tu documentación fue {aprobada / rechazada}." Si fue rechazada: " Motivo: {motivo}." |
| `incidente_resuelto` | "Tu incidente fue resuelto: {resumen_resolucion}." |
| `veto_aplicado` | "Tu cuenta fue {vetada temporalmente hasta {fecha} / vetada de forma definitiva}. Motivo: {motivo}." |

---

## 4. Módulos

Cada módulo: requisitos, decisiones, base, backend, app y criterio de terminado. Los endpoints van todos bajo `/api` (D-11) y se documentan en `docs/ENDPOINTS.md`.

### Módulo 1 — Plataforma

- **Requisitos:** RNF-01, RNF-02.
- **Decisiones:** D-03, D-04, D-05, D-06, D-11, D-12, D-16.
- **Backend:**
  - `go mod edit -module github.com/matexs/fletway-backend` y actualizar imports (D-05).
  - Dependencias: `pgx/v5`, `golang-jwt/jwt/v5`, `MicahParks/keyfunc/v3`, `testify`.
  - `database.Open`, `Ping` y `WithinTx` reales (`SET LOCAL role authenticated` + `request.jwt.claims`).
  - `auth.Verify` contra el JWKS ES256, sin HS256; validar `exp`, `aud`, `iss`.
  - Tests de integración contra Supabase local.
- **App:**
  - Design system: `lib/shared/design_system/` con los tokens (`CLAUDE.md` §5) y `theme.dart` desde la semilla (A-2).
  - Componentes base `Fletway*`: botón, campo de texto, card y vistas de carga, error y vacío.
- **Terminado:** un request con JWT válido de Supabase local pasa el middleware y una query con `WithinTx` respeta RLS (test que lo demuestra); uno sin JWT o con JWT inválido recibe 401.

### Módulo 2 — Identidad

- **Requisitos:** RF-05, RF-16 (alta de cuenta), RNF-01.
- **Decisiones:** D-17, D-18.
- **Base:** trigger `AFTER INSERT ON auth.users` que crea `usuario` sólo con rol `cliente` o `transportista` (rechaza cualquier otro valor).
- **Backend:** `POST /api/auth/registro/cliente`, `POST /api/auth/registro/transportista` (crea la fila de rol; Transportista en `pendiente`), `GET /api/me`.
- **App:** login, registro de Cliente, registro de Transportista (datos personales; documentos en el módulo 3); `auth_controller` usa `/me` para rol y habilitación; guards del router con ese rol.
- **Setup:** bootstrap del primer Administrador (§1.2).
- **Terminado:** un usuario que pone `rol=administrador` en la metadata no obtiene ese rol (test del trigger); el rol de la app sale de `/me`.

### Módulo 3 — Habilitación de Transportistas

- **Requisitos:** RF-01, RF-16 (documentación).
- **Decisiones:** D-17 (Administrador por Postman, sin web admin), D-18, D-19, D-22.
- **Base:** bucket privado `documentos-transportista` y sus policies de Storage.
- **Backend:**
  - Transportista: `POST /api/transportista/documentos` (registra el path ya subido a Storage), `GET /api/transportista/documentos`.
  - Administrador: `GET /api/admin/transportistas?estado=pendiente`, `POST /api/admin/documentos/{id}/aprobar`, `POST /api/admin/documentos/{id}/rechazar` (con motivo); al cerrar la revisión, cambia `estado_habilitacion_codigo`.
  - Interfaz `Notificador` (implementación in-app): notificación `documentacion_revisada`.
- **App:** carga de documentos (`image_picker` + `file_picker`), pantalla de estado de habilitación, reintento tras rechazo.
- **Terminado:** un Transportista rechazado puede volver a cargar sin límite; sólo un Administrador aprueba o rechaza (test de RLS).

### Módulo 4 — Vehículos, costos y zonas

- **Requisitos:** RF-18, RN-04.
- **Decisiones:** D-21, costos en un segundo paso del alta del vehículo, A-1.
- **Base:** migración de `tipo_vehiculo` (§2.3).
- **Backend:** `POST/GET /api/transportista/vehiculos` (con `vehiculo_costo` en un segundo paso: `PUT /api/transportista/vehiculos/{id}/costos`), activar o desactivar vehículo; `GET /api/zonas`; `PUT /api/transportista/zonas` (multi-selección); `PUT /api/transportista/disponibilidad`.
- **App:** alta de vehículo (elige el tipo, la app propone sus medidas estándar y el Transportista las corrige; peso), pantalla separada de costos con ayuda por campo, selector de zonas, interruptor de disponibilidad.
- **Terminado:** nadie más que el dueño y el Administrador lee `vehiculo_costo` (test de RLS).

### Módulo 5 — Catálogo de objetos

- **Requisitos:** RN-08.
- **Base:** seed §2.2 y `SET NOT NULL` de las medidas.
- **Backend:** `GET /api/catalogo/objetos`.
- **App:** selector de objetos del catálogo (se usa en el módulo 6).

### Módulo 6 — Solicitud

- **Requisitos:** RF-06.
- **Decisiones:** D-20.
- **Base:** `solicitud.fecha_servicio_deseada` (date NOT NULL), `franja_horaria_inicio` y `franja_horaria_fin` (time, nullable).
- **Backend:**
  - `POST /api/solicitudes` (copia peso, medidas y flags de cada objeto a `solicitud_objeto`; no calcula ni devuelve monto).
  - `GET /api/solicitudes` (mías, con "vencida" calculada al vuelo), `GET /api/solicitudes/{id}`.
  - `POST /api/solicitudes/{id}/cancelar` (ofertas pendientes a `no_seleccionada`).
  - `POST /api/solicitudes/{id}/republicar` (sólo si está vencida; crea una nueva con otra fecha).
  - Interfaz `Geocodificador` sin proveedor conectado (D-17).
- **App:** publicar solicitud (selector de zona de origen y destino, dirección manual, fecha y franja, pisos, ascensor y distancia a pie, objetos del catálogo o manuales con medidas, ayudantes deseados), mis solicitudes, opción de republicar una vencida.
- **Terminado:** una solicitud con objetos manuales y de catálogo queda con todas las medidas copiadas; ningún endpoint de solicitud devuelve un monto.

### Módulo 7 — Matchmaking

- **Requisitos:** RN-04, RN-05 (aviso in-app).
- **Decisiones:** D-21, D-22.
- **Backend:** `GET /api/transportista/solicitudes` (compatibles según D-21, sin vencidas); al publicar una solicitud, un job async crea la notificación `solicitud_compatible` para cada Transportista compatible.
- **App:** listado de solicitudes compatibles y detalle.
- **Terminado:** un Transportista sin zona coincidente, sin vehículo con capacidad, no disponible, no habilitado o vetado no ve la solicitud.

### Módulo 8 — Oferta, precio y cálculo de viajes

- **Requisitos:** RF-17, RN-01, RN-02.
- **Decisiones:** D-13, D-14, D-15, D-23, D-24; `ALGORITMO_COTIZACION.md`, `ALGORITMO_VIAJES_EMPAQUETADO.md`.
- **Base:** `config_margen` (versionada, seed con el margen inicial que defina el equipo; 0 si no hay otro valor); tabla **`oferta_costo`** (1 a 1, RLS sólo Transportista dueño y Administrador) con las columnas de costo que hoy están en `oferta` (se mueven; la tabla está vacía); `trg_proteger_campos_oferta` sobre lo que queda en `oferta`.
- **Backend:** interfaz `Ruteador` con proveedores `aproximado` (local: línea recta × 1,3 a 30 km/h), `fijo` (tests) y `google` (producción, cuando se integre), elegidos con `RUTEO_PROVEEDOR`; con `APP_ENV=production` y otro proveedor, el backend no arranca. `POST /api/solicitudes/{id}/ofertas` (vehículo + ayudantes 0..3; `planificarViajes` con timeout de 5 s; precio con redondeo final; guarda `oferta` y `oferta_costo`), `POST /api/ofertas/{id}/retirar`, `GET /api/transportista/ofertas`. La vinculación de la cuenta de Mercado Pago como requisito para ofertar se activa en el módulo 12.
- **App:** armar oferta (elegir vehículo y ayudantes; ver precio y viajes calculados o el motivo si la carga no entra), mis ofertas.
- **Terminado:** tests table-driven de las funciones puras de precio y de viajes (casos de `ALGORITMO_*`); el Cliente no puede leer el desglose ni nadie modificar el precio (tests de RLS y del trigger).

### Módulo 9 — Top 3, perfil y aceptación

- **Requisitos:** RF-07, RF-11, RN-05.
- **Decisiones:** D-23, D-25; `ALGORITMO_SCORE.md`.
- **Backend:**
  - `GET /api/solicitudes/{id}/ofertas` (top 3 por score; `?ver_mas=true` pagina). Sólo expone: nombre del Transportista, calificación, tasa de cumplimiento, tipo de vehículo, precio, cantidad de viajes y de ayudantes (más los ayudantes solicitados). Nunca el desglose.
  - `GET /api/transportistas/{id}` (perfil: forma de trabajo, zonas, reseñas).
  - `POST /api/ofertas/{id}/aceptar`: llama a **`fn_aceptar_oferta`** (`SECURITY DEFINER`; verifica que quien llama sea el Cliente de la solicitud). En una transacción crea el `viaje`, copia el desglose de `oferta_costo` a **`viaje_costo`**, genera los dos PIN de 4 dígitos en **`viaje_pin`**, pasa las demás ofertas a `no_seleccionada` y la solicitud a `asignada`. La retención del pago se suma en el módulo 12.
- **Base:** tablas `viaje_costo` (las columnas de costo de los snapshots de `viaje` se mueven ahí; RLS Transportista del viaje y Administrador) y `viaje_pin` (RLS sólo Cliente del viaje y Administrador; nadie la escribe directo); se eliminan `viaje.pin_inicio` y `viaje.pin_fin`; función `fn_aceptar_oferta`. Las tablas están vacías.
- **App:** detalle de la solicitud con top 3 y "ver más", perfil del Transportista, confirmar elección. La patente se muestra recién después de aceptar.
- **Terminado:** tests del score (`ALGORITMO_SCORE.md` §5) y de la transacción de aceptación.

### Módulo 10 — Ejecución del viaje

- **Requisitos:** RF-10, RF-15, RF-20, RF-21, RF-22, RN-06, RI-04, RI-05.
- **Decisiones:** D-10, D-26, D-30.
- **Base:** `viaje.salio_en`; función **`fn_validar_pin(viaje_id, tipo, pin)`** (`SECURITY DEFINER`: verifica que quien llama sea el Transportista del viaje, compara, cuenta intentos, actualiza el estado y al quinto fallo abre el incidente; nunca devuelve el valor); condición de veto en las policies de `mensaje` y `viaje_ubicacion`; `rls-policy-review` sobre esas dos tablas; recién después, agregarlas a la publicación `supabase_realtime`.
- **Backend:** `GET /api/viajes/{id}` (al Cliente le muestra los PIN leídos de `viaje_pin`; al Transportista nunca), `POST /api/viajes/{id}/salida`, `POST /api/viajes/{id}/pin-inicio` y `/pin-fin` (llaman a `fn_validar_pin` con lat/lng y precisión; tolerancia 150 m y precisión 50 m registran discrepancia sin rechazar).
- **App:** viaje del Cliente (PIN visibles, mapa o seguimiento, chat); viaje del Transportista (botón "salí", carga de PIN, envío de ubicación cada 15 a 30 s con cola sin conexión, permiso de ubicación en segundo plano, chat).
- **Terminado:** un Transportista no puede obtener el PIN por ningún camino (test de RLS); el chat sólo existe con viaje confirmado.

### Módulo 11 — Cancelaciones

- **Requisitos:** RF-08, RF-19, RN-07.
- **Decisiones:** D-27, D-29.
- **Base:** `viaje.cancelado_en`, `cancelado_por_usuario_id`, `motivo_cancelacion`, `cargo_resarcimiento_monto`; trigger de `tasa_cumplimiento`.
- **Backend:** `POST /api/viajes/{id}/cancelar` (Cliente: cargo del 20 % si `salio_en` no es null; Transportista: sin cargo, la solicitud vuelve a `publicada`, baja la tasa).
- **App:** cancelar con aviso previo del cargo, si corresponde.
- **Terminado:** tests de los dos casos del Cliente y del caso del Transportista, incluida la tasa.

### Módulo 12 — Pagos

- **Requisitos:** RI-03, RN-03.
- **Decisiones:** D-09, D-28.
- **Base:** cuenta de Mercado Pago del Transportista (columna o tabla); índice único para la idempotencia de webhooks.
- **Backend:** vincular la cuenta (OAuth) y exigirla para ofertar; retención al aceptar (módulo 9); liberación al PIN de fin (módulo 10) salvo incidente abierto; `POST /api/webhooks/pagos/mercadopago` (sin JWT, con firma e idempotencia); reembolso del 80 % en la cancelación con cargo. Todo detrás de `pago.Gateway`.
- **App:** vincular la cuenta de Mercado Pago (Transportista); pagar al aceptar (Cliente).
- **Terminado:** flujo completo en el sandbox de Mercado Pago, con test del webhook repetido.

### Módulo 13 — Reseñas, incidentes, vetos y notificaciones

- **Requisitos:** RF-02, RF-03, RF-04, RF-09, RF-12, RF-13, RF-23.
- **Decisiones:** D-22, D-29, D-30.
- **Base:** trigger de `calificacion_promedio`.
- **Backend:** `POST /api/viajes/{id}/resena` (14 días); `POST /api/incidentes` (con adjuntos); administración: `GET /api/admin/incidentes`, `POST /api/admin/incidentes/{id}/resolver` (reembolso o liberación total/parcial, veto, cierre sin acción), `POST /api/admin/vetos`; middleware de veto (`cuenta_vetada`); `GET /api/notificaciones`, `POST /api/notificaciones/{id}/leida`.
- **App:** calificar, reportar un problema, notificaciones.
- **Terminado:** un usuario vetado recibe 403 en el backend y no puede escribir en el chat ni en el GPS.

### Módulo 14 — Historial

- **Requisitos:** RF-14, RF-24.
- **Backend:** `GET /api/viajes?rol=cliente|transportista` (paginado).
- **App:** historial de viajes de cada rol.

---

## 5. Fuera de esta etapa (D-17)

Notificaciones push (FCM), iOS, web admin en Angular, APIs de mapas, geocodificación y ruteo conectadas (los componentes quedan listos detrás de interfaces), internacionalización, facturación fiscal y preferencias de notificación.

# Fletway — Algoritmo de cotización

**Estado:** estimativo / de referencia. Documenta el diseño acordado para implementarlo en `fletway-backend` (Go). No reemplaza a `DECISIONES_TECNICAS.md`: si algo cambia, hay que actualizar ambos.

**Fuente:** adaptado del borrador `ALGORITMO_COTIZACION.md` (a su vez derivado de `Algoritmo_de_cotizacion_de_Fletway.pdf`). Los nombres de tabla y columna son los **del schema real de `dbFletway`**, incluidas las migraciones `migrations/0001`–`0007` (aplicadas el 2026-09-24).

**Convención de este documento:**
- `columna` sin marca → existía en el schema original.
- `columna` (nuevo) → agregada por las migraciones `0001`–`0007` para este algoritmo (**aplicadas el 2026-09-24**). Todas existen en la base.
- **Atención:** marca una diferencia con el borrador original, o un punto a tener en cuenta al implementar.

---

## 0. Decisiones de producto

### 0.1 Una sola cotización, no dos

La ERS (RN-01) describe el cálculo aplicado a dos cosas: a una "cotización estimada" al publicar la Solicitud y al "precio específico" de cada Oferta. **La cotización estimada previa se descarta.** El Cliente no ve ningún monto al publicar la Solicitud. El primer precio que ve de cada Transportista es el precio real y final de esa Oferta.

- `solicitud` **no** expone ningún precio al Cliente. Si se agrega algún chequeo interno por `tipo_vehiculo` (factibilidad, matchmaking), sólo puede usar cotas de peso y volumen (`ALGORITMO_VIAJES_EMPAQUETADO.md` §7) y **nunca se convierte en un monto visible**.
- El único cálculo de precio del sistema ocurre **cuando un Transportista se postula** (RF-17). Usa su `vehiculo` real (con `vehiculo_costo` (nuevo)) y la `oferta.cantidad_ayudantes` que elige.
- Cada Oferta puede tener un precio distinto: vehículo (consumo, capacidad, depreciación, seguro), cantidad de ayudantes y cantidad de viajes necesarios.

> **Atención:** `solicitud.cotizacion_estimada_monto` sigue existiendo (nullable), marcada **DEPRECATED** (0003). Se elimina en una migración posterior. El backend **no la escribe**.
>
> **Pendiente en la app:** `fletway-mobile/docs/API_CONTRATOS.md` todavía define `POST /solicitudes` → `CotizacionEstimada`. Hay que alinearlo cuando se construya ese endpoint.

### 0.2 Sin tramo de acercamiento

La distancia entre la ubicación del Transportista y el origen de la Solicitud (el trayecto "en vacío") **no se calcula ni se cobra**. El costo de ruta sale sólo de origen → destino (`solicitud.origen_lat/lng` → `solicitud.destino_lat/lng`). Es una simplificación deliberada, no un olvido: RN-01 menciona "distancia recorrida" sin aclarar si incluye el acercamiento.

### 0.3 Deuda conocida

RN-01 menciona la "demanda" como factor de precio. Este algoritmo no tiene ningún término de demanda ni tarifa dinámica: queda fuera del alcance del prototipo.

### 0.4 Decisión ABIERTA — ubicación de `margen_pct`

**No está resuelta. Este documento no la resuelve.** Hay dos alternativas:

- (a) **Parámetro de plataforma**, versionado como el resto de la config.
- (b) **Configurable por Transportista.** En ese caso el margen pasa a ser otra variable por la que compiten las ofertas.

Según cuál se elija cambia **dónde vive** el campo de configuración. Por eso el schema **no tiene ninguna columna de configuración de margen**. Lo único que existe es el **valor aplicado** en cada oferta (`oferta.margen_pct` (nuevo)) y congelarlo en el viaje (`viaje.margen_pct_snapshot` (nuevo)). Eso sirve para cualquiera de las dos opciones. En el pseudocódigo aparece como `margenPct`, sin decir de dónde se lee.

---

## 1. Cuándo corre el algoritmo

```
Transportista ve Solicitud compatible (RN-04/RN-05)
        │
        ▼
Transportista arma su Oferta (POST /solicitudes/{id}/ofertas):
  - elige su vehiculo           → oferta.vehiculo_id
  - indica cantidad de ayudantes → oferta.cantidad_ayudantes
        │
        ▼
Sistema ejecuta (con el rol del Transportista — RLS pass-through, D-02):
  1. planificarViajes()     → ver ALGORITMO_VIAJES_EMPAQUETADO.md (boxpacker3 v2, greedy;
                              estimativo para el precio, no el mínimo de viajes).
                              Si devuelve ErrNoFactible → error de validación, sin oferta.
  2. calcularRuta()         → distancia_km, duracion_h (origen→destino)
  3. duracionOperacion()    → horas de carga/descarga por viaje
  4. calcularCostoOferta()  → costo operativo total (suma de todos los viajes)
  5. precioFinal()          → precio que paga el Cliente
        │
        ▼
INSERT oferta (precio_calculado + desglose (nuevo))
        │
        ▼  (Cliente acepta — viaje_insert corre con el rol del CLIENTE)
INSERT viaje copiando el desglose de la oferta a columnas *_snapshot
```

El cálculo corre **una sola vez por Oferta creada**. El resultado se guarda en `oferta` y se congela en `viaje` cuando el Cliente la acepta.

> **Consecuencia de RLS que el borrador no contemplaba:** la policy real `viaje_insert` exige `cliente_id = auth.uid()`, así que el viaje lo crea **la sesión del Cliente**. Esa sesión **no puede** leer `vehiculo_costo` (nuevo), que es privada del Transportista. Por eso el snapshot de `viaje` no se recalcula: **se copia** desde las columnas de desglose de la `oferta` aceptada, (columnas agregadas en 0005).

---

## 2. Entradas del algoritmo (mapeo al schema real)

| Origen | Columnas reales | Estado |
|---|---|---|
| `solicitud` | `origen_lat`, `origen_lng`, `destino_lat`, `destino_lng` (numeric(9,6)) | Existen |
| `solicitud` | `pisos_origen` (nuevo), `ascensor_utilizable_origen` (nuevo), `distancia_vehiculo_origen_m` (nuevo), `pisos_destino` (nuevo), `ascensor_utilizable_destino` (nuevo), `distancia_vehiculo_destino_m` (nuevo) | Existe (`requiere_escalera` y `pisos_escalera` quedan DEPRECATED) |
| `solicitud_objeto` | `cantidad`, `peso_unitario_kg`, `largo_m` (nuevo), `ancho_m` (nuevo), `alto_m` (nuevo) | Existe (`volumen_unitario_m3` queda DEPRECATED) |
| `oferta` (en construcción) | `vehiculo_id`, `cantidad_ayudantes` | Existen |
| `vehiculo` | `peso_maximo_kg` (**es carga útil**), `largo_util_m` (nuevo), `ancho_util_m` (nuevo), `alto_util_m` (nuevo) | Existe (`volumen_carga_m3` queda DEPRECATED) |
| `vehiculo_costo` (nuevo) (1:1, privada) | ver §4.2 | Existe |
| Resultado de `planificarViajes()` | cantidad de viajes y carga de cada uno | → `oferta.cantidad_viajes` (existente) |
| `config_costo_laboral` (nuevo) (vigente) | ver §4.1 | Existe (1 fila seed ilustrativa) |
| `config_operacion` (nuevo) (vigente) | ver §4.3 | Existe (1 fila seed ilustrativa) |
| `config_comision` (vigente) | `porcentaje` (**0–100**, numeric(5,2)) | Existe; legible por cualquier usuario autenticado desde 0007 |
| `config_impuesto` (nuevo) (vigente) | `iva_pct` (0–100) | Existe (1 fila seed: 21 %) |
| `margenPct` | — | **decisión abierta** (§0.4) |

> **Diferencias con el borrador:**
> - El borrador lista `solicitud.rango_horario` como entrada. **Esa columna no existe** y **el algoritmo no la usa** en ninguna fórmula. No se creó.
> - Pisos, ascensor y distancia a pie **por separado para origen y destino** se agregaron en 0003 y reemplazan a `requiere_escalera` + `pisos_escalera`.
> - Existe `solicitud.cantidad_ayudantes_solicitados`, que el borrador no menciona. El algoritmo usa `oferta.cantidad_ayudantes`, que decide el Transportista. **No está definido** si la oferta debe respetar el valor que pidió el Cliente (¿`>=`?, ¿igual?, ¿sólo informativo?). Queda abierto; este documento no lo resuelve.
> - `config_tarifa` (el modelo de tarifa plana original) **no se usa** en este algoritmo y quedó DEPRECATED (0004).

---

## 3. Estructuras (Go, alineadas al schema)

Los nombres de campo Go siguen a las columnas reales. Un comentario indica de qué columna sale cada uno.

```go
type Acceso struct {
    Pisos              int     // solicitud.pisos_origen / pisos_destino (nuevo)
    AscensorUtilizable bool    // solicitud.ascensor_utilizable_origen / _destino (nuevo)
    DistanciaVehiculoM float64 // solicitud.distancia_vehiculo_origen_m / _destino_m (nuevo)
}

type Solicitud struct {
    OrigenLat, OrigenLng   float64 // solicitud.origen_lat / origen_lng
    DestinoLat, DestinoLng float64 // solicitud.destino_lat / destino_lng
    AccesoOrigen           Acceso
    AccesoDestino          Acceso
}

// Una fila de solicitud_objeto. Los ítems de catálogo (objeto_id) también
// traen su copia propia de peso y dimensiones en la fila, igual que hoy
// con peso_unitario_kg.
type Objeto struct {
    Nombre             string  // objeto.nombre | solicitud_objeto.nombre_personalizado
    PesoKg             float64 // solicitud_objeto.peso_unitario_kg
    LargoM, AnchoM, AltoM float64 // solicitud_objeto.largo_m / ancho_m / alto_m (nuevo) (alto = eje vertical)
    // Con boxpacker3 v2 se aplican como restricciones reales. Única excepción:
    // rotacion_horizontal=false con rotacion_vertical=true queda informativa.
    // Ver ALGORITMO_VIAJES_EMPAQUETADO.md §2.1.
    RotacionHorizontal bool    // solicitud_objeto.rotacion_horizontal (nuevo)
    RotacionVertical   bool    // solicitud_objeto.rotacion_vertical (nuevo)
    Apilable           bool    // solicitud_objeto.apilable (nuevo)
}
// VolumenM3() = LargoM * AnchoM * AltoM

type Item struct {
    Objeto   Objeto
    Cantidad int // solicitud_objeto.cantidad (CHECK > 0)
}

type Vehiculo struct {
    Patente                   string  // vehiculo.patente
    PesoUtilKg                float64 // vehiculo.peso_maximo_kg — YA es carga útil
    LargoUtilM, AnchoUtilM, AltoUtilM float64 // vehiculo.largo_util_m / ancho_util_m / alto_util_m (nuevo)
    Costo                     VehiculoCosto
}
// VolumenUtilM3() = LargoUtilM * AnchoUtilM * AltoUtilM

type VehiculoCosto struct { // vehiculo_costo (nuevo) (1:1, privada)
    CombustiblePrecioL   float64 // combustible_precio_l
    RendimientoKmL       float64 // rendimiento_km_l      (CHECK > 0)
    CantidadNeumaticos   int     // cantidad_neumaticos
    CostoNeumatico       float64 // costo_neumatico
    VidaNeumaticoKm      float64 // vida_neumatico_km     (CHECK > 0)
    CostoMantenimientoKm float64 // costo_mantenimiento_km
    ValorCompra          float64 // valor_compra
    ValorResidual        float64 // valor_residual        (CHECK <= valor_compra)
    VidaUtilKm           float64 // vida_util_km          (CHECK > 0)
    SeguroMensual        float64 // seguro_mensual
    PatenteMensual       float64 // patente_mensual
}

// Un viaje físico de la planificación, NO la tabla `viaje`.
// planificarViajes() sólo completa Carga; el resto lo completa quien arma la
// oferta antes de llamar a calcularCostoOferta().
type Viaje struct {
    Solicitud         Solicitud
    Vehiculo          Vehiculo
    CantidadAyudantes int // oferta.cantidad_ayudantes
    Carga             []Item
}

type Ruta struct {
    DistanciaKm float64
    DuracionH   float64
}
```

> **Discrepancias con las estructuras del borrador:**
> - **`PesoMaximoKg − PesoVehiculoKg` se elimina.** El borrador calculaba la carga útil restando la tara. Confirmado con el humano (2026-09-24): `vehiculo.peso_maximo_kg` **ya es carga útil**, así que no se agrega `peso_vehiculo_kg`.
> - **`Objeto.Categoria` se elimina.** No existe en `objeto` y ninguna fórmula la usa.
> - **Tipo `Viaje` (Go) ≠ tabla `viaje`.** En el algoritmo, "viaje" es cada ida con carga dentro de una misma oferta (`oferta.cantidad_viajes`). La tabla `viaje` es el servicio confirmado, que es 1:1 con la oferta aceptada. El borrador los mezclaba.
> - **Se quita `Viaje.Inicio time.Time`.** Ninguna fórmula lo usa y en la base no hay dónde sacarlo (no hay `rango_horario`).

---

## 4. Costo operativo

Es lo que le cuesta al Transportista hacer el servicio, sin margen, comisión ni impuestos: costo laboral + costo del vehículo + costos adicionales (fijos, no se calculan automáticamente en esta versión).

**Convención de porcentajes:** en la base van de 0 a 100 (`numeric(5,2)`), igual que `config_comision.porcentaje`. El código los divide por 100 al leerlos.

### 4.1 Costo laboral (por hora) — lee `config_costo_laboral` (nuevo) vigente

```go
// Fila vigente de config_costo_laboral (nuevo) (vigente_hasta IS NULL).
// Todos los *Pct llegan en 0–100.
type CostoLaboral struct {
    SalarioBasicoChofer, SalarioBasicoAyudante       float64 // salario_basico_chofer / _ayudante
    AdicionalesPctChofer, AdicionalesPctAyudante     float64 // adicionales_pct_chofer / _ayudante
    ViaticosDiarios                                  float64 // viaticos_diarios (no remunerativo)
    ContribucionesSegSocialPct, ObraSocialPct, ArtPct float64 // contribuciones_seg_social_pct, obra_social_pct, art_pct
    SeguroVidaMensual                                float64 // seguro_vida_mensual
    HorasMensuales, HorasDiarias                     float64 // horas_mensuales / horas_diarias
}

// Aguinaldo (SAC) = 1 sueldo extra por año → factor 13/12. Es la definición
// legal del SAC, no un parámetro: no se guarda en la base.
const aguinaldoFactor = 13.0 / 12.0

func costoPorHoraTrabajo(c CostoLaboral, salarioBasico, adicionalesPct float64) float64 {
    remunerativo := salarioBasico * (1 + adicionalesPct/100) * aguinaldoFactor
    cargas := remunerativo * (c.ContribucionesSegSocialPct + c.ObraSocialPct + c.ArtPct) / 100
    diasMensuales := c.HorasMensuales / c.HorasDiarias
    noRemunerativo := c.ViaticosDiarios*diasMensuales + c.SeguroVidaMensual
    return (remunerativo + cargas + noRemunerativo) / c.HorasMensuales
}

func costoPorHoraChofer(c CostoLaboral) float64 {
    return costoPorHoraTrabajo(c, c.SalarioBasicoChofer, c.AdicionalesPctChofer)
}
func costoPorHoraAyudante(c CostoLaboral) float64 {
    return costoPorHoraTrabajo(c, c.SalarioBasicoAyudante, c.AdicionalesPctAyudante)
}
```

Valores semilla **ilustrativos**, tomados del borrador (en la seed de `0004`): chofer $1.037.544,76, ayudante $963.809,78, adicionales 18 % y 16 %, viáticos $24.724,33/día, seguridad social 18 %, obra social 6 %, ART 10 %, seguro de vida $424,62/mes, 192 h/mes, 8 h/día.

> **Inconsistencia interna del borrador (no resuelta):** el texto dice "alícuota de ART del empleador al **18 %**", pero la constante es `ArtPct = 0.10`. El 18 % es `ContribucionesSegSocialPct`. La seed usa **10 %**, el valor del código. Hay que confirmarlo contra la normativa vigente antes de producción.
>
> **Cambio respecto del borrador:** `AguinaldoPct = 13.0/12.0 − 1.0` se reemplaza por `aguinaldoFactor = 13/12`. Es la misma matemática y no se parametriza.

### 4.2 Costo del vehículo — lee `vehiculo_costo` (nuevo)

```go
func costoPorHoraVehiculo(v VehiculoCosto, horasMensuales float64) float64 {
    fijosMensuales := v.SeguroMensual + v.PatenteMensual
    return fijosMensuales / horasMensuales // horas_mensuales de config_costo_laboral (nuevo)
}

func costoPorKmVehiculo(v VehiculoCosto) float64 {
    combustible := v.CombustiblePrecioL / v.RendimientoKmL
    neumaticos := float64(v.CantidadNeumaticos) * v.CostoNeumatico / v.VidaNeumaticoKm
    mantenimiento := v.CostoMantenimientoKm
    depreciacion := (v.ValorCompra - v.ValorResidual) / v.VidaUtilKm
    return combustible + neumaticos + mantenimiento + depreciacion
}
```

Estos campos son **de cada vehículo**, no de la config global. Por eso dos ofertas para la misma Solicitud pueden tener precios distintos.

> **Por qué es una tabla aparte:** `vehiculo_select` es `USING (true)` (lectura pública). Si estos campos fueran columnas de `vehiculo`, cualquier usuario vería el valor de compra y el seguro de cada Transportista. Por eso viven en **`vehiculo_costo` (nuevo)** (1:1, PK = `vehiculo_id`), con policies de select/insert/update sólo para el dueño y el Administrador. Decisión confirmada el 2026-09-24.
>
> Consecuencia: **sólo la sesión del Transportista dueño puede calcular** con estos datos. Por eso el desglose se guarda en `oferta` (§6).
>
> Los divisores (`rendimiento_km_l`, `vida_neumatico_km`, `vida_util_km`) tienen `CHECK > 0` en la base, así que no hay división por cero.
>
> **Observación (no resuelta):** `combustible_precio_l` es por vehículo, como dice el borrador. Eso permite diferenciar diésel, nafta y GNC, pero obliga al Transportista a actualizar el precio de mercado a mano.

### 4.3 Duración de la operación (carga y descarga) — lee `config_operacion` (nuevo) vigente

Modelo lineal paramétrico. Respecto del documento fuente se mantienen las dos correcciones del borrador: `TiempoPorM3Min` ya no pisa a `TiempoPorKgMin`, y `acensor` → `ascensor`.

```go
// Fila vigente de config_operacion (nuevo).
type Operacion struct {
    TiempoBaseOperacionMin float64 // tiempo_base_operacion_min (seed 10)
    TiempoEsperaMin        float64 // tiempo_espera_min (seed 30) — reservado, no se usa
    TiempoPorObjetoMin     float64 // tiempo_por_objeto_min (seed 2)
    TiempoPorKgMin         float64 // tiempo_por_kg_min (seed 0.01)
    TiempoPorM3Min         float64 // tiempo_por_m3_min (seed 8)
    TiempoPorMetroMin      float64 // tiempo_por_metro_min (seed 0.02)
    TiempoPorEscaleraMin   float64 // tiempo_por_escalera_min (seed 5)
    EficienciaAyudante     float64 // eficiencia_ayudante (seed 0.7)
}

func duracionOperacion(op Operacion, v Viaje) float64 {
    nObjetos := sumCantidad(v.Carga)
    pesoKg := sumPeso(v.Carga)       // Σ cantidad * peso_unitario_kg
    volumenM3 := sumVolumen(v.Carga) // Σ cantidad * largo_m*ancho_m*alto_m

    o, d := v.Solicitud.AccesoOrigen, v.Solicitud.AccesoDestino
    distanciaM := o.DistanciaVehiculoM + d.DistanciaVehiculoM
    pisos := o.Pisos*boolToInt(!o.AscensorUtilizable) + d.Pisos*boolToInt(!d.AscensorUtilizable)

    trabajoMin := op.TiempoBaseOperacionMin +
        float64(nObjetos)*op.TiempoPorObjetoMin +
        pesoKg*op.TiempoPorKgMin +
        volumenM3*op.TiempoPorM3Min +
        distanciaM*op.TiempoPorMetroMin +
        float64(pisos)*op.TiempoPorEscaleraMin

    n := float64(v.CantidadAyudantes)
    eficiencia := 1 + math.Pow(op.EficienciaAyudante, n)*n

    return trabajoMin / eficiencia / 60.0 // horas
}
```

> - `tiempo_espera_min` se declara en la fuente, pero **ninguna fórmula lo usa**. Queda en la config como reservado. **Hay que confirmar** si debe sumarse a `trabajoMin`.
> - **Observación sobre la fórmula de eficiencia (heredada, no corregida):** con 0,7, `1 + 0,7ⁿ·n` da 1,70 (n=1), 1,98 (n=2), 2,03 (n=3), 1,96 (n=4) y 1,84 (n=5). **A partir del 4.º ayudante la operación se vuelve más lenta**, pero el costo laboral sigue subiendo. Así que agregar ayudantes nunca abarata la oferta desde n=3. Si eso es intencional (rendimientos decrecientes), conviene dejarlo escrito; si no, hay que revisar la fórmula.
> - Los accesos salen de `solicitud.pisos_*`, `ascensor_utilizable_*` y `distancia_vehiculo_*_m` (nuevo). Si el Cliente no los informa, los defaults son planta baja, sin ascensor y vehículo en la puerta (0 / false / 0).

### 4.4 Costo por viaje y por oferta

```go
func calcularCostoViaje(cl CostoLaboral, v Viaje, ruta Ruta, operacionH float64) (laboral, vehiculo float64) {
    costoLaborH := costoPorHoraChofer(cl) + float64(v.CantidadAyudantes)*costoPorHoraAyudante(cl)
    costoVehiculoH := costoPorHoraVehiculo(v.Vehiculo.Costo, cl.HorasMensuales)
    costoVehiculoKm := costoPorKmVehiculo(v.Vehiculo.Costo)

    h := ruta.DuracionH + operacionH
    km := ruta.DistanciaKm * 2 // ida y vuelta, sin tramo de acercamiento (§0.2)

    return costoLaborH * h, costoVehiculoH*h + costoVehiculoKm*km
}

type DesgloseOferta struct { // se persiste en oferta (nuevo) — ver §6
    DistanciaKm, DuracionRutaH, DuracionOperacionH float64
    CostoLaboral, CostoVehiculo, CostosAdicionales float64
    CostoOperativo                                 float64
}

func calcularCostoOferta(cl CostoLaboral, op Operacion, viajes []Viaje, ruta Ruta, costosAdicionales float64) DesgloseOferta {
    d := DesgloseOferta{DistanciaKm: ruta.DistanciaKm, DuracionRutaH: ruta.DuracionH, CostosAdicionales: costosAdicionales}
    for _, v := range viajes {
        operacionH := duracionOperacion(op, v)
        lab, veh := calcularCostoViaje(cl, v, ruta, operacionH)
        d.DuracionOperacionH += operacionH
        d.CostoLaboral += lab
        d.CostoVehiculo += veh
    }
    d.CostoOperativo = d.CostoLaboral + d.CostoVehiculo + d.CostosAdicionales
    return d
}
```

`costosAdicionales` (peajes, estacionamiento, embalaje, grúa) llega como monto fijo, 0 por defecto. No se calcula automáticamente en esta versión.

> **Cambios respecto del borrador (en pseudocódigo, sin cambio de modelo):**
> - `calcularCostoViaje` devuelve los componentes (laboral y vehículo) por separado para poder guardarlos en `oferta` y `viaje`. La suma es la misma que en el borrador.
> - La ruta es **una sola** (`Ruta`, no `[]Ruta`). Todos los viajes de una oferta hacen el mismo origen → destino, así que `rutas[i]` del borrador sería siempre igual.
>
> **Ambigüedades heredadas del borrador (no resueltas, confirmar antes de implementar):**
> 1. **`costosAdicionales` por viaje o por oferta.** El borrador lo suma **dentro del loop**, o sea N veces. Para peajes puede tener sentido (uno por ida), pero no para embalaje o grúa. El pseudocódigo de arriba lo suma **una vez por oferta**, como supuesto a validar.
> 2. **Horas de ida contra km de ida y vuelta.** `km = DistanciaKm * 2` cobra la vuelta en kilómetros, pero `h = DuracionH + operacionH` usa sólo la duración de ida. No cobra las horas de chofer y ayudantes del regreso. ¿Es a propósito?
> 3. **Proveedor de ruteo.** `calcularRuta()` necesita un servicio de distancia y duración (OSRM, Google, etc.). No está definido en ninguna parte del repo.

---

## 5. Precio al cliente

```go
// comisionPct: config_comision.porcentaje vigente (0–100).
// ivaPct:      config_impuesto.iva_pct (nuevo) vigente (0–100).
// margenPct:   DECISIÓN ABIERTA — no se define de dónde se lee (§0.4).
func precioNeto(costoOperativo, margenPct, comisionPct float64) float64 {
    return costoOperativo * (1 + margenPct/100) / (1 - comisionPct/100)
}

func precioFinal(precioNeto, ivaPct float64) float64 {
    return precioNeto * (1 + ivaPct/100)
}
```

- La comisión se calcula sobre el precio neto (lo que paga el Cliente antes de impuestos), no sobre el costo operativo. Un 15 % quiere decir "Fletway se queda con el 15 % de lo que paga el Cliente".
- **Discrepancia de escala:** el borrador usa `ComisionPct = 0.15`, pero `config_comision.porcentaje` en la base es **numeric(5,2) con CHECK 0–100**, o sea 15.00. Por eso el pseudocódigo divide por 100.
- **Permisos:** con RLS pass-through el cálculo corre con el rol del Transportista, que necesita leer la comisión vigente. Por eso la migración 0007 reemplazó la policy `config_comision_admin` (FOR ALL, sólo admin) por `config_comision_select USING (true)` + insert/update/delete sólo admin.
- `iva_pct` supone 21 %. El tratamiento fiscal definitivo depende de la estructura jurídica de Fletway y del Transportista. Se parametriza en `config_impuesto` (nuevo), no se hardcodea.
- **`margenPct`: decisión abierta** (§0.4). El borrador lo dejaba "global por defecto (0.00)". Acá **no** se toma esa decisión: el valor aplicado se guarda en `oferta.margen_pct` (nuevo) sin importar de dónde salga.

---

## 6. Qué se guarda en `oferta` y en `viaje`

| Variable del algoritmo | `oferta` | `viaje` (snapshot al aceptar) |
|---|---|---|
| cantidad de viajes | `cantidad_viajes` (existente, CHECK > 0) | `cantidad_viajes_snapshot` (nuevo) |
| cantidad de ayudantes | `cantidad_ayudantes` (existente) | `cantidad_ayudantes` (existente, columna operativa) |
| distancia origen→destino | `distancia_km` (nuevo) | `distancia_km_snapshot` (existente, se reutiliza) |
| duración de ruta | `duracion_ruta_h` (nuevo) | `duracion_ruta_h_snapshot` (nuevo) |
| duración de operación (Σ viajes) | `duracion_operacion_h` (nuevo) | `duracion_operacion_h_snapshot` (nuevo) |
| costo laboral | `costo_laboral` (nuevo) | `costo_laboral_snapshot` (nuevo) |
| costo del vehículo | `costo_vehiculo` (nuevo) | `costo_vehiculo_snapshot` (nuevo) |
| costos adicionales | `costos_adicionales` (nuevo) | `costos_adicionales_snapshot` (nuevo) |
| costo operativo | `costo_operativo` (nuevo) | `costo_operativo_snapshot` (nuevo) |
| margen aplicado | `margen_pct` (nuevo) | `margen_pct_snapshot` (nuevo) |
| precio neto | `precio_neto` (nuevo) | `precio_neto_snapshot` (nuevo) |
| % comisión aplicado | `porcentaje_comision` (nuevo) | `porcentaje_comision_snapshot` (existente, se reutiliza) |
| % IVA aplicado | `iva_pct` (nuevo) | `iva_pct_snapshot` (nuevo) |
| **precio final** | `precio_calculado` (existente) | `monto_total_snapshot` (existente, se reutiliza) |

> - Los snapshots del **modelo de tarifa plana** (`tarifa_base_snapshot`, `valor_por_km_snapshot`, `valor_por_m3_snapshot`, `valor_por_hora_snapshot`, `recargo_escalera_snapshot`, `recargo_ayudante_snapshot`) quedaron **nullable y DEPRECATED** (0006). El backend **no los escribe**.
> - `trg_proteger_campos_viaje` (función `fn_proteger_campos_viaje`, reescrita en 0006) impide que Cliente o Transportista modifiquen por UPDATE todas las columnas de la tabla de arriba, además de las FKs. `distancia_km_snapshot` también quedó protegida.
> - El borrador proponía sólo `costo_operativo` en `oferta`. Se guardó **el desglose completo** porque el viaje lo inserta la sesión del Cliente (§1) y tiene que copiarlo de algún lado.

---

## 7. Checklist de implementación

1. Leer `config_costo_laboral` (nuevo), `config_operacion` (nuevo), `config_comision` y `config_impuesto` (nuevo) **vigentes** (`vigente_hasta IS NULL`) una vez por cálculo. No hardcodear constantes. Los porcentajes llegan en 0–100.
2. `costoPorHoraVehiculo` y `costoPorKmVehiculo` leen `vehiculo_costo` (nuevo) del `oferta.vehiculo_id`. Si el vehículo **no tiene fila** en `vehiculo_costo`, devolver un error de validación. Nunca calcular con ceros.
3. Implementar `duracionOperacion` con la fórmula de §4.3, usando `TiempoPorM3Min` y no `TiempoPorKgMin`.
4. `calcularCostoOferta` recorre los viajes que devuelve `planificarViajes` (ver `ALGORITMO_VIAJES_EMPAQUETADO.md`). No asumir un solo viaje.
5. `precioFinal` → `oferta.precio_calculado` es el único monto que ve el Cliente. **No exponer** `costo_operativo`, `precio_neto` ni el desglose en respuestas de API al Cliente, sólo al propio Transportista y al Administrador.
   **Atención:** A nivel base, `oferta_select` y `viaje_select` **sí** dejan al Cliente leer esas columnas (lectura directa por PostgREST). Ver §8.
6. No calcular, escribir ni exponer ningún monto en el endpoint de creación de `solicitud`. No escribir `cotizacion_estimada_monto`.
7. Al aceptar la oferta, el `viaje` copia el desglose de la `oferta` (§6). No lo recalcula.
8. No implementar la fuente de `margenPct` hasta que se resuelva §0.4.
9. Resolver antes de implementar las ambigüedades de §4.3 (`tiempo_espera_min`, eficiencia) y §4.4 (costos adicionales, horas de vuelta, proveedor de ruteo).

---

## 8. Estado del schema y pendientes

**Migraciones aplicadas el 2026-09-24** (versionadas en `migrations/`):

| Migración | Qué hizo |
|---|---|
| `0001_add_dimensiones_objeto_solicitud_objeto` | `largo_m/ancho_m/alto_m` + `rotacion_horizontal/rotacion_vertical/apilable` en `objeto` (dimensiones nullable) y `solicitud_objeto` (NOT NULL). `solicitud_objeto.volumen_unitario_m3` → DEPRECATED |
| `0002_add_dimensiones_vehiculo_create_vehiculo_costo` | `vehiculo.largo_util_m/ancho_util_m/alto_util_m`. Tabla `vehiculo_costo` con RLS dueño/admin. `vehiculo.volumen_carga_m3` → DEPRECATED |
| `0003_add_acceso_origen_destino_solicitud` | Pisos, ascensor y distancia a pie por origen y destino en `solicitud`. `requiere_escalera`, `pisos_escalera` y `cotizacion_estimada_monto` → DEPRECATED |
| `0004_create_config_costo_laboral_operacion_impuesto` | Tablas `config_costo_laboral`, `config_operacion` y `config_impuesto`, versionadas y con seed ilustrativa. `config_tarifa` → DEPRECATED |
| `0005_add_costos_oferta` | Desglose de costo en `oferta` (§6) |
| `0006_redisenar_snapshot_costo_viaje` | Snapshots del desglose en `viaje` + reescritura de `fn_proteger_campos_viaje`. Snapshots de tarifa plana → DEPRECATED |
| `0007_split_policy_config_comision` | `config_comision` legible por autenticados; escritura sólo admin |

**Pendientes de datos:**
- Las 5 filas del catálogo `objeto` **no tienen dimensiones** (`largo_m/ancho_m/alto_m` en NULL). Hasta que el Administrador las cargue, esos objetos no se pueden copiar a una `solicitud_objeto`. Cuando estén cargadas: migración para `SET NOT NULL`.
- Los valores de `config_costo_laboral`, `config_operacion` y `config_impuesto` son **ilustrativos** (los del borrador). Hay que reemplazarlos por los vigentes antes de producción. **Atención:** El borrador dice "ART al 18 %" en el texto, pero la constante es 10 %; la seed usa 10 %.

**Limpieza posterior (DROP, no reversible, requiere confirmación aparte):** cuando el backend ya no lea lo deprecado, borrar `solicitud_objeto.volumen_unitario_m3`, `vehiculo.volumen_carga_m3`, `solicitud.requiere_escalera/pisos_escalera/cotizacion_estimada_monto`, los 6 snapshots de tarifa plana de `viaje` y la tabla `config_tarifa`. Para `viaje`, primero hay que reescribir `fn_proteger_campos_viaje` sin esas columnas y después hacer el DROP. `objeto.volumen_estimado_m3` es redundante con las dimensiones; se puede borrar una vez cargadas.

**Riesgos de RLS detectados (no resueltos):**
1. **Exposición del desglose:** `oferta_select` deja al Cliente dueño de la solicitud leer **todas** las columnas de la oferta, y `viaje_select` hace lo mismo con el viaje. Con lectura directa vía PostgREST, el Cliente vería `costo_operativo` y el desglose. Posibles mitigaciones: GRANT por columna, una tabla 1:1 separada, o aceptar que sólo la API los oculta.
2. **Integridad de `oferta`:** `oferta_update` deja a Cliente y Transportista modificar **cualquier** columna, incluidos `precio_calculado` y el desglose. No hay trigger de protección como en `viaje`. Correspondería un `trg_proteger_campos_oferta` (patrón de `trg_proteger_campos_viaje`).

**Decisiones abiertas:**
- Ubicación de `margen_pct` (§0.4).
- Relación entre `solicitud.cantidad_ayudantes_solicitados` y `oferta.cantidad_ayudantes` (§2).
- Ambigüedades de fórmula de §4.3 y §4.4.

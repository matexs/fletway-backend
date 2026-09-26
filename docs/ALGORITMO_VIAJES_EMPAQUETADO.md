# Fletway — Algoritmo de cálculo de viajes y ubicación de objetos

**Estado:** estimativo / de referencia. Documenta el diseño acordado para implementarlo en `fletway-backend` (Go) con [`bavix/boxpacker3` **v2**](https://github.com/bavix/boxpacker3) (módulo `github.com/bavix/boxpacker3/v2`, versión v2.0.0 del 2026-09-04).

**Corresponde a:** RN-02 de la ERS ("cálculo automático de la cantidad de viajes", en base al volumen de carga y la capacidad del vehículo).

**Fuente:** adaptado del borrador `ALGORITMO_VIAJES_EMPAQUETADO.md`, que estaba escrito para la v1. Los nombres de tabla y columna son los **del schema real de `dbFletway`**, incluidas las migraciones `migrations/0001`–`0007` (aplicadas el 2026-09-24). La sección de la librería se reescribió después de leer el código de la v2.0.0 y medirla (2026-09-24).

**Convención:** `columna` = existía en el schema original · `columna` (nuevo) = agregada por las migraciones `0001`–`0007` (**aplicadas el 2026-09-24**; todas existen) · **Atención:** marca una diferencia con el borrador o punto a tener en cuenta.

---

## 0. Objetivo: un estimativo para el precio, no el mínimo de viajes

La cantidad de viajes **multiplica el costo** de la oferta (`ALGORITMO_COTIZACION.md` §4.4). Lo que se busca es una estimación **realista y rápida** que sirva para calcular el precio. **No se busca garantizar el mínimo de viajes.** Por eso:

- Se usa la heurística **greedy** de la librería (una sola pasada), no los modos de búsqueda.
- Se acepta que el resultado quede algo por encima del óptimo. Medido: entre 0 y 3 viajes por encima de la cota inferior que informa la propia librería (§6).
- Se desactivan las post-optimizaciones que no mejoran la cantidad de viajes y cuestan tiempo (§3).

---

## 1. Qué resuelve y qué no resuelve `boxpacker3` v2

Verificado leyendo el código de la v2.0.0 y con pruebas propias, no sólo con el README.

**Ejes:** en la v2 **`Depth` es el eje vertical** (gravedad, apoyo y apilado se evalúan sobre `DepthAxis`). Mapeo fijo en todo el flujo: `Width` = largo, `Height` = ancho, `Depth` = **alto**.

| Necesidad del modelo | ¿La resuelve la v2? | Cómo |
|---|---|---|
| Peso máximo del vehículo | Sí | `BoxSpec.MaxWeight` |
| Ubicar sin superposición | Sí | Chequeo geométrico interno |
| Cantidad de viajes sin buscar N a mano | Sí | Un solo `BoxSpec` con `Quantity: maxViajes`. La librería abre copias del vehículo a medida que las necesita y `len(result.Boxes)` es la cantidad de viajes |
| Cantidad por ítem | Sí | `ItemSpec.Quantity` (no hay que expandir unidad por unidad) |
| `rotacion_vertical = false` (no se puede voltear) | Sí | `ItemSpec.VerticalAxes = []Axis{DepthAxis}`: el alto queda vertical y el objeto sólo gira sobre el piso |
| `rotacion_horizontal = false` **y** `rotacion_vertical = false` | Sí | `ItemSpec.Rotation = RotationNever`: sin ninguna rotación, el largo del objeto va a lo largo del vehículo |
| `rotacion_horizontal = false` **sola** (se puede voltear, pero no girar sobre el piso) | **No directo** | No hay una opción para eso. Se podría implementar con una `PlacementRule` propia que inspeccione la orientación. **En esta versión se trata como informativa** (§2.1) |
| `apilable = false` | Sí | `ItemSpec.NothingOnTop = true` |
| Objetos "flotando" sin apoyo | Sí | `Rules{MinSupportRatio}` exige que un porcentaje de la base esté apoyado. Se usa 0,6 (§3) |
| Orientaciones inestables (alto y angosto) | Sí, automático | La librería las descarta salvo que el objeto no tenga ninguna orientación estable que entre |
| Por qué no entró un objeto | Sí | `result.Unpacked[i].Reason`: `ReasonTooBig`, `ReasonTooHeavy`, `ReasonNoOrientationFits`, `ReasonNoRoom`, … |
| Cota inferior de viajes | Sí | `result.Report.Bound.Boxes`, útil para logs y para la memoria del TFG |
| Peso máximo que soporta un objeto encima | Sí, disponible, **no usado** | `ItemSpec.MaxLoadOnTop` (kg). El schema no tiene esa columna y no se propone |
| Mínimo de viajes garantizado | No | Es una heurística. No se necesita (§0) |
| Centro de gravedad, carga por eje, orden de descarga, fragilidad | No | Fuera de alcance |
| Tiempo máximo de cómputo confiable | Parcial | `WithBudget` existe, pero en las pruebas el buscador `NewSearch` lo excedió (54 s con un límite de 5 s), y al cortar devuelve error sin resultado parcial. **No se usa.** Ver §3 |

> **Diferencia con el borrador (que describía la v1):** el borrador decía que la librería "no soporta restringir rotación por ítem" y que "no tiene concepto de apilamiento/soporte". **Eso es cierto para la v1.3.2, no para la v2.** Con la v2, `rotacion_vertical` y `apilable` dejan de ser metadata informativa y **se aplican como restricciones reales**. Sólo `rotacion_horizontal` aislada sigue siendo informativa.

---

## 2. Entradas (mapeo al schema real)

| Origen | Columnas | Estado |
|---|---|---|
| `solicitud_objeto` (por `solicitud_id`) | `cantidad` (CHECK > 0), `peso_unitario_kg`, `objeto_id` **o** `nombre_personalizado` (CHECK `chk_objeto_o_personalizado`) | Existen |
| `solicitud_objeto` | `largo_m` (nuevo), `ancho_m` (nuevo), `alto_m` (nuevo) (NOT NULL), `rotacion_horizontal` (nuevo), `rotacion_vertical` (nuevo), `apilable` (nuevo) (NOT NULL DEFAULT true) | Existe (`volumen_unitario_m3` queda DEPRECATED) |
| `objeto` (sólo el nombre, si `objeto_id` no es null) | `nombre` | Existe |
| `vehiculo` (el `oferta.vehiculo_id` en curso) | `patente`, `peso_maximo_kg` (**carga útil**) | Existen |
| `vehiculo` | `largo_util_m` (nuevo), `ancho_util_m` (nuevo), `alto_util_m` (nuevo) (NOT NULL, CHECK > 0) | Existe (`volumen_carga_m3` queda DEPRECATED) |

**Requisito de datos:** la librería trabaja en 3D, así que cada ítem y cada vehículo necesitan **tres dimensiones por separado**. Un volumen ya multiplicado no alcanza. Además, `alto` tiene que ser el alto real: es el eje vertical, sobre el que se aplican `rotacion_vertical` y `apilable`.

> **A tener en cuenta:**
> - El borrador proponía "join a `objeto` si `objeto_id` no es null, o campos propios si `nombre_personalizado`". **En la base, `solicitud_objeto` siempre guarda su propia copia** de peso, dimensiones y flags, también para ítems de catálogo. Al publicar la solicitud (RF-06), el backend copia los valores de `objeto` a la fila de `solicitud_objeto`. El planificador **lee una sola tabla**.
> - `objeto.largo_m/ancho_m/alto_m` (nuevo) son **nullable** porque las 5 filas semilla del catálogo todavía no tienen dimensiones. Hasta que el Administrador las cargue, esos objetos no se pueden copiar a una solicitud (el INSERT en `solicitud_objeto` fallaría por NOT NULL).
> - `vehiculo.peso_maximo_kg` **ya es carga útil** (confirmado por el humano, 2026-09-24). Se elimina la resta `PesoMaximoKg − PesoVehiculoKg` del borrador y **no** se usa `BoxSpec.EmptyWeight` (tara).
> - `solicitud_objeto.volumen_unitario_m3` y `vehiculo.volumen_carga_m3` son derivables de las dimensiones: quedaron nullable y DEPRECATED, y el backend no las escribe. `objeto.volumen_estimado_m3` está marcada como redundante. Plan de borrado: `ALGORITMO_COTIZACION.md` §8.

### 2.1 Flags de rotación y apilado → opciones de la librería

| `rotacion_horizontal` | `rotacion_vertical` | `ItemSpec` | Efecto |
|---|---|---|---|
| true | true | `Rotation: RotationBestFit` (default) | Cualquiera de las 6 orientaciones |
| true | **false** | `VerticalAxes: []Axis{DepthAxis}` | Queda parado y puede girar sobre el piso |
| **false** | **false** | `Rotation: RotationNever` | Sin rotación: el largo va a lo largo del vehículo |
| **false** | true | `Rotation: RotationBestFit` | **Informativo:** no hay opción directa en la librería. Se loguea (§7) |

| `apilable` | `ItemSpec` |
|---|---|
| true | (nada) |
| **false** | `NothingOnTop: true` |

---

## 3. Configuración de la librería (fija)

```go
import "github.com/bavix/boxpacker3/v2"

packer := boxpacker3.NewPacker(
    // Greedy de una pasada: ítems de mayor a menor volumen, cada uno en el primer viaje donde entre.
    boxpacker3.WithAlgorithm(boxpacker3.NewGreedy(boxpacker3.OrderDecreasing, boxpacker3.SelectFirstFit)),
    // Sin post-optimizaciones: la que viene por defecto (BalanceWeight) equilibra el peso
    // entre viajes, no reduce la cantidad y es muy cara (camión con 400 unidades: 62 s con ella, 2,4 s sin ella).
    boxpacker3.WithFinishers(),
    // Al menos el 60 % de la base de cada objeto tiene que estar apoyado.
    boxpacker3.WithRules(boxpacker3.Rules{MinSupportRatio: 0.6}),
)
```

| Decisión | Motivo |
|---|---|
| `NewGreedy(OrderDecreasing, SelectFirstFit)` | Rápido y con buen resultado (§6). La selección por defecto (`SelectFullestBox`) da los mismos viajes y tarda parecido; `SelectFirstFit` es más simple de explicar. |
| `WithFinishers()` vacío | La post-optimización por defecto no cambia la cantidad de viajes y en cargas grandes multiplica el tiempo por más de 20. |
| `MinSupportRatio: 0.6` | Evita estimaciones con objetos flotando. El valor es un criterio de razonabilidad, no una norma. Puede pasar a `config_operacion` si se quiere ajustar sin recompilar (hoy no es una columna). |
| **No** usar `NewSearch` ni `NewPortfolio` | Más lentos. En la prueba, `NewSearch(128, 3)` dio peor resultado que el greedy (5 viajes contra 4) y no respetó el límite de tiempo. |
| **No** usar `WithBudget` | No es confiable (§1). El límite de tiempo se pone con el `context.Context` del request: el greedy revisa el contexto por cada ítem y devuelve error si vence. Esto último no se midió. |

---

## 4. Planificación de viajes

```go
var ErrNoFactible = errors.New("carga no factible con este vehículo")

const (
    maxViajes    = 20  // salvaguarda: más viajes que esto → no se puede ofertar con este vehículo
    minApoyoBase = 0.6 // Rules.MinSupportRatio
)

// Viaje = un viaje físico dentro de la oferta (NO la tabla `viaje`).
// Item/Objeto/Vehiculo: ver ALGORITMO_COTIZACION.md §3.
func planificarViajes(ctx context.Context, v Vehiculo, carga []Item) ([]Viaje, error) {
    // 1. Cota rápida por peso y volumen: si ni idealmente entra en maxViajes, cortar ya.
    //    Sin esto la librería tarda en descubrirlo (1000 cajas en un utilitario: 16 s).
    pesoTotal, volumenTotal := sumarPesoYVolumen(carga)
    volumenUtil := v.LargoUtilM * v.AnchoUtilM * v.AltoUtilM
    if math.Ceil(pesoTotal/v.PesoUtilKg) > maxViajes || math.Ceil(volumenTotal/volumenUtil) > maxViajes {
        return nil, fmt.Errorf("%w: la carga necesita más de %d viajes", ErrNoFactible, maxViajes)
    }

    // 2. Ítems: una entrada por fila de solicitud_objeto, con su cantidad.
    items, err := armarItems(carga)
    if err != nil {
        return nil, err
    }

    // 3. El vehículo, disponible hasta maxViajes veces.
    vehiculo, err := boxpacker3.NewBoxFromSpec(boxpacker3.BoxSpec{
        ID:          v.Patente,
        OuterWidth:  v.LargoUtilM, // vehiculo.largo_util_m (nuevo)
        OuterHeight: v.AnchoUtilM, // vehiculo.ancho_util_m (nuevo)
        OuterDepth:  v.AltoUtilM,  // vehiculo.alto_util_m (nuevo) (eje vertical)
        MaxWeight:   v.PesoUtilKg, // vehiculo.peso_maximo_kg (ya es carga útil)
        Quantity:    maxViajes,
    })
    if err != nil {
        return nil, fmt.Errorf("vehículo %s: %w", v.Patente, err)
    }

    // 4. Empaquetar en una sola llamada.
    packer := boxpacker3.NewPacker(
        boxpacker3.WithAlgorithm(boxpacker3.NewGreedy(boxpacker3.OrderDecreasing, boxpacker3.SelectFirstFit)),
        boxpacker3.WithFinishers(),
        boxpacker3.WithRules(boxpacker3.Rules{MinSupportRatio: minApoyoBase}),
    )
    result, err := packer.Pack(ctx, []*boxpacker3.Box{vehiculo}, items)
    if err != nil {
        return nil, fmt.Errorf("empaquetar carga: %w", err) // incluye ctx vencido
    }

    // 5. Si algo quedó afuera, la oferta no es factible con este vehículo.
    if len(result.Unpacked) > 0 {
        return nil, noFactible(carga, result.Unpacked)
    }

    return dividirEnViajes(carga, result), nil
}

func armarItems(carga []Item) ([]*boxpacker3.Item, error) {
    items := make([]*boxpacker3.Item, 0, len(carga))
    for idx, it := range carga {
        rot, ejes := rotacionDe(it.Objeto) // tabla §2.1
        item, err := boxpacker3.NewItemFromSpec(boxpacker3.ItemSpec{
            ID:           strconv.Itoa(idx),  // índice de la fila en `carga`, para reagrupar después
            Width:        it.Objeto.LargoM,   // solicitud_objeto.largo_m (nuevo)
            Height:       it.Objeto.AnchoM,   // solicitud_objeto.ancho_m (nuevo)
            Depth:        it.Objeto.AltoM,    // solicitud_objeto.alto_m (nuevo) (eje vertical)
            Weight:       it.Objeto.PesoKg,   // solicitud_objeto.peso_unitario_kg
            Quantity:     it.Cantidad,        // solicitud_objeto.cantidad
            Rotation:     rot,
            VerticalAxes: ejes,
            NothingOnTop: !it.Objeto.Apilable, // solicitud_objeto.apilable (nuevo)
        })
        if err != nil {
            return nil, fmt.Errorf("ítem %q: %w", it.Objeto.Nombre, err)
        }
        items = append(items, item)
    }
    return items, nil
}

func rotacionDe(o Objeto) (boxpacker3.Rotation, []boxpacker3.Axis) {
    switch {
    case !o.RotacionVertical && !o.RotacionHorizontal:
        return boxpacker3.RotationNever, nil
    case !o.RotacionVertical:
        return boxpacker3.RotationBestFit, []boxpacker3.Axis{boxpacker3.DepthAxis}
    default: // incluye rotacion_horizontal=false sola → informativo (§2.1)
        return boxpacker3.RotationBestFit, nil
    }
}
```

Unidades: **metros y kg** en todo el flujo, igual que las columnas (`*_m`, `*_kg`). La librería no impone unidades, sólo necesita que todas las medidas usen la misma escala.

> **Cambios respecto del borrador:**
> - **Se elimina `expandirItems`:** la v2 recibe la cantidad por ítem (`ItemSpec.Quantity`).
> - **Se elimina la búsqueda iterativa de N (`n++` hasta `maxIntentos`):** la v2 abre vehículos a medida que los necesita dentro de una sola llamada. Esa búsqueda multiplicaba el costo hasta 15 veces en la v1.
> - **`maxViajes` reemplaza a `maxIntentos`** y cumple la misma función que en el borrador: si la carga necesita más de 20 viajes con ese vehículo, el Transportista no puede ofertar con él.
> - **Se agrega la cota rápida del paso 1** para no depender de la librería en cargas desproporcionadas.
> - `StrategyBestFitDecreasing` y `PackCtx` son de la v1 y no existen en la v2.

---

## 5. De vuelta a `Viaje` y qué pasa si no entra

```go
func dividirEnViajes(carga []Item, result *boxpacker3.Result) []Viaje {
    viajes := make([]Viaje, 0, len(result.Boxes)) // result.Boxes sólo trae vehículos usados
    for _, box := range result.Boxes {
        porFila := map[int]int{}
        for _, p := range box.Items {
            idx, _ := strconv.Atoi(p.Item.ID()) // ID = índice de la fila en `carga`
            porFila[idx]++
        }
        var v Viaje
        for idx := range carga {
            if n := porFila[idx]; n > 0 {
                v.Carga = append(v.Carga, Item{Objeto: carga[idx].Objeto, Cantidad: n})
            }
        }
        viajes = append(viajes, v)
    }
    return viajes
}

func noFactible(carga []Item, unpacked []boxpacker3.UnpackedItem) error {
    motivos := []string{}
    vistos := map[string]bool{}
    for _, u := range unpacked {
        idx, _ := strconv.Atoi(u.Item.ID())
        m := fmt.Sprintf("%s: %s", carga[idx].Objeto.Nombre, u.Reason)
        if u.Reason == boxpacker3.ReasonNoRoom {
            m = fmt.Sprintf("la carga necesita más de %d viajes", maxViajes)
        }
        if !vistos[m] {
            vistos[m] = true
            motivos = append(motivos, m)
        }
    }
    return fmt.Errorf("%w: %s", ErrNoFactible, strings.Join(motivos, "; "))
}
```

- `len(viajes)` es el valor de **`oferta.cantidad_viajes`** (existe, CHECK > 0) y se congela en `viaje.cantidad_viajes_snapshot` (nuevo) al aceptar la oferta.
- Las posiciones y orientaciones de `box.Items` **no se guardan** en la base: son una aproximación, no un plano de carga. Mostrar un plano queda fuera de alcance.
- **`ErrNoFactible`:** el endpoint `POST /solicitudes/{id}/ofertas` devuelve un error de validación con el motivo por objeto (`"Ropero: no permitted orientation fits"`, `"Sofá 2 cuerpos: too big"`) y no inserta la `oferta`. Significa "no es físicamente posible con este vehículo", no "sale muy caro". Los textos de `Reason` vienen en inglés desde la librería; traducirlos para la API es tarea del handler.

| `Reason` | Significado para el Transportista |
|---|---|
| `ReasonTooBig` | El objeto no entra en el vehículo en ninguna orientación |
| `ReasonTooHeavy` | El objeto solo supera la carga útil del vehículo |
| `ReasonNoOrientationFits` | Entraría girado, pero su restricción (`rotacion_*`) no lo permite. Ejemplo: un ropero de 1,90 m que va parado, en un furgón de 1,70 m de alto |
| `ReasonNoRoom` | Hacen falta más de `maxViajes` viajes |

---

## 6. Costo computacional (medido)

Pruebas del 2026-09-24 con la configuración de §3, en la máquina de desarrollo. Carga sintética: 80 % cajas de 0,5×0,4×0,4 m y 15 kg, 20 % muebles (sofá, heladera, cama, mesa, silla, lavarropas, ropero, TV). "Con restricciones" = muebles parados, sin nada encima de heladera, ropero, silla y TV, y 60 % de apoyo.

| Vehículo | Unidades | Restricciones | Viajes | Cota inferior | Tiempo |
|---|---|---|---|---|---|
| Furgón 3,0×1,7×1,7 m / 1200 kg | 100 | sí | 4 | 3 | 0,09 s |
| Furgón | 200 | sí | 8 | 5 | 0,4 s |
| Furgón | 400 | sí | 14 | 9 | 2,0 s |
| Camión 5,0×2,2×2,2 m / 3500 kg | 100 | no | 2 | 1 | 0,13 s |
| Camión | 400 | no | 5 | 4 | 2,5 s |
| Camión | 400 | sí | 7 | 4 | 1,4 s |

Algunas observaciones:
- En el furgón con restricciones, los roperos quedaron afuera por `ReasonNoOrientationFits` (1,90 m de alto parado contra 1,70 m del furgón) y se excluyen de la cota. Por eso esas filas tendrían `ErrNoFactible` en uso real. Se midieron igual para ver el tiempo.
- La v1.3.2 con la misma carga estimaba **7 viajes** para el furgón con 100 unidades y **10** para el camión con 400. La v2 estima 4 y 5, mucho más cerca de la cota. Como la cantidad de viajes multiplica el costo, esto hace el precio bastante más realista.
- Una **mudanza típica (50–100 unidades) tarda menos de 0,2 s.** Entre 200 y 400 unidades, de 0,4 a 2,5 s. El cálculo corre **una vez por oferta**, no por consulta.
- **Abierto (no decidido):** si conviene un **tope de unidades por solicitud** para acotar el peor caso. No se propone ningún valor.

---

## 7. RN-02 y `tipo_vehiculo`

> **Diferencia con la ERS:** la ERS describe RN-02 como un cálculo **por `tipo_vehiculo` al publicar la solicitud**. Este algoritmo corre **por `vehiculo` real, al ofertar** (decisión D-14, reflejada en `CLAUDE.md`, `TRAZABILIDAD.md` y `DOCUMENTACION_BASE_DE_DATOS.md`). `tipo_vehiculo` sólo tiene `volumen_estandar_m3` y `peso_maximo_estandar_kg`, **sin dimensiones**, así que no se le puede aplicar el empaquetado 3D. Si se mantiene algún chequeo interno por tipo, sólo puede usar la cota de peso y volumen del paso 1 de §4. No se proponen dimensiones para `tipo_vehiculo`: queda abierto.

---

## 8. Checklist de implementación

1. Agregar la dependencia **v2**: `go get github.com/bavix/boxpacker3/v2@v2.0.0` (pide Go ≥ 1.25; el repo usa 1.27). **No** usar el módulo sin `/v2`, que baja la v1.3.2 con otra API y sin restricciones de rotación ni apilado.
2. Leer los ítems **sólo de `solicitud_objeto`**: `cantidad`, `peso_unitario_kg`, `largo_m` (nuevo), `ancho_m` (nuevo), `alto_m` (nuevo), `rotacion_horizontal` (nuevo), `rotacion_vertical` (nuevo) y `apilable` (nuevo) (migración 0001, aplicada).
3. Leer la capacidad del vehículo de `vehiculo.peso_maximo_kg` (carga útil, sin tara) y `vehiculo.largo_util_m`, `ancho_util_m` y `alto_util_m` (nuevo) (migración 0002, aplicada).
4. Respetar el mapeo de ejes: `Width` = largo, `Height` = ancho, **`Depth` = alto (vertical)**. Invertirlo hace que `rotacion_vertical` y `apilable` se apliquen sobre el eje equivocado.
5. Usar exactamente la configuración de §3: greedy de una pasada, `WithFinishers()` vacío y `MinSupportRatio: 0.6`. No `NewSearch`, no `NewPortfolio`, no `WithBudget`.
6. Aplicar la cota rápida por peso y volumen **antes** de llamar a la librería.
7. Poner un límite de tiempo con el `context.Context` del request (`context.WithTimeout`) y devolver un error claro si vence. El valor del límite no está definido.
8. Loguear cuando la carga tenga `rotacion_horizontal = false` con `rotacion_vertical = true` (el único caso informativo), dejando escrito que esa restricción no se aplicó.
9. Loguear `result.Report.Bound.Boxes` junto a `len(viajes)`, para poder evaluar la calidad de la estimación en la memoria del TFG.
10. Definir `ErrNoFactible` como un error de dominio explícito que incluya el motivo por objeto, nunca un `panic` ni un `nil` silencioso.
11. El resultado de `planificarViajes` alimenta directamente a `calcularCostoOferta` (`ALGORITMO_COTIZACION.md` §4.4): no recalcular nada por separado.

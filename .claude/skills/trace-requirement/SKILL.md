---
name: trace-requirement
description: >-
  Verificar la trazabilidad entre un requisito de la ERS de Fletway (RF-01 a
  RF-24, RN-01 a RN-08, RNF, RI) y su implementación real en el backend Go antes
  de dar un endpoint por cerrado. Cruza la descripción/entradas/proceso/salida
  del requisito con el código, las policies RLS, los tests y los casos de prueba
  de qa/, y actualiza docs/TRAZABILIDAD.md. Usar antes de marcar cualquier RF/RN
  como terminado, en code review, o al auditar cobertura de la ERS.
---

# Skill: trace-requirement

## Cuándo usar

- Antes de marcar un RF/RN como ✅ en `docs/TRAZABILIDAD.md`.
- En la revisión de un PR que dice implementar un requisito.
- Para auditar qué % de la ERS está realmente cubierto.

## Insumos

- `docs/ERS_Fletway.pdf` — texto del requisito (4 campos: descripción, entradas,
  proceso, salida; o el enunciado de la RN).
- `docs/TRAZABILIDAD.md` — estado declarado.
- Código en `internal/feature/<ctx>/`.
- Policies RLS reales (MCP, solo lectura) para las tablas involucradas.
- `qa/casos-prueba/` y tests `*_test.go`.

## Procedimiento

### 1. Extraer el requisito

Copiar del PDF el texto exacto del RF/RN. Para un RF, listar sus **entradas**,
**proceso** y **salida** como ítems verificables. Para una RN, descomponer la
regla en condiciones atómicas (ej. RN-07: "sin cargo si no salió" + "con cargo
si salió" + "el transportista que cancela no paga penalización").

### 2. Localizar la implementación

- Endpoint(s): método + path en `internal/server` / `routes.go` de la feature.
- Handler → service → repository: seguir la cadena.
- ¿El handler está detrás del middleware `Auth`? (RNF-01 — si no, es un hallazgo).
- ¿El repository usa `db.WithinTx(ctx, identity, ...)` (RLS pass-through)? Si
  hace queries fuera de ese camino, es un hallazgo.

### 3. Cotejar campo por campo

Para cada entrada/proceso/salida del RF (o cada condición de la RN), marcar:

| Ítem del requisito | ¿Dónde en el código? | ¿Test que lo cubre? | Estado |
|--------------------|----------------------|---------------------|--------|

Estados por ítem: `implementado` / `parcial` / `ausente` / `contradice la ERS`.

### 4. Verificar las reglas de negocio ligadas

Si el RF referencia una RN (ej. RF-06 → RN-01, RN-02), verificar **también** esas
RN acá. Chequeos frecuentes:

- **RN-01 / RN-02:** el precio y la cantidad de viajes salen de un cálculo del
  sistema; el Cliente **no** los envía en el request. La tarifa viene de
  `config_tarifa` **vigente** (una sola fila sin `vigente_hasta`).
- **RN-03:** el `%` de comisión se toma de `config_comision` vigente y se
  **congela** en `viaje.porcentaje_comision_snapshot`.
- **RN-04:** el matching compara zona del transportista contra **origen o
  destino** de la solicitud, vía `transportista_zona`. Sin ensanchamiento de radio.
- **RN-05:** la notificación push es un **job async** (no bloquea el POST de la
  solicitud); el listado al Cliente devuelve **top 3 por score**.
- **RN-06:** dos PIN (inicio y fin) + registro de ubicación; la `resena` se
  habilita recién con PIN de fin válido.
- **RNF-03:** al confirmar un `viaje` se **escriben** todas las columnas
  `*_snapshot` (tarifa, vehículo, coordenadas, nombres).

### 5. Verificar RLS

Para cada tabla que toca el endpoint: leer sus policies reales (MCP) y confirmar
que un usuario del rol equivocado, o dueño de otro recurso, **no** puede
ejecutar la operación. Idealmente esto es un test.

### 6. Veredicto

- **CERRADO (✅):** todos los ítems `implementado`, RN ligadas verificadas, hay
  al menos un test por camino feliz + un test de RLS, y existe caso de prueba en
  `qa/` en estado ok.
- **NO CERRADO:** cualquier ítem `parcial` / `ausente` / `contradice la ERS`, o
  falta test de RLS. Listar los hallazgos concretos.

### 7. Actualizar `docs/TRAZABILIDAD.md`

Poner el estado real, los endpoints, las tablas y los tests. Si algo
**contradice la ERS**, no "arreglar" la ERS: registrar la discrepancia y
escalarla.

## Anti-patrones a detectar

- Endpoint de negocio sin middleware `Auth` (viola RNF-01).
- Query que no pasa por `WithinTx` con identidad → RLS no aplica.
- Precio/viajes recibidos del cliente en el body (viola RN-01/RN-02/RNF-04).
- Confirmar `viaje` sin escribir los `*_snapshot` (viola RNF-03).
- Notificación de RN-05 hecha sincrónica dentro del request (viola RNF-02).
- `resena` aceptada sin PIN de fin validado (viola RN-06).
- Marcar ✅ sin test de RLS.

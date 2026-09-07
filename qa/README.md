# qa/

Artefactos de prueba del backend que **no** son tests automatizados de Go
(esos van junto al código, en `*_test.go`).

```
qa/
├── postman/         Colecciones y entornos de Postman/Insomnia
└── casos-prueba/    Casos de prueba manuales / de aceptación (docs Word, planillas)
```

## postman/

- `fletway-backend.postman_collection.json` — colección principal (agregar cuando
  existan endpoints).
- `*.postman_environment.json` — entornos (`local`, `staging`). Los que tengan
  secretos van con sufijo `.local.json` y están gitignoreados.
- Cada request debería referenciar el RF/RN que ejercita en su descripción.

## casos-prueba/

- Casos de prueba y de aceptación en Word / planilla, uno por RF o por flujo.
- Convención de nombre: `CP-RF06-publicar-solicitud.docx`, `CP-RN01-cotizacion.xlsx`.
- Cada caso: precondición, pasos, datos, resultado esperado, RF/RN cubierto,
  estado (pendiente / ok / falla).

> La skill `trace-requirement` usa el estado de estos casos + la tabla de
> `docs/TRAZABILIDAD.md` para decidir si un requisito está realmente cerrado.

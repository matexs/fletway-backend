---
name: update-project-state
description: >-
  Actualizar de forma consistente los archivos de contexto persistente del repo
  fletway-backend (docs/ESTADO_PROYECTO.md, docs/DECISIONES_TECNICAS.md,
  docs/TRAZABILIDAD.md, docs/ENDPOINTS.md y CLAUDE.md) al cerrar un bloque de
  trabajo, de modo que la próxima sesión de Claude Code arranque con el contexto
  correcto sin releer todo. Usar al terminar una feature, tomar o confirmar una
  decisión de arquitectura, aplicar una migración, o antes de cerrar la sesión.
---

# Skill: update-project-state

## Cuándo usar

- Terminaste (o avanzaste sustancialmente) una feature / endpoint.
- Se tomó, confirmó o revirtió una decisión de arquitectura.
- Se aplicó una migración a Supabase.
- Cambió el estado de un bloqueo (ej. el MCP se conectó, se eligió pasarela).
- Antes de cerrar la sesión, si algo relevante cambió.

## Objetivo

Que estos cinco archivos sean, juntos, suficientes para que otra sesión retome el
trabajo sin releer la ERS ni el código entero:

| Archivo | Qué mantener al día |
|---------|---------------------|
| `docs/ESTADO_PROYECTO.md` | tabla "Qué está hecho", lista "Pendientes" ordenada, "Bloqueos", bitácora con fecha absoluta |
| `docs/DECISIONES_TECNICAS.md` | estado de cada decisión (CONFIRMADA / PROPUESTA A CONFIRMAR / ABIERTA); agregar decisiones nuevas con número D-NN |
| `docs/TRAZABILIDAD.md` | estado real de cada RF/RN (`NO`, `EN CURSO`, `OK`, `N/A`), endpoints, tablas, tests |
| `docs/ENDPOINTS.md` | contrato de cada endpoint nuevo/cambiado (lo consume la app Flutter) |
| `CLAUDE.md` | solo si cambió el rol del agente, una RN, o una convención — no es un diario |

## Procedimiento

### 1. Recolectar lo que cambió en esta sesión

- Commits hechos (`git log` desde el inicio de la sesión).
- Features tocadas, endpoints agregados.
- Migraciones aplicadas (nombre + número).
- Decisiones tomadas o confirmadas.
- Bloqueos que aparecieron o se resolvieron.

### 2. `docs/ESTADO_PROYECTO.md`

- Actualizar el "Resumen de una línea" si cambió el panorama.
- Mover ítems de "Pendientes" a "Qué está hecho" (con estado real, sin inflar).
- Reordenar "Pendientes" por prioridad actual.
- Actualizar "Bloqueos".
- **Agregar una fila a la bitácora** con fecha absoluta (hoy) y el hito.
- Actualizar el campo "Última actualización" y "Etapa".

### 3. `docs/DECISIONES_TECNICAS.md`

- ¿Se confirmó una decisión que estaba "PROPUESTA A CONFIRMAR"? Cambiar el estado
  en el índice y en su sección, y anotar quién/cuándo.
- ¿Decisión nueva? Agregar `D-NN` al índice y una sección con: decisión,
  alternativas descartadas, reversibilidad, implicancias.
- ¿Se revirtió algo? No borrar: marcar "REVERTIDA" y por qué.

### 4. `docs/TRAZABILIDAD.md`

- Para cada RF/RN tocado: estado real, endpoint(s), tablas, tests. Usar la skill
  `trace-requirement` para decidir `OK` vs `EN CURSO` — **no** marcar `OK` a ojo.
- Actualizar "Última actualización".

### 5. `docs/ENDPOINTS.md`

- Documentar endpoints nuevos/cambiados: método, path, auth, request, response,
  errores, RF/RN. Mover de "planificados" a "implementados".
- Subir la "Versión de contrato" si hubo un cambio incompatible.
- Avisar en el resumen de la sesión que el repo `fletway-mobile` debería correr
  su skill `sync-api-models`.

### 6. `CLAUDE.md` (solo si aplica)

Tocar únicamente si cambió el rol del agente, una regla de negocio crítica, una
convención de commits/branches, o la arquitectura de alto nivel. Si no, dejarlo.

### 7. Verificación final

- [ ] Fechas en formato absoluto (no "hoy", "ayer").
- [ ] "Última actualización" al día en los archivos tocados.
- [ ] Ninguna decisión no reversible quedó como cerrada sin confirmación humana.
- [ ] `docs/TRAZABILIDAD.md` no tiene `OK` sin test de RLS.
- [ ] El resumen de sesión menciona si `fletway-mobile` debe resincronizar modelos.

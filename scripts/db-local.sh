#!/usr/bin/env bash
# Base de datos local de Fletway (D-16): Supabase CLI sobre Docker.
#
# Deja la base local con el mismo esquema que dbFletway:
#   1. genera la clave ES256 de firma de JWT si no existe (supabase/signing_keys.json,
#      fuera de git), para que el Auth local firme igual que producción (D-04);
#   2. levanta el stack local si no está corriendo;
#   3. resetea la base (borra todo);
#   4. aplica el esquema base (migrations/baseline/) y después, en orden, cada
#      migrations/NNNN_*.sql posterior a las que ya incluye el esquema base.
#
# Uso:  scripts/db-local.sh          (reset completo)
#       scripts/db-local.sh status   (muestra URLs y claves locales)
#
# Nunca toca dbFletway: sólo trabaja contra el contenedor local.
set -euo pipefail

cd "$(dirname "$0")/.."

# Servicios que esta etapa no usa (D-17): se dejan afuera para arrancar más rápido.
EXCLUIR="edge-runtime,imgproxy,logflare,vector"
# Última migración numerada que ya está incluida en el esquema base.
BASELINE_HASTA=7
CONTENEDOR_DB="supabase_db_fletway-backend"

if [[ "${1:-}" == "status" ]]; then
  supabase status
  exit 0
fi

if [[ ! -s supabase/signing_keys.json ]]; then
  echo "> generando clave ES256 local"
  echo '[]' > supabase/signing_keys.json
  supabase gen signing-key --algorithm ES256 --append
fi

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTENEDOR_DB"; then
  echo "> levantando Supabase local"
  supabase start -x "$EXCLUIR"
fi

echo "> reseteando la base local"
supabase db reset --local

psql_local() {
  docker exec -i "$CONTENEDOR_DB" psql -v ON_ERROR_STOP=1 -q -U postgres -d postgres "$@"
}

shopt -s nullglob
baseline=(migrations/baseline/*.sql)
if (( ${#baseline[@]} == 0 )); then
  echo "error: falta el esquema base en migrations/baseline/ (ver migrations/README.md)" >&2
  exit 1
fi
for f in "${baseline[@]}"; do
  echo "> aplicando esquema base: $f"
  psql_local < "$f"
done

for f in migrations/[0-9][0-9][0-9][0-9]_*.sql; do
  n=$((10#$(basename "$f" | cut -c1-4)))
  if (( n > BASELINE_HASTA )); then
    echo "> aplicando $f"
    psql_local < "$f"
  fi
done

echo "> listo"
supabase status

module github.com/fletway/fletway-backend

go 1.27

// Dependencias previstas (agregar con `go mod tidy` al implementar):
//   github.com/jackc/pgx/v5            -- driver + pool Postgres (D-02)
//   github.com/golang-jwt/jwt/v5       -- verificación de JWT de Supabase (D-04)
//   github.com/MicahParks/keyfunc/v3   -- cache de JWKS para el JWT asimétrico (D-04)
//   github.com/stretchr/testify        -- asserts en tests (D-12)
//   github.com/jackc/pgx/v5/pgxpool    -- (incluido en pgx/v5)

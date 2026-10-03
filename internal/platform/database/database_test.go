package database_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/config"
	"github.com/matexs/fletway-backend/internal/platform/database"
)

// abrirBaseDePrueba abre la base local de Supabase (D-16). Si TEST_DATABASE_URL
// no está definida, el test se saltea: nunca se usa dbFletway (D-12).
func abrirBaseDePrueba(t *testing.T) *database.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida: test de integración salteado (requiere Supabase local)")
	}
	db, err := database.Open(context.Background(), config.Config{Database: config.DatabaseConfig{URL: url, MaxConns: 2, MinConns: 0}})
	require.NoError(t, err)
	t.Cleanup(db.Close)
	return db
}

func identidad(t *testing.T, sub string) database.Identity {
	t.Helper()
	raw, err := json.Marshal(map[string]string{"sub": sub, "role": "authenticated"})
	require.NoError(t, err)
	return database.Identity{UserID: sub, Role: "authenticated", RawClaims: string(raw)}
}

func TestOpenSinURL(t *testing.T) {
	t.Parallel()
	_, err := database.Open(context.Background(), config.Config{})
	require.ErrorIs(t, err, database.ErrSinURL)
}

func TestWithinTxAplicaRolYClaims(t *testing.T) {
	db := abrirBaseDePrueba(t)
	const sub = "8f1d2c3b-0000-4000-8000-000000000003"

	var rol, uid string
	err := db.WithinTx(context.Background(), identidad(t, sub), func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT current_user, auth.uid()::text").Scan(&rol, &uid)
	})
	require.NoError(t, err)
	assert.Equal(t, "authenticated", rol)
	assert.Equal(t, sub, uid)
}

func TestWithinTxNoFiltraFueraDeLaTransaccion(t *testing.T) {
	db := abrirBaseDePrueba(t)
	require.NoError(t, db.WithinTx(context.Background(), identidad(t, "8f1d2c3b-0000-4000-8000-000000000004"), func(pgx.Tx) error { return nil }))

	// Otra transacción con otro usuario no ve los claims de la anterior.
	var uid string
	err := db.WithinTx(context.Background(), identidad(t, "8f1d2c3b-0000-4000-8000-000000000005"), func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT auth.uid()::text").Scan(&uid)
	})
	require.NoError(t, err)
	assert.Equal(t, "8f1d2c3b-0000-4000-8000-000000000005", uid)
}

func TestWithinTxRevierteAnteError(t *testing.T) {
	db := abrirBaseDePrueba(t)
	errDeNegocio := errors.New("falla de negocio")
	err := db.WithinTx(context.Background(), identidad(t, "8f1d2c3b-0000-4000-8000-000000000006"), func(pgx.Tx) error { return errDeNegocio })
	require.ErrorIs(t, err, errDeNegocio)
}

func TestWithinTxSinIdentidad(t *testing.T) {
	t.Parallel()
	// No hace falta base: la validación es previa a abrir la transacción.
	var db *database.DB
	err := db.WithinTx(context.Background(), database.Identity{}, func(pgx.Tx) error { return nil })
	require.ErrorIs(t, err, database.ErrSinIdentidad)
}

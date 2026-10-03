// Package decimal representa montos y medidas como decimales exactos en la API
// (D-11): se reciben y se devuelven como números JSON y se guardan en columnas
// numeric de Postgres, sin pasar por float64.
package decimal

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// ErrFormato indica que el valor no es un número decimal válido.
var ErrFormato = errors.New("decimal: formato inválido")

// Decimal es un número decimal exacto. El valor cero es 0.
type Decimal struct {
	r big.Rat
}

// Parse convierte s ("1234.5", "-3", "0.01") en Decimal. No acepta notación
// exponencial. Devuelve ErrFormato si s no es un decimal.
func Parse(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "eE/") {
		return Decimal{}, ErrFormato
	}
	var d Decimal
	if _, ok := d.r.SetString(s); !ok {
		return Decimal{}, ErrFormato
	}
	return d, nil
}

// MustParse es Parse para constantes en código y tests; entra en pánico si s
// no es válido.
func MustParse(s string) Decimal {
	d, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("decimal.MustParse(%q): %v", s, err))
	}
	return d
}

// Escala devuelve la cantidad mínima de decimales necesaria para escribir d
// exacto (por ejemplo 2 para 10.25); -1 si d no tiene representación decimal
// finita.
func (d Decimal) Escala() int {
	den := new(big.Int).Set(d.r.Denom())
	dos, cinco := big.NewInt(2), big.NewInt(5)
	n2, n5 := 0, 0
	for new(big.Int).Mod(den, dos).Sign() == 0 {
		den.Div(den, dos)
		n2++
	}
	for new(big.Int).Mod(den, cinco).Sign() == 0 {
		den.Div(den, cinco)
		n5++
	}
	if den.Cmp(big.NewInt(1)) != 0 {
		return -1
	}
	return max(n2, n5)
}

// Cmp compara d con o: -1, 0 o 1.
func (d Decimal) Cmp(o Decimal) int { return d.r.Cmp(&o.r) }

// Sign devuelve -1, 0 o 1 según el signo de d.
func (d Decimal) Sign() int { return d.r.Sign() }

// String devuelve d con la cantidad justa de decimales ("10.25", "3").
func (d Decimal) String() string {
	e := d.Escala()
	if e < 0 {
		e = 10
	}
	return d.r.FloatString(e)
}

// MarshalJSON escribe d como número JSON.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalJSON lee un número JSON (no acepta strings ni null).
func (d *Decimal) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || b[0] == '"' || bytes.Equal(b, []byte("null")) {
		return ErrFormato
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return ErrFormato
	}
	v, err := Parse(n.String())
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// Scan implementa sql.Scanner para leer columnas numeric. La consulta tiene que
// castear la columna a texto (col::text) para no pasar por float64.
func (d *Decimal) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("%w: no se puede leer %T", ErrFormato, src)
	}
	v, err := Parse(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// Value implementa driver.Valuer: envía d como texto; la consulta lo castea a
// numeric ($1::numeric).
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

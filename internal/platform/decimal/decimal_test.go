package decimal_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
)

func TestParseYString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    string
		escala  int
		wantErr bool
	}{
		{"10.25", "10.25", 2, false},
		{"3", "3", 0, false},
		{"0.10", "0.1", 1, false},
		{"-1.5", "-1.5", 1, false},
		{"1234567890.12", "1234567890.12", 2, false},
		{"", "", 0, true},
		{"1e3", "", 0, true},
		{"1/3", "", 0, true},
		{"abc", "", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			d, err := decimal.Parse(tc.in)
			if tc.wantErr {
				require.ErrorIs(t, err, decimal.ErrFormato)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, d.String())
			assert.Equal(t, tc.escala, d.Escala())
		})
	}
}

func TestJSON(t *testing.T) {
	t.Parallel()
	var v struct {
		Monto decimal.Decimal `json:"monto"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"monto": 1500.75}`), &v))
	assert.Equal(t, "1500.75", v.Monto.String())

	out, err := json.Marshal(v)
	require.NoError(t, err)
	assert.JSONEq(t, `{"monto": 1500.75}`, string(out))

	for _, malo := range []string{`{"monto": "1500"}`, `{"monto": null}`, `{"monto": true}`} {
		assert.Error(t, json.Unmarshal([]byte(malo), &v), malo)
	}
}

func TestCmp(t *testing.T) {
	t.Parallel()
	assert.Equal(t, -1, decimal.MustParse("1.5").Cmp(decimal.MustParse("2")))
	assert.Equal(t, 0, decimal.MustParse("2.00").Cmp(decimal.MustParse("2")))
	assert.Equal(t, 1, decimal.MustParse("0.01").Sign())
}

func TestScanYValue(t *testing.T) {
	t.Parallel()
	var d decimal.Decimal
	require.NoError(t, d.Scan("2.50"))
	assert.Equal(t, "2.5", d.String())
	require.NoError(t, d.Scan([]byte("7")))
	v, err := d.Value()
	require.NoError(t, err)
	assert.Equal(t, "7", v)
	assert.Error(t, d.Scan(3.5), "un float no se acepta: perdería precisión")
}

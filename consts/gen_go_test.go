package consts

import "testing"

// TestGoFloatConst verifies float const rendering: float64 stays an UNTYPED
// float literal (whole values keep ".0" so Go never reads them as untyped
// ints), while float32 stays typed.
func TestGoFloatConst(t *testing.T) {
	cases := []struct {
		typeName string
		value    float64
		want     string
	}{
		{"float64", 1.0, "1.0"},
		{"float64", 1.5, "1.5"},
		{"float64", 1e-9, "1e-09"},
		{"double", 42.0, "42.0"},
		{"float32", 1.0, "float32(1)"},
		{"float32", 3.14, "float32(3.14)"},
	}
	for _, tc := range cases {
		val := &Value{Float: &tc.value}
		got := goConstValue(tc.typeName, val)
		if got != tc.want {
			t.Errorf("goConstValue(%q, %v) = %q, want %q", tc.typeName, tc.value, got, tc.want)
		}
	}
}

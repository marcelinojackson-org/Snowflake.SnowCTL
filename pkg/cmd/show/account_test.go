package showcmd

import "testing"

func TestQuoteLiteralEscapes(t *testing.T) {
    got := quoteLiteral("O'Reilly")
    want := "'O''Reilly'"
    if got != want {
        t.Fatalf("quoteLiteral mismatch: got %s want %s", got, want)
    }
}

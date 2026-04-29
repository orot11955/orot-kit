package secret

import "testing"

func TestUUIDShape(t *testing.T) {
	value, err := UUID()
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 36 {
		t.Fatalf("UUID length = %d, want 36: %s", len(value), value)
	}
	if value[14] != '4' {
		t.Fatalf("UUID version = %q, want 4: %s", value[14], value)
	}
}

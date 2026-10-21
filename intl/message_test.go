package intl

import "testing"

func TestIsolate(t *testing.T) {
	for _, tc := range []struct{ dir, s, want string }{
		{"ltr", "Ada", "Ada"},
		{"ltr", "سارة", "⁨سارة⁩"},
		{"rtl", "سارة", "سارة"},
		{"rtl", "Ada", "⁨Ada⁩"},
		{"ltr", "123", "⁨123⁩"},
	} {
		if got := isolate(tc.dir, tc.s); got != tc.want {
			t.Errorf("isolate(%s, %q) = %q, want %q", tc.dir, tc.s, got, tc.want)
		}
	}
}

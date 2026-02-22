package api

import "testing"

func TestCheckCompat(t *testing.T) {
	tests := []struct {
		name        string
		cliVersion  string
		minRequired string
		wantEmpty   bool
	}{
		{"equal versions", "0.2.0", "0.2.0", true},
		{"cli newer than min", "0.3.0", "0.2.0", true},
		{"cli older than min", "0.1.4", "0.2.0", false},
		{"major version ahead", "1.0.0", "0.9.0", true},
		{"major version behind", "0.9.0", "1.0.0", false},
		{"patch difference ok", "0.2.1", "0.2.0", true},
		{"patch behind", "0.2.0", "0.2.1", false},
		{"dev build skips check", "dev", "0.2.0", true},
		{"empty cli version", "", "0.2.0", true},
		{"empty min required", "0.2.0", "", true},
		{"both empty", "", "", true},
		{"v prefix handled", "v0.2.0", "0.2.0", true},
		{"v prefix on min", "0.2.0", "v0.2.0", true},
		{"pre-release stripped", "0.2.0-beta", "0.2.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckCompat(tt.cliVersion, tt.minRequired)
			if tt.wantEmpty && got != "" {
				t.Errorf("CheckCompat(%q, %q) = %q, want empty", tt.cliVersion, tt.minRequired, got)
			}
			if !tt.wantEmpty && got == "" {
				t.Errorf("CheckCompat(%q, %q) = empty, want warning", tt.cliVersion, tt.minRequired)
			}
		})
	}
}

func TestCheckCompat_WarningMessage(t *testing.T) {
	msg := CheckCompat("0.1.4", "0.2.0")
	if msg == "" {
		t.Fatal("expected warning message")
	}
	// Verify the message contains useful info
	if !contains(msg, "0.1.4") || !contains(msg, "0.2.0") || !contains(msg, "stompy update") {
		t.Errorf("warning message missing key info: %q", msg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
		ok    bool
	}{
		{"1.2.3", [3]int{1, 2, 3}, true},
		{"0.1.4", [3]int{0, 1, 4}, true},
		{"v1.0.0", [3]int{1, 0, 0}, true},
		{"1.2.3-beta", [3]int{1, 2, 3}, true},
		{"1.2", [3]int{}, false},
		{"abc", [3]int{}, false},
		{"1.2.abc", [3]int{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := parseSemver(tt.input)
			if ok != tt.ok {
				t.Errorf("parseSemver(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("parseSemver(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b [3]int
		want int
	}{
		{[3]int{1, 0, 0}, [3]int{1, 0, 0}, 0},
		{[3]int{1, 0, 0}, [3]int{0, 9, 9}, 1},
		{[3]int{0, 1, 0}, [3]int{0, 2, 0}, -1},
		{[3]int{0, 2, 0}, [3]int{0, 2, 1}, -1},
		{[3]int{0, 2, 1}, [3]int{0, 2, 0}, 1},
	}

	for _, tt := range tests {
		got := compareSemver(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareSemver(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

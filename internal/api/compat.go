package api

import (
	"fmt"
	"strconv"
	"strings"
)

// CheckCompat compares the CLI version against the server's minimum required version.
// Returns a warning message if the CLI is outdated, or "" if compatible.
// Returns "" for "dev" builds or empty versions (no comparison possible).
func CheckCompat(cliVersion, minRequired string) string {
	if cliVersion == "" || cliVersion == "dev" || minRequired == "" {
		return ""
	}

	cliParts, ok := parseSemver(cliVersion)
	if !ok {
		return ""
	}
	minParts, ok := parseSemver(minRequired)
	if !ok {
		return ""
	}

	if compareSemver(cliParts, minParts) < 0 {
		return fmt.Sprintf("Warning: stompy-cli %s is below minimum supported version %s. Run 'stompy update' to upgrade.", cliVersion, minRequired)
	}
	return ""
}

// parseSemver extracts [major, minor, patch] from a version string like "1.2.3".
// Returns false if the format is invalid.
func parseSemver(v string) ([3]int, bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var result [3]int
	for i, p := range parts {
		// Strip any pre-release suffix (e.g., "1-beta" -> "1")
		if idx := strings.IndexAny(p, "-+"); idx >= 0 {
			p = p[:idx]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		result[i] = n
	}
	return result, true
}

// compareSemver returns -1 if a < b, 0 if equal, 1 if a > b.
func compareSemver(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

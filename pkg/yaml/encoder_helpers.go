package yaml

import "strconv"

// appendEscapedYAMLString appends a YAML-escaped string to buf (without surrounding quotes).
// Zero-allocation: writes directly to provided buffer.
func appendEscapedYAMLString(buf []byte, s string) []byte {
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		var esc byte
		switch c {
		case '"':
			esc = '"'
		case '\\':
			esc = '\\'
		case '\n':
			esc = 'n'
		case '\r':
			esc = 'r'
		case '\t':
			esc = 't'
		default:
			continue
		}
		buf = append(buf, s[start:i]...)
		buf = append(buf, '\\', esc)
		start = i + 1
	}
	buf = append(buf, s[start:]...)
	return buf
}

// needsQuotingFast checks if a YAML string needs quoting, operating on the string directly.
// This is the zero-alloc version of needsQuoting.
func needsQuotingFast(s string) bool {
	if len(s) == 0 {
		return true
	}

	// Special YAML values
	switch s {
	case "true", "false", "yes", "no", "null", "~":
		return true
	}

	// Looks like a number
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}

	// Check for characters that require quoting
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case ':', '#', '@', '`', '"', '\'', '{', '}', '[', ']', '|', '>', '-':
			return true
		}
	}

	// Starts with special characters
	if s[0] == ' ' || s[0] == '-' || s[0] == '?' {
		return true
	}

	return false
}

// sortYAMLStrings sorts a string slice in-place using insertion sort.
// For the small key counts typical in YAML maps (< 20 keys) this is
// faster than sort.Strings because it avoids the interface overhead.
func sortYAMLStrings(s []string) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

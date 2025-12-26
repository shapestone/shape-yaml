// Package yaml provides YAML format validation utilities.
// This package has zero external dependencies and can be used in any Go project.
package yaml

import (
	"fmt"
	"strings"
)

// Validate validates YAML content using line-based parsing and indentation rules.
// Returns nil if valid, error with description if invalid.
//
// Supports:
//   - Key-value pairs: key: value
//   - Lists: - item
//   - Comments: # comment
//   - Scalars (strings, numbers, booleans)
//   - Quoted strings (single and double quotes)
//
// Validates:
//   - No tab characters (YAML requires spaces for indentation)
//   - Proper quote matching
//   - Basic syntax correctness
//
// Limitations:
//   - Does not support complex nested structures
//   - Basic validation only, not full YAML spec compliance
func Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return nil // Empty YAML is valid
	}

	// Validate indentation and basic syntax
	if err := validateSyntax(content); err != nil {
		return err
	}

	return nil
}

// validateSyntax performs line-by-line syntax validation
func validateSyntax(content string) error {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for tabs (YAML doesn't allow tabs for indentation)
		if strings.Contains(line, "\t") {
			return fmt.Errorf("invalid YAML: tab character found at line %d (YAML requires spaces for indentation)", lineNum)
		}

		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for unclosed quotes
		if err := validateQuotes(trimmed, lineNum); err != nil {
			return err
		}

		// Validate line structure
		if err := validateLine(trimmed, lineNum); err != nil {
			return err
		}
	}

	return nil
}

// validateQuotes checks for properly matched quotes
func validateQuotes(line string, lineNum int) error {
	var inSingleQuote, inDoubleQuote bool
	var escaped bool

	for i, ch := range line {
		if escaped {
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			if inSingleQuote || inDoubleQuote {
				escaped = true
			}
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		case '#':
			// Comments outside quotes
			if !inSingleQuote && !inDoubleQuote {
				// Rest of line is comment
				break
			}
		}

		// If we're at the end and still in quotes, that's an error
		if i == len(line)-1 {
			if inSingleQuote {
				return fmt.Errorf("invalid YAML: unclosed single quote at line %d", lineNum)
			}
			if inDoubleQuote {
				return fmt.Errorf("invalid YAML: unclosed double quote at line %d", lineNum)
			}
		}
	}

	return nil
}

// validateLine validates the structure of a non-comment, non-empty line
func validateLine(line string, lineNum int) error {
	// Remove inline comments (outside quotes)
	line = removeInlineComments(line)

	// Check for list item
	if strings.HasPrefix(line, "-") {
		// Valid list item format: "- value" or just "-"
		rest := strings.TrimPrefix(line, "-")
		if len(rest) > 0 && rest[0] != ' ' && rest[0] != '\t' {
			// Might be a continuation like "---" (document separator)
			if strings.TrimSpace(line) == "---" || strings.TrimSpace(line) == "..." {
				return nil // Valid YAML document separator
			}
			return fmt.Errorf("invalid YAML: list item must be followed by space at line %d", lineNum)
		}
		return nil
	}

	// Check for key-value pair
	if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])

			// Key should not be empty
			if key == "" {
				return fmt.Errorf("invalid YAML: empty key at line %d", lineNum)
			}

			// Key should not contain certain special characters (basic validation)
			if strings.ContainsAny(key, "{}[]|>") {
				// These might be valid in quoted keys, but we'll be lenient
			}

			return nil
		}
	}

	// If line doesn't match any pattern, it might be a scalar value
	// which is valid in certain contexts (like after a key with no value)
	// We'll allow it for leniency
	return nil
}

// removeInlineComments removes comments from a line (preserving quotes)
func removeInlineComments(line string) string {
	var result strings.Builder
	var inSingleQuote, inDoubleQuote bool
	var escaped bool

	for _, ch := range line {
		if escaped {
			result.WriteRune(ch)
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			if inSingleQuote || inDoubleQuote {
				escaped = true
			}
			result.WriteRune(ch)
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
			result.WriteRune(ch)
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
			result.WriteRune(ch)
		case '#':
			if !inSingleQuote && !inDoubleQuote {
				// Comment starts here, return what we have
				return result.String()
			}
			result.WriteRune(ch)
		default:
			result.WriteRune(ch)
		}
	}

	return result.String()
}

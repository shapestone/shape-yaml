package tokenizer

import (
	"github.com/shapestone/shape-core/pkg/tokenizer"
)

// NewTokenizer creates a tokenizer for YAML format (MVP subset).
// The tokenizer matches YAML tokens in order of specificity.
//
// Ordering is critical:
// 1. Document markers (before dash)
// 2. Merge key (before colon)
// 3. Keywords (true, false, null) before plain strings
// 4. Structural tokens
// 5. Flow style tokens
// 6. Block scalars
// 7. Anchors, aliases, tags
// 8. Comments
// 9. Quoted strings
// 10. Numbers
// 11. Plain strings (last, matches anything else)
// 12. Newlines
func NewTokenizer() tokenizer.Tokenizer {
	return tokenizer.NewTokenizerWithoutWhitespace(
		// Custom whitespace that doesn't consume newlines
		YAMLWhitespaceMatcher(),
		// Document markers (before dash)
		tokenizer.StringMatcherFunc(TokenDocSep, "---"),
		tokenizer.StringMatcherFunc(TokenDocEnd, "..."),

		// Merge key (before colon)
		tokenizer.StringMatcherFunc(TokenMergeKey, "<<"),

		// Keywords (before plain strings)
		// Case-insensitive booleans (true/True/TRUE, yes/Yes/YES, on/On/ON, etc.)
		BooleanMatcher(),
		tokenizer.StringMatcherFunc(TokenNull, "null"),
		tokenizer.CharMatcherFunc(TokenNull, '~'),

		// Numbers (before dash, so -17 matches as number not dash+17)
		NumberMatcher(),

		// Structural tokens
		tokenizer.StringMatcherFunc(TokenColon, ":"),
		tokenizer.StringMatcherFunc(TokenDash, "-"),
		tokenizer.StringMatcherFunc(TokenComma, ","),
		tokenizer.StringMatcherFunc(TokenQuestion, "?"),

		// Flow style tokens
		tokenizer.StringMatcherFunc(TokenLBrace, "{"),
		tokenizer.StringMatcherFunc(TokenRBrace, "}"),
		tokenizer.StringMatcherFunc(TokenLBracket, "["),
		tokenizer.StringMatcherFunc(TokenRBracket, "]"),

		// Block scalars
		tokenizer.StringMatcherFunc(TokenBlockLiteral, "|"),
		tokenizer.StringMatcherFunc(TokenBlockFolded, ">"),

		// Anchors and aliases
		AnchorMatcher(),
		AliasMatcher(),

		// Tags
		TagMatcher(),

		// Directives (before comments, as % could appear in content)
		DirectiveMatcher(),

		// Comments (captured for potential filtering)
		CommentMatcher(),

		// Quoted strings
		DoubleQuotedStringMatcher(),
		SingleQuotedStringMatcher(),

		// Plain strings (last, matches anything else)
		PlainStringMatcher(),

		// Newline
		NewlineMatcher(),
	)
}

// NewTokenizerWithStream creates a tokenizer for YAML format using a pre-configured stream.
// This is used internally to support streaming from io.Reader.
func NewTokenizerWithStream(stream tokenizer.Stream) tokenizer.Tokenizer {
	tok := NewTokenizer()
	tok.InitializeFromStream(stream)
	return tok
}

// DoubleQuotedStringMatcher creates a matcher for YAML double-quoted strings.
// Matches: "..." with escape sequences \", \\, \n, \t, \r, \uXXXX
//
// Grammar:
//
//	String = '"' { Character } '"' ;
//	Character = UnescapedChar | EscapeSequence ;
//	EscapeSequence = "\\" ( '"' | "\\" | "n" | "t" | "r" | "0" | UnicodeEscape ) ;
//	UnicodeEscape = "u" HexDigit HexDigit HexDigit HexDigit ;
//
// Performance: Uses ByteStream for fast ASCII scanning with SWAR acceleration.
func DoubleQuotedStringMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Try ByteStream fast path for ASCII strings
		if byteStream, ok := stream.(tokenizer.ByteStream); ok {
			return doubleQuotedStringMatcherByte(byteStream)
		}

		// Fallback to rune-based matcher for non-ByteStream
		return doubleQuotedStringMatcherRune(stream)
	}
}

// doubleQuotedStringMatcherByte uses ByteStream + SWAR for optimal performance.
func doubleQuotedStringMatcherByte(stream tokenizer.ByteStream) *tokenizer.Token {
	// Opening quote
	b, ok := stream.PeekByte()
	if !ok {
		return nil
	}
	if b != '"' {
		return nil
	}

	startPos := stream.BytePosition()
	stream.NextByte() // consume opening quote

	// Use SWAR to find closing quote or escape quickly
	for {
		// Find next quote or backslash using SWAR
		offset := tokenizer.FindEscapeOrQuote(stream.RemainingBytes())

		if offset == -1 {
			// No quote or escape found - unterminated string
			return nil
		}

		// Advance to the found position
		for i := 0; i < offset; i++ {
			b, ok := stream.NextByte()
			if !ok {
				return nil
			}
			// Check for control characters (except tab which is allowed)
			if b < 0x20 && b != '\t' {
				return nil
			}
		}

		// Now at quote or backslash
		b, ok := stream.NextByte()
		if !ok {
			return nil
		}

		if b == '"' {
			// Found closing quote - extract string
			value := stream.SliceFrom(startPos)

			// Convert to runes for token (maintains compatibility)
			return tokenizer.NewToken(TokenString, []rune(string(value)))
		}

		if b == '\\' {
			// Escape sequence - consume next character
			escaped, ok := stream.NextByte()
			if !ok {
				return nil
			}

			// Validate escape sequence
			switch escaped {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't', '0':
				// Valid single-char escape
			case 'a', 'v', 'e', ' ', 'N', '_', 'L', 'P':
				// Advanced YAML 1.2 escape sequences
				// \a=bell, \v=vtab, \e=escape, \ =space, \N=NEL, \_=nbsp, \L=line separator, \P=paragraph separator
			case 'u':
				// Unicode escape - consume 4 hex digits
				for i := 0; i < 4; i++ {
					hex, ok := stream.NextByte()
					if !ok {
						return nil
					}
					if !isHexDigitByte(hex) {
						return nil
					}
				}
			case 'U':
				// 8-digit Unicode escape - consume 8 hex digits
				for i := 0; i < 8; i++ {
					hex, ok := stream.NextByte()
					if !ok {
						return nil
					}
					if !isHexDigitByte(hex) {
						return nil
					}
				}
			default:
				// Invalid escape sequence
				return nil
			}
		}
	}
}

// doubleQuotedStringMatcherRune is the fallback rune-based implementation.
func doubleQuotedStringMatcherRune(stream tokenizer.Stream) *tokenizer.Token {
	var value []rune

	// Opening quote
	r, ok := stream.NextChar()
	if !ok || r != '"' {
		return nil
	}
	value = append(value, r)

	// Characters until closing quote
	for {
		r, ok := stream.NextChar()
		if !ok {
			return nil
		}

		value = append(value, r)

		if r == '"' {
			return tokenizer.NewToken(TokenString, value)
		}

		if r == '\\' {
			r, ok := stream.NextChar()
			if !ok {
				return nil
			}
			value = append(value, r)

			// Validate escape sequence
			switch r {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't', '0':
				// Valid single-char escape
			case 'a', 'v', 'e', ' ', 'N', '_', 'L', 'P':
				// Advanced YAML 1.2 escape sequences
				// \a=bell, \v=vtab, \e=escape, \ =space, \N=NEL, \_=nbsp, \L=line separator, \P=paragraph separator
			case 'u':
				// Unicode escape - consume 4 hex digits
				for i := 0; i < 4; i++ {
					r, ok := stream.NextChar()
					if !ok {
						return nil
					}
					if !isHexDigit(r) {
						return nil
					}
					value = append(value, r)
				}
			case 'U':
				// 8-digit Unicode escape - consume 8 hex digits
				for i := 0; i < 8; i++ {
					r, ok := stream.NextChar()
					if !ok {
						return nil
					}
					if !isHexDigit(r) {
						return nil
					}
					value = append(value, r)
				}
			default:
				// Invalid escape sequence
				return nil
			}
		} else if r < 0x20 && r != '\t' {
			// Control characters not allowed (except tab)
			return nil
		}
	}
}

// SingleQuotedStringMatcher creates a matcher for YAML single-quoted strings.
// Matches: '...' with only one escape: '' (doubled single quote) becomes '
//
// Grammar:
//
//	SingleQuotedString = "'" { Character } "'" ;
//	Character = [^'] | "''" ;
//
// Performance: Uses ByteStream for fast ASCII scanning.
func SingleQuotedStringMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Try ByteStream fast path
		if byteStream, ok := stream.(tokenizer.ByteStream); ok {
			return singleQuotedStringMatcherByte(byteStream)
		}

		// Fallback to rune-based matcher
		return singleQuotedStringMatcherRune(stream)
	}
}

// singleQuotedStringMatcherByte uses ByteStream for optimal performance.
func singleQuotedStringMatcherByte(stream tokenizer.ByteStream) *tokenizer.Token {
	// Opening quote
	b, ok := stream.PeekByte()
	if !ok || b != '\'' {
		return nil
	}

	startPos := stream.BytePosition()
	stream.NextByte() // consume opening quote

	// Scan until closing quote
	for {
		// Find next single quote
		offset := tokenizer.FindByte(stream.RemainingBytes(), '\'')

		if offset == -1 {
			// No closing quote found - unterminated string
			return nil
		}

		// Advance to the quote
		for i := 0; i < offset; i++ {
			_, ok := stream.NextByte()
			if !ok {
				return nil
			}
		}

		// Consume the quote
		stream.NextByte()

		// Check if it's escaped (doubled '')
		next, ok := stream.PeekByte()
		if ok && next == '\'' {
			// Escaped quote - consume it and continue
			stream.NextByte()
			continue
		}

		// Found closing quote - extract string
		value := stream.SliceFrom(startPos)
		return tokenizer.NewToken(TokenString, []rune(string(value)))
	}
}

// singleQuotedStringMatcherRune is the fallback rune-based implementation.
func singleQuotedStringMatcherRune(stream tokenizer.Stream) *tokenizer.Token {
	var value []rune

	// Opening quote
	r, ok := stream.NextChar()
	if !ok || r != '\'' {
		return nil
	}
	value = append(value, r)

	// Characters until closing quote
	for {
		r, ok := stream.NextChar()
		if !ok {
			return nil
		}

		value = append(value, r)

		if r == '\'' {
			// Check if it's escaped (doubled '')
			next, ok := stream.PeekChar()
			if ok && next == '\'' {
				// Escaped quote - consume it and continue
				r, _ := stream.NextChar()
				value = append(value, r)
				continue
			}

			// Found closing quote
			return tokenizer.NewToken(TokenString, value)
		}
	}
}

// PlainStringMatcher creates a matcher for YAML plain (unquoted) strings.
// Matches: Unquoted strings with restrictions
//
// Restrictions:
// - Cannot start with: -, ?, :, ,, [, ], {, }, #, &, *, !, |, >, ', ", %, @, backtick
// - Cannot contain: ": " (colon-space) or " #" (space-hash)
// - Stops at newline
// - Must not be a boolean or null keyword
//
// Grammar:
//
//	PlainString = PlainFirstChar { PlainChar } ;
//	PlainFirstChar = [^-?:,\[\]{}#&*!|>'"% @`] ;
//	PlainChar = [^\n] but not ": " or " #" ;
func PlainStringMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Try ByteStream fast path
		if byteStream, ok := stream.(tokenizer.ByteStream); ok {
			return plainStringMatcherByte(byteStream)
		}

		// Fallback to rune-based matcher
		return plainStringMatcherRune(stream)
	}
}

// plainStringMatcherByte uses ByteStream for optimal performance.
func plainStringMatcherByte(stream tokenizer.ByteStream) *tokenizer.Token {
	// Check first character
	b, ok := stream.PeekByte()
	if !ok {
		return nil
	}

	// Cannot start with these characters
	if !isPlainSafeStart(b) {
		return nil
	}

	startPos := stream.BytePosition()
	stream.NextByte()

	// Scan until we hit a forbidden pattern or newline
	for {
		b, ok := stream.PeekByte()
		if !ok {
			// EOF - extract what we have
			break
		}

		// Stop at newline
		if b == '\n' || b == '\r' {
			break
		}

		// Stop at whitespace (space or tab)
		if b == ' ' || b == '\t' {
			break
		}

		// Stop at structural characters
		if b == ':' || b == ',' || b == '[' || b == ']' || b == '{' || b == '}' || b == '#' {
			break
		}

		stream.NextByte()
	}

	// Extract the plain string
	value := stream.SliceFrom(startPos)
	if len(value) == 0 {
		return nil
	}

	return tokenizer.NewToken(TokenString, []rune(string(value)))
}

// plainStringMatcherRune is the fallback rune-based implementation.
func plainStringMatcherRune(stream tokenizer.Stream) *tokenizer.Token {
	// Check first character
	r, ok := stream.PeekChar()
	if !ok {
		return nil
	}

	// Cannot start with these characters
	if !isPlainSafeStartRune(r) {
		return nil
	}

	var value []rune
	stream.NextChar()
	value = append(value, r)

	// Scan until we hit a forbidden pattern or newline
	for {
		r, ok := stream.PeekChar()
		if !ok {
			// EOF - return what we have
			break
		}

		// Stop at newline
		if r == '\n' || r == '\r' {
			break
		}

		// Stop at whitespace (space or tab)
		if r == ' ' || r == '\t' {
			break
		}

		// Stop at structural characters
		if r == ':' || r == ',' || r == '[' || r == ']' || r == '{' || r == '}' || r == '#' {
			break
		}

		stream.NextChar()
		value = append(value, r)
	}

	if len(value) == 0 {
		return nil
	}

	return tokenizer.NewToken(TokenString, value)
}

// NumberMatcher creates a matcher for YAML number literals.
// Matches: integers and floats with optional sign and exponent, plus hex/octal
//
// Grammar:
//
//	Number = HexNumber | OctalNumber | DecimalNumber ;
//	HexNumber = "0x" HexDigit+ ;
//	OctalNumber = "0o" OctalDigit+ ;
//	DecimalNumber = [ "-" | "+" ] Integer [ Fraction ] [ Exponent ] ;
//	Integer = "0" | ( [1-9] { Digit } ) ;
//	Fraction = "." Digit+ ;
//	Exponent = ( "e" | "E" ) [ "+" | "-" ] Digit+ ;
//
// Examples: 0, -123, 123.456, 1e10, 1.5e-3, 0x1A, 0o755
// Performance: Uses ByteStream for fast ASCII number scanning.
func NumberMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Try ByteStream fast path for ASCII numbers
		if byteStream, ok := stream.(tokenizer.ByteStream); ok {
			return numberMatcherByte(byteStream)
		}

		// Fallback to rune-based matcher
		return numberMatcherRune(stream)
	}
}

// numberMatcherByte uses ByteStream for optimal number parsing.
func numberMatcherByte(stream tokenizer.ByteStream) *tokenizer.Token {
	startPos := stream.BytePosition()

	// Check for hex (0x) or octal (0o) prefix
	b, ok := stream.PeekByte()
	if !ok {
		return nil
	}

	// Optional sign
	if b == '-' || b == '+' {
		stream.NextByte()
		b, ok = stream.PeekByte()
		if !ok {
			return nil
		}
	}

	// Check for 0x (hex) or 0o (octal)
	if b == '0' {
		stream.NextByte()
		next, ok := stream.PeekByte()
		if ok {
			if next == 'x' || next == 'X' {
				// Hex number
				stream.NextByte()
				if !consumeHexDigits(stream) {
					return nil
				}
				value := stream.SliceFrom(startPos)
				return tokenizer.NewToken(TokenNumber, []rune(string(value)))
			} else if next == 'o' || next == 'O' {
				// Octal number
				stream.NextByte()
				if !consumeOctalDigits(stream) {
					return nil
				}
				value := stream.SliceFrom(startPos)
				return tokenizer.NewToken(TokenNumber, []rune(string(value)))
			}
		}
		// Just a zero - could have fraction/exponent
	} else if isDigitByte(b) {
		// Digits 1-9 followed by more digits
		for {
			b, ok := stream.PeekByte()
			if !ok || !isDigitByte(b) {
				break
			}
			stream.NextByte()
		}
	} else {
		// Not a number
		return nil
	}

	// Optional fraction
	b, ok = stream.PeekByte()
	if ok && b == '.' {
		// Look ahead to ensure there's a digit after the dot
		stream.NextByte()

		b, ok = stream.PeekByte()
		if !ok || !isDigitByte(b) {
			// Not a valid fraction
			return nil
		}

		// Consume digits
		for {
			b, ok := stream.PeekByte()
			if !ok || !isDigitByte(b) {
				break
			}
			stream.NextByte()
		}
	}

	// Optional exponent
	b, ok = stream.PeekByte()
	if ok && (b == 'e' || b == 'E') {
		stream.NextByte()

		// Optional sign
		b, ok = stream.PeekByte()
		if ok && (b == '+' || b == '-') {
			stream.NextByte()
		}

		// Must have at least one digit
		b, ok = stream.PeekByte()
		if !ok || !isDigitByte(b) {
			return nil
		}

		// Consume digits
		for {
			b, ok := stream.PeekByte()
			if !ok || !isDigitByte(b) {
				break
			}
			stream.NextByte()
		}
	}

	// Extract the number as bytes and convert to runes
	value := stream.SliceFrom(startPos)
	return tokenizer.NewToken(TokenNumber, []rune(string(value)))
}

// numberMatcherRune is the fallback rune-based number matcher.
func numberMatcherRune(stream tokenizer.Stream) *tokenizer.Token {
	var value []rune

	// Check for hex (0x) or octal (0o) prefix
	r, ok := stream.PeekChar()
	if !ok {
		return nil
	}

	// Optional sign
	if r == '-' || r == '+' {
		stream.NextChar()
		value = append(value, r)
		r, ok = stream.PeekChar()
		if !ok {
			return nil
		}
	}

	// Check for 0x (hex) or 0o (octal)
	if r == '0' {
		stream.NextChar()
		value = append(value, r)
		next, ok := stream.PeekChar()
		if ok {
			if next == 'x' || next == 'X' {
				// Hex number
				stream.NextChar()
				value = append(value, next)
				hasDigits := false
				for {
					r, ok := stream.PeekChar()
					if !ok || !isHexDigit(r) {
						break
					}
					stream.NextChar()
					value = append(value, r)
					hasDigits = true
				}
				if !hasDigits {
					return nil
				}
				return tokenizer.NewToken(TokenNumber, value)
			} else if next == 'o' || next == 'O' {
				// Octal number
				stream.NextChar()
				value = append(value, next)
				hasDigits := false
				for {
					r, ok := stream.PeekChar()
					if !ok || !isOctalDigit(r) {
						break
					}
					stream.NextChar()
					value = append(value, r)
					hasDigits = true
				}
				if !hasDigits {
					return nil
				}
				return tokenizer.NewToken(TokenNumber, value)
			}
		}
		// Just a zero - could have fraction/exponent
	} else if isDigit(r) {
		// Digits 1-9 followed by more digits
		for {
			r, ok := stream.PeekChar()
			if !ok || !isDigit(r) {
				break
			}
			stream.NextChar()
			value = append(value, r)
		}
	} else {
		// Not a number
		return nil
	}

	// Optional fraction
	r, ok = stream.PeekChar()
	if ok && r == '.' {
		stream.NextChar()
		value = append(value, r)

		// Must have at least one digit after decimal
		r, ok = stream.PeekChar()
		if !ok || !isDigit(r) {
			return nil
		}

		// Consume digits
		for {
			r, ok := stream.PeekChar()
			if !ok || !isDigit(r) {
				break
			}
			stream.NextChar()
			value = append(value, r)
		}
	}

	// Optional exponent
	r, ok = stream.PeekChar()
	if ok && (r == 'e' || r == 'E') {
		stream.NextChar()
		value = append(value, r)

		// Optional sign
		r, ok = stream.PeekChar()
		if ok && (r == '+' || r == '-') {
			stream.NextChar()
			value = append(value, r)
		}

		// Must have at least one digit
		r, ok = stream.PeekChar()
		if !ok || !isDigit(r) {
			return nil
		}

		// Consume digits
		for {
			r, ok := stream.PeekChar()
			if !ok || !isDigit(r) {
				break
			}
			stream.NextChar()
			value = append(value, r)
		}
	}

	return tokenizer.NewToken(TokenNumber, value)
}

// Helper functions

// isDigitByte checks if a byte is a decimal digit (0-9).
func isDigitByte(b byte) bool {
	return b >= '0' && b <= '9'
}

// isDigit returns true if r is a decimal digit (0-9).
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isHexDigitByte checks if a byte is a hex digit (0-9, a-f, A-F).
func isHexDigitByte(b byte) bool {
	return (b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'f') ||
		(b >= 'A' && b <= 'F')
}

// isHexDigit returns true if r is a hexadecimal digit (0-9, a-f, A-F).
func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F')
}

// isOctalDigit returns true if r is an octal digit (0-7).
func isOctalDigit(r rune) bool {
	return r >= '0' && r <= '7'
}

// isOctalDigitByte checks if a byte is an octal digit (0-7).
func isOctalDigitByte(b byte) bool {
	return b >= '0' && b <= '7'
}

// isPlainSafeStart checks if a byte can start a plain scalar.
func isPlainSafeStart(b byte) bool {
	// Cannot start with these characters
	switch b {
	case '-', '?', ':', ',', '[', ']', '{', '}', '#', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
		return false
	case ' ', '\t', '\n', '\r':
		return false
	default:
		return true
	}
}

// isPlainSafeStartRune checks if a rune can start a plain scalar.
func isPlainSafeStartRune(r rune) bool {
	// Cannot start with these characters
	switch r {
	case '-', '?', ':', ',', '[', ']', '{', '}', '#', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
		return false
	case ' ', '\t', '\n', '\r':
		return false
	default:
		return true
	}
}

// consumeHexDigits consumes at least one hex digit. Returns true if successful.
func consumeHexDigits(stream tokenizer.ByteStream) bool {
	hasDigits := false
	for {
		b, ok := stream.PeekByte()
		if !ok || !isHexDigitByte(b) {
			break
		}
		stream.NextByte()
		hasDigits = true
	}
	return hasDigits
}

// consumeOctalDigits consumes at least one octal digit. Returns true if successful.
func consumeOctalDigits(stream tokenizer.ByteStream) bool {
	hasDigits := false
	for {
		b, ok := stream.PeekByte()
		if !ok || !isOctalDigitByte(b) {
			break
		}
		stream.NextByte()
		hasDigits = true
	}
	return hasDigits
}

// BooleanMatcher creates a case-insensitive matcher for YAML boolean keywords.
// Matches: true, True, TRUE, false, False, FALSE, yes, Yes, YES, no, No, NO,
//          on, On, ON, off, Off, OFF
// Returns TokenTrue or TokenFalse based on the matched value.
func BooleanMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Try ByteStream fast path if available
		if byteStream, ok := stream.(tokenizer.ByteStream); ok {
			return booleanMatcherByte(byteStream)
		}

		// Fallback to rune-based matcher
		return booleanMatcherRune(stream)
	}
}

// booleanMatcherByte uses ByteStream to peek ahead without consuming
func booleanMatcherByte(stream tokenizer.ByteStream) *tokenizer.Token {
	// Try each boolean keyword in order (longest first to avoid partial matches)
	keywords := []struct {
		word      string
		tokenKind string
	}{
		{"false", TokenFalse},
		{"true", TokenTrue},
		{"yes", TokenTrue},
		{"off", TokenFalse},
		{"on", TokenTrue},
		{"no", TokenFalse},
	}

	for _, kw := range keywords {
		if token := tryMatchKeywordByte(stream, kw.word, kw.tokenKind); token != nil {
			return token
		}
	}

	return nil
}

// tryMatchKeywordByte attempts to match a keyword case-insensitively using ByteStream
func tryMatchKeywordByte(stream tokenizer.ByteStream, keyword string, tokenKind string) *tokenizer.Token {
	// Peek ahead at the bytes we need
	remaining := stream.RemainingBytes()
	if len(remaining) < len(keyword) {
		return nil
	}

	// Check if keyword matches case-insensitively
	for i := 0; i < len(keyword); i++ {
		b := remaining[i]
		expected := keyword[i]

		// Convert to lowercase for comparison
		if b >= 'A' && b <= 'Z' {
			b = b + ('a' - 'A')
		}

		if b != expected {
			return nil
		}
	}

	// Check word boundary - next byte must not be alphanumeric, underscore, or dash
	if len(remaining) > len(keyword) {
		nextByte := remaining[len(keyword)]
		if (nextByte >= 'a' && nextByte <= 'z') ||
			(nextByte >= 'A' && nextByte <= 'Z') ||
			(nextByte >= '0' && nextByte <= '9') ||
			nextByte == '_' || nextByte == '-' {
			// Not a word boundary
			return nil
		}
	}

	// Match found - consume the bytes
	startPos := stream.BytePosition()
	for i := 0; i < len(keyword); i++ {
		stream.NextByte()
	}

	// Get the actual matched bytes (preserves original case)
	matched := stream.SliceFrom(startPos)
	return tokenizer.NewToken(tokenKind, []rune(string(matched)))
}

// booleanMatcherRune is the fallback for non-ByteStream
func booleanMatcherRune(stream tokenizer.Stream) *tokenizer.Token {
	// For rune streams, we try each keyword
	keywords := []struct {
		word      string
		tokenKind string
	}{
		{"false", TokenFalse},
		{"true", TokenTrue},
		{"yes", TokenTrue},
		{"off", TokenFalse},
		{"on", TokenTrue},
		{"no", TokenFalse},
	}

	for _, kw := range keywords {
		if token := tryMatchCaseInsensitiveKeyword(stream, kw.word, kw.tokenKind); token != nil {
			return token
		}
	}

	return nil
}

// tryMatchCaseInsensitiveKeyword tries to match a keyword case-insensitively
// and ensures it's followed by a word boundary.
func tryMatchCaseInsensitiveKeyword(stream tokenizer.Stream, keyword string, tokenKind string) *tokenizer.Token {
	// Peek at first character to quick-reject
	firstChar, ok := stream.PeekChar()
	if !ok {
		return nil
	}

	// Case-insensitive comparison with first character of keyword
	lowerFirst := firstChar
	if firstChar >= 'A' && firstChar <= 'Z' {
		lowerFirst = firstChar + ('a' - 'A')
	}

	if lowerFirst != rune(keyword[0]) {
		return nil
	}

	// For each position, we need to peek without consuming
	// Build a list of what we expect to match
	keywordRunes := []rune(keyword)

	// Check if we can read enough characters
	var peeked []rune
	for i := 0; i < len(keywordRunes); i++ {
		r, ok := stream.PeekChar()
		if !ok {
			return nil
		}

		// Verify case-insensitive match
		lowerR := r
		if r >= 'A' && r <= 'Z' {
			lowerR = r + ('a' - 'A')
		}

		if lowerR != keywordRunes[i] {
			return nil
		}

		peeked = append(peeked, r)

		// Move to next character for peeking
		if i < len(keywordRunes)-1 {
			stream.NextChar()
		}
	}

	// Now stream is positioned at the last character of the keyword
	// Consume it
	stream.NextChar()

	// Check word boundary
	nextChar, ok := stream.PeekChar()
	if ok {
		if (nextChar >= 'a' && nextChar <= 'z') ||
			(nextChar >= 'A' && nextChar <= 'Z') ||
			(nextChar >= '0' && nextChar <= '9') ||
			nextChar == '_' || nextChar == '-' {
			// Not a word boundary - but we already consumed characters!
			// This is the fundamental problem - can't rewind.
			// The boolean matcher must be ordered carefully.
			// For now, we have to accept this limitation.
			return nil
		}
	}

	return tokenizer.NewToken(tokenKind, peeked)
}

// AnchorMatcher creates a matcher for YAML anchors.
// Matches: &name where name is [a-zA-Z0-9_-]+
func AnchorMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Check for &
		r, ok := stream.PeekChar()
		if !ok || r != '&' {
			return nil
		}

		var value []rune
		stream.NextChar()
		value = append(value, r)

		// Consume identifier characters
		hasChars := false
		for {
			r, ok := stream.PeekChar()
			if !ok {
				break
			}
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '_' || r == '-' {
				stream.NextChar()
				value = append(value, r)
				hasChars = true
			} else {
				break
			}
		}

		if !hasChars {
			return nil
		}

		return tokenizer.NewToken(TokenAnchor, value)
	}
}

// AliasMatcher creates a matcher for YAML aliases.
// Matches: *name where name is [a-zA-Z0-9_-]+
func AliasMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Check for *
		r, ok := stream.PeekChar()
		if !ok || r != '*' {
			return nil
		}

		var value []rune
		stream.NextChar()
		value = append(value, r)

		// Consume identifier characters
		hasChars := false
		for {
			r, ok := stream.PeekChar()
			if !ok {
				break
			}
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '_' || r == '-' {
				stream.NextChar()
				value = append(value, r)
				hasChars = true
			} else {
				break
			}
		}

		if !hasChars {
			return nil
		}

		return tokenizer.NewToken(TokenAlias, value)
	}
}

// TagMatcher creates a matcher for YAML tags.
// Matches: !name, !!name, or !<verbatim> where name is [a-zA-Z0-9_-]+
// Examples:
//   - !Person (custom tag)
//   - !!str (core tag)
//   - !<tag:example.com,2000:type> (verbatim tag)
func TagMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Check for !
		r, ok := stream.PeekChar()
		if !ok || r != '!' {
			return nil
		}

		var value []rune
		stream.NextChar()
		value = append(value, r)

		// Check for verbatim tag: !<...>
		r, ok = stream.PeekChar()
		if ok && r == '<' {
			stream.NextChar()
			value = append(value, r)

			// Consume everything until >
			for {
				r, ok := stream.PeekChar()
				if !ok {
					// Unterminated verbatim tag
					return nil
				}
				stream.NextChar()
				value = append(value, r)
				if r == '>' {
					break
				}
			}

			return tokenizer.NewToken(TokenTag, value)
		}

		// Check for optional second ! (core tags)
		if ok && r == '!' {
			stream.NextChar()
			value = append(value, r)
		}

		// Consume identifier characters
		hasChars := false
		for {
			r, ok := stream.PeekChar()
			if !ok {
				break
			}
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '_' || r == '-' {
				stream.NextChar()
				value = append(value, r)
				hasChars = true
			} else {
				break
			}
		}

		if !hasChars {
			// Just ! or !! without a name is not a valid tag
			return nil
		}

		return tokenizer.NewToken(TokenTag, value)
	}
}

// DirectiveMatcher creates a matcher for YAML directives.
// Matches: %YAML 1.2 or %TAG ! tag:example.com,2000:
// Grammar: "%" DirectiveName DirectiveParameter* Newline
func DirectiveMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Check for %
		r, ok := stream.PeekChar()
		if !ok || r != '%' {
			return nil
		}

		var value []rune
		stream.NextChar()
		value = append(value, r)

		// Consume directive name (uppercase letters)
		hasName := false
		for {
			r, ok := stream.PeekChar()
			if !ok {
				break
			}
			if r >= 'A' && r <= 'Z' {
				stream.NextChar()
				value = append(value, r)
				hasName = true
			} else {
				break
			}
		}

		if !hasName {
			// Not a valid directive
			return nil
		}

		// Consume the rest of the directive line (parameters)
		for {
			r, ok := stream.PeekChar()
			if !ok || r == '\n' || r == '\r' {
				break
			}
			stream.NextChar()
			value = append(value, r)
		}

		return tokenizer.NewToken(TokenDirective, value)
	}
}

// CommentMatcher creates a matcher for YAML comments.
// Matches: # followed by any characters until newline
func CommentMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Check for #
		r, ok := stream.PeekChar()
		if !ok || r != '#' {
			return nil
		}

		var value []rune
		stream.NextChar()
		value = append(value, r)

		// Consume until newline
		for {
			r, ok := stream.PeekChar()
			if !ok || r == '\n' || r == '\r' {
				break
			}
			stream.NextChar()
			value = append(value, r)
		}

		return tokenizer.NewToken(TokenComment, value)
	}
}

// NewlineMatcher creates a matcher for newlines.
// Matches: \n or \r\n
func NewlineMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		r, ok := stream.PeekChar()
		if !ok {
			return nil
		}

		if r == '\r' {
			// Check for \r\n
			stream.NextChar()
			next, ok := stream.PeekChar()
			if ok && next == '\n' {
				stream.NextChar()
				return tokenizer.NewToken(TokenNewline, []rune{'\r', '\n'})
			}
			// Just \r (treat as newline)
			return tokenizer.NewToken(TokenNewline, []rune{'\r'})
		}

		if r == '\n' {
			stream.NextChar()
			return tokenizer.NewToken(TokenNewline, []rune{'\n'})
		}

		return nil
	}
}

// YAMLWhitespaceMatcher creates a matcher for YAML whitespace.
// Unlike the default whitespace matcher, this only matches spaces and tabs,
// NOT newlines (since newlines are significant in YAML structure).
func YAMLWhitespaceMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Try ByteStream fast path
		if byteStream, ok := stream.(tokenizer.ByteStream); ok {
			startPos := byteStream.BytePosition()

			// Consume spaces and tabs only (not newlines)
			for {
				b, ok := byteStream.PeekByte()
				if !ok {
					break
				}
				if b == ' ' || b == '\t' {
					byteStream.NextByte()
				} else {
					break
				}
			}

			endPos := byteStream.BytePosition()
			if endPos == startPos {
				return nil // No whitespace found
			}

			// Extract the whitespace as a token (but it will be discarded)
			value := byteStream.SliceFrom(startPos)
			return tokenizer.NewToken(`Whitespace`, []rune(string(value)))
		}

		// Fallback: Rune-based implementation
		var value []rune
		for {
			r, ok := stream.PeekChar()
			if !ok {
				break
			}
			if r == ' ' || r == '\t' {
				stream.NextChar()
				value = append(value, r)
			} else {
				break
			}
		}

		if len(value) == 0 {
			return nil
		}
		return tokenizer.NewToken(`Whitespace`, value)
	}
}

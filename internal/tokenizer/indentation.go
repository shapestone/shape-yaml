package tokenizer

import (
	"github.com/shapestone/shape-core/pkg/tokenizer"
)

// IndentationTokenizer wraps a base tokenizer and emits INDENT/DEDENT tokens.
// This is critical for YAML parsing as YAML's structure is defined by indentation.
//
// The tokenizer maintains a stack of indentation levels and emits:
// - INDENT when indentation increases
// - DEDENT when indentation decreases (may emit multiple DEDENTs)
// - Tracks line starts to measure indentation
//
// Example:
//
//	Input:
//	  name: Alice
//	  children:
//	    - Bob
//	    - Carol
//
//	Tokens emitted:
//	  INDENT, "name", COLON, "Alice", NEWLINE,
//	  "children", COLON, NEWLINE,
//	  INDENT, DASH, "Bob", NEWLINE,
//	  DASH, "Carol", NEWLINE,
//	  DEDENT, DEDENT
type IndentationTokenizer struct {
	base          tokenizer.Tokenizer
	indentStack   []int             // Stack of indentation levels [0, 2, 4, ...]
	pendingTokens []tokenizer.Token // Queue of tokens to emit
	atLineStart   bool              // Are we at the start of a line?
	lastNewline   bool              // Did we just emit a newline?
	columnAtStart int               // Column number at line start (for indentation)
}

// NewIndentationTokenizer creates an indentation-aware tokenizer that wraps a base tokenizer.
// The wrapper intercepts newlines and emits INDENT/DEDENT tokens based on indentation changes.
func NewIndentationTokenizer(base tokenizer.Tokenizer) *IndentationTokenizer {
	return &IndentationTokenizer{
		base:          base,
		indentStack:   []int{0}, // Start at column 0
		pendingTokens: []tokenizer.Token{},
		atLineStart:   true,
		lastNewline:   false,
		columnAtStart: 1, // Columns are 1-indexed
	}
}

// NextToken returns the next token from the stream, including synthetic INDENT/DEDENT tokens.
func (it *IndentationTokenizer) NextToken() (*tokenizer.Token, bool) {
	// 1. If we have pending tokens (INDENT/DEDENT), return them first
	if len(it.pendingTokens) > 0 {
		token := it.pendingTokens[0]
		it.pendingTokens = it.pendingTokens[1:]
		return &token, true
	}

	// 2. Get next token from base tokenizer
	token, ok := it.base.NextToken()
	if !ok {
		// EOF: emit DEDENTs to return to column 0
		if len(it.indentStack) > 1 {
			it.indentStack = it.indentStack[:len(it.indentStack)-1]
			dedent := tokenizer.NewToken(TokenDedent, []rune{})
			return dedent, true
		}
		return nil, false
	}

	// 3. Track newlines
	if token.Kind() == TokenNewline {
		it.atLineStart = true
		it.lastNewline = true
		return token, true
	}

	// 4. Skip comments (they don't affect indentation)
	if token.Kind() == TokenComment {
		return token, true
	}

	// 5. Skip whitespace tokens at line start - we measure indentation
	//    from the first non-whitespace token
	if it.atLineStart && token.Kind() == "Whitespace" {
		// Don't reset atLineStart - we're still waiting for actual content
		return token, true
	}

	// 6. At line start: measure indentation and emit INDENT/DEDENT
	if it.atLineStart {
		it.atLineStart = false

		// Get the column where this token starts
		// This represents the indentation level
		indent := it.getTokenColumn(*token)

		// Compare with current indentation level
		currentLevel := it.indentStack[len(it.indentStack)-1]

		if indent > currentLevel {
			// INDENT: push new level and emit INDENT token
			it.indentStack = append(it.indentStack, indent)
			indentToken := tokenizer.NewToken(TokenIndent, []rune{})

			// Queue the current token to be returned after INDENT
			it.pendingTokens = append(it.pendingTokens, *token)

			return indentToken, true

		} else if indent < currentLevel {
			// DEDENT: pop levels until we match
			dedentCount := 0
			for len(it.indentStack) > 1 && it.indentStack[len(it.indentStack)-1] > indent {
				it.indentStack = it.indentStack[:len(it.indentStack)-1]
				dedentCount++
			}

			// Check if we landed on exact indentation
			if len(it.indentStack) > 0 && it.indentStack[len(it.indentStack)-1] != indent {
				// Indentation error - not aligned with any previous level
				// For now, we'll be lenient and adjust
				if indent > it.indentStack[len(it.indentStack)-1] {
					it.indentStack = append(it.indentStack, indent)
				}
			}

			// Emit DEDENT tokens
			for i := 0; i < dedentCount; i++ {
				dedentToken := tokenizer.NewToken(TokenDedent, []rune{})
				it.pendingTokens = append(it.pendingTokens, *dedentToken)
			}

			// Queue the current token
			it.pendingTokens = append(it.pendingTokens, *token)

			// Return the first DEDENT (or the token if no dedents)
			if dedentCount > 0 {
				firstToken := it.pendingTokens[0]
				it.pendingTokens = it.pendingTokens[1:]
				return &firstToken, true
			}

			return token, true
		}
	}

	// No indentation change - return token as-is
	return token, true
}

// getTokenColumn extracts the column number from the token's position.
// For YAML, we use column position (1-indexed) as the indentation level.
//
// Note: This is a simplified implementation. A more robust version would:
// - Track actual column positions from the stream
// - Handle tabs vs spaces consistently
// - Provide better error messages for inconsistent indentation
func (it *IndentationTokenizer) getTokenColumn(token tokenizer.Token) int {
	// The token position gives us the column
	// Columns are 1-indexed, but indentation is relative
	// We'll use column - 1 to get 0-based indentation
	col := token.Column()

	// Convert 1-indexed column to 0-based indentation
	if col > 0 {
		return col - 1
	}
	return 0
}

// Initialize initializes the tokenizer with a string input.
func (it *IndentationTokenizer) Initialize(input string) {
	it.base.Initialize(input)
	it.Reset()
}

// InitializeFromStream initializes the tokenizer with a pre-configured stream.
func (it *IndentationTokenizer) InitializeFromStream(stream tokenizer.Stream) {
	it.base.InitializeFromStream(stream)
	it.Reset()
}

// Reset resets the indentation state.
func (it *IndentationTokenizer) Reset() {
	it.indentStack = []int{0}
	it.pendingTokens = []tokenizer.Token{}
	it.atLineStart = true
	it.lastNewline = false
	it.columnAtStart = 1
}

// GetPosition returns the current position in the stream.
// This delegates to the base tokenizer.
func (it *IndentationTokenizer) GetPosition() tokenizer.Position {
	// Get position from the base tokenizer's stream
	// This is a bit tricky since we're wrapping it
	// For now, return a default position
	return tokenizer.Position{
		Offset: 0,
		Line:   1,
		Column: 1,
	}
}

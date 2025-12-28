// Package tokenizer provides YAML tokenization using Shape's tokenizer framework.
package tokenizer

// Token type constants for YAML format.
// These correspond to the terminals in the YAML grammar (docs/grammar/yaml-simple.ebnf).
const (
	// Structural tokens
	TokenColon    = "Colon"    // :
	TokenDash     = "Dash"     // -
	TokenComma    = "Comma"    // , (flow style)
	TokenLBrace   = "LBrace"   // { (flow style)
	TokenRBrace   = "RBrace"   // } (flow style)
	TokenLBracket = "LBracket" // [ (flow style)
	TokenRBracket = "RBracket" // ] (flow style)

	// Indentation tokens (emitted by indentation tracker)
	TokenIndent = "Indent" // Indentation increase
	TokenDedent = "Dedent" // Indentation decrease

	// Value tokens
	TokenString = "String" // Quoted or plain string
	TokenNumber = "Number" // 123, -45.67, 1.23e10
	TokenTrue   = "True"   // true, yes
	TokenFalse  = "False"  // false, no
	TokenNull   = "Null"   // null, ~

	// Special tokens
	TokenNewline      = "Newline"      // \n or \r\n
	TokenComment      = "Comment"      // # ... (usually skipped)
	TokenDocSep       = "DocSep"       // ---
	TokenDocEnd       = "DocEnd"       // ...
	TokenAnchor       = "Anchor"       // &name
	TokenAlias        = "Alias"        // *name
	TokenTag          = "Tag"          // !type or !!type
	TokenBlockLiteral = "BlockLiteral" // | (literal block)
	TokenBlockFolded  = "BlockFolded"  // > (folded block)
	TokenQuestion     = "Question"     // ? (complex key marker)
	TokenMergeKey     = "MergeKey"     // <<
	TokenDirective    = "Directive"    // %YAML or %TAG directive
	TokenEOF          = "EOF"          // End of file
)

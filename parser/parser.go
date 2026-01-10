// Package parser handles parsing text messages
package parser

// Parser parses text messages into structured data
type Parser struct {
	delimiter string
}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{delimiter: ","}
}

// SetDelimiter sets the parsing delimiter
func (p *Parser) SetDelimiter(delim string) {
	p.delimiter = delim
}

// GetDelimiter returns the current delimiter
func (p *Parser) GetDelimiter() string {
	return p.delimiter
}

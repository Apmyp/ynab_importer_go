// Package formatter handles formatting messages for YNAB
package formatter

// Formatter formats parsed messages for YNAB API
type Formatter struct {
	dateFormat string
}

// NewFormatter creates a new Formatter instance
func NewFormatter() *Formatter {
	return &Formatter{dateFormat: "2006-01-02"}
}

// SetDateFormat sets the date format for formatting
func (f *Formatter) SetDateFormat(format string) {
	f.dateFormat = format
}

// GetDateFormat returns the current date format
func (f *Formatter) GetDateFormat() string {
	return f.dateFormat
}

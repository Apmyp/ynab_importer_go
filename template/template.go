// Package template handles parsing SMS messages using predefined templates
package template

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// Transaction represents a parsed bank transaction
type Transaction struct {
	Operation   string
	Card        string
	Status      string
	Amount      float64
	Currency    string
	Balance     float64
	DateTime    string
	Address     string
	Support     string
	FromAccount string
	ToAccount   string
	RawMessage  string
}

// Template defines the interface for message templates
type Template interface {
	// Match returns true if this template matches the message content
	Match(content string) bool
	// Parse extracts transaction data from the message content
	Parse(content string) (*Transaction, error)
	// Name returns the template name
	Name() string
}

// MAIBTemplate handles MAIB bank (sender 102) transaction messages
type MAIBTemplate struct {
	opRegex     *regexp.Regexp
	fieldRegex  *regexp.Regexp
	amountRegex *regexp.Regexp
}

// NewMAIBTemplate creates a new MAIB template
func NewMAIBTemplate() *MAIBTemplate {
	return &MAIBTemplate{
		opRegex:     regexp.MustCompile(`^Op: (.+)`),
		fieldRegex:  regexp.MustCompile(`^([^:]+): (.*)$`),
		amountRegex: regexp.MustCompile(`^([\d,]+)\s*(\w+)$`),
	}
}

// Name returns the template name
func (t *MAIBTemplate) Name() string {
	return "MAIB"
}

// Match returns true if this template matches the message
func (t *MAIBTemplate) Match(content string) bool {
	return t.opRegex.MatchString(content)
}

// Parse extracts transaction data from MAIB message
func (t *MAIBTemplate) Parse(content string) (*Transaction, error) {
	lines := strings.Split(content, "\n")
	tx := &Transaction{RawMessage: content}

	for _, line := range lines {
		matches := t.fieldRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		key := strings.TrimSpace(matches[1])
		value := strings.TrimSpace(matches[2])

		switch key {
		case "Op":
			tx.Operation = value
		case "Karta":
			tx.Card = value
		case "Status":
			tx.Status = value
		case "Summa":
			amount, currency, err := t.parseAmount(value)
			if err != nil {
				return nil, err
			}
			tx.Amount = amount
			tx.Currency = currency
		case "Dost":
			balance, err := t.parseNumber(value)
			if err != nil {
				return nil, err
			}
			tx.Balance = balance
		case "Data/vremya":
			tx.DateTime = value
		case "Adres":
			tx.Address = value
		case "Podderzhka":
			tx.Support = value
		}
	}

	return tx, nil
}

func (t *MAIBTemplate) parseAmount(value string) (float64, string, error) {
	matches := t.amountRegex.FindStringSubmatch(value)
	if matches == nil {
		return 0, "", errors.New("invalid amount format")
	}

	amount, err := t.parseNumber(matches[1])
	if err != nil {
		return 0, "", err
	}

	return amount, matches[2], nil
}

func (t *MAIBTemplate) parseNumber(value string) (float64, error) {
	// Replace comma with dot for decimal parsing
	normalized := strings.Replace(value, ",", ".", -1)
	return strconv.ParseFloat(normalized, 64)
}

// EximTransactionTemplate handles Eximbank transaction confirmation messages
type EximTransactionTemplate struct {
	regex *regexp.Regexp
}

// NewEximTransactionTemplate creates a new Exim transaction template
func NewEximTransactionTemplate() *EximTransactionTemplate {
	return &EximTransactionTemplate{
		regex: regexp.MustCompile(`Tranzactia din (\d{2}/\d{2}/\d{4}) din contul (\S+) in contul (\S+) in suma de ([\d.]+) (\w+) a fost (\w+)`),
	}
}

// Name returns the template name
func (t *EximTransactionTemplate) Name() string {
	return "EximTransaction"
}

// Match returns true if this template matches the message
func (t *EximTransactionTemplate) Match(content string) bool {
	return t.regex.MatchString(content)
}

// Parse extracts transaction data from Exim transaction message
func (t *EximTransactionTemplate) Parse(content string) (*Transaction, error) {
	matches := t.regex.FindStringSubmatch(content)
	if matches == nil {
		return nil, errors.New("failed to parse Exim transaction")
	}

	amount, err := strconv.ParseFloat(matches[4], 64)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		DateTime:    matches[1],
		FromAccount: matches[2],
		ToAccount:   matches[3],
		Amount:      amount,
		Currency:    matches[5],
		Status:      matches[6],
		RawMessage:  content,
	}, nil
}

// Matcher holds all templates and finds matching ones
type Matcher struct {
	templates []Template
}

// NewMatcher creates a new Matcher with all registered templates
func NewMatcher() *Matcher {
	return &Matcher{
		templates: []Template{
			NewMAIBTemplate(),
			NewEximTransactionTemplate(),
		},
	}
}

// FindTemplate returns the first matching template for the content
func (m *Matcher) FindTemplate(content string) Template {
	for _, tmpl := range m.templates {
		if tmpl.Match(content) {
			return tmpl
		}
	}
	return nil
}

// Parse parses the content using the first matching template
func (m *Matcher) Parse(content string) (*Transaction, error) {
	tmpl := m.FindTemplate(content)
	if tmpl == nil {
		return nil, errors.New("no matching template found")
	}
	return tmpl.Parse(content)
}

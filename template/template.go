// Package template handles parsing SMS messages using predefined templates
package template

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// Amount represents a monetary amount in a specific currency
type Amount struct {
	Value    float64
	Currency string
}

// Transaction represents a parsed bank transaction
type Transaction struct {
	Operation   string
	Card        string
	Status      string
	Original    Amount
	Converted   Amount
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
			tx.Original = Amount{Value: amount, Currency: currency}
		case "Dost":
			balance, err := parseNumber(value)
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

	amount, err := parseNumber(matches[1])
	if err != nil {
		return 0, "", err
	}

	return amount, matches[2], nil
}

// parseNumber normalizes and parses a numeric string
func parseNumber(value string) (float64, error) {
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
		Original:    Amount{Value: amount, Currency: matches[5]},
		Status:      matches[6],
		RawMessage:  content,
	}, nil
}

// DebitareTemplate handles Eximbank debit (withdrawal) messages
type DebitareTemplate struct {
	regex *regexp.Regexp
}

// NewDebitareTemplate creates a new Debitare template
func NewDebitareTemplate() *DebitareTemplate {
	return &DebitareTemplate{
		// Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii ..., Disponibil 38400.60 MDL
		// Detalii may contain commas, so use .+? with $ anchor
		regex: regexp.MustCompile(`Debitare cont Card ([^,]+), Data ([^,]+), Suma ([\d.]+) (\w+), Detalii (.+?), Disponibil ([\d.]+) \w+$`),
	}
}

// Name returns the template name
func (t *DebitareTemplate) Name() string {
	return "Debitare"
}

// Match returns true if this template matches the message
func (t *DebitareTemplate) Match(content string) bool {
	return t.regex.MatchString(content)
}

// Parse extracts transaction data from Debitare message
func (t *DebitareTemplate) Parse(content string) (*Transaction, error) {
	matches := t.regex.FindStringSubmatch(content)
	if matches == nil {
		return nil, errors.New("failed to parse Debitare message")
	}

	amount, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return nil, err
	}

	balance, err := strconv.ParseFloat(matches[6], 64)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		Operation:  "Debitare",
		Card:       matches[1],
		DateTime:   matches[2],
		Original:   Amount{Value: amount, Currency: matches[4]},
		Address:    matches[5], // Using Address field for Detalii
		Balance:    balance,
		RawMessage: content,
	}, nil
}

// TranzactieReusitaTemplate handles successful transaction messages (debit)
type TranzactieReusitaTemplate struct {
	regex *regexp.Regexp
}

// NewTranzactieReusitaTemplate creates a new TranzactieReusita template
func NewTranzactieReusitaTemplate() *TranzactieReusitaTemplate {
	return &TranzactieReusitaTemplate{
		// Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie MAIB GROCERY STORE BETA>CHISINAU, MDA, Disponibil 31200.80 MDL
		regex: regexp.MustCompile(`Tranzactie reusita, Data ([^,]+), Card ([^,]+), Suma ([\d.]+) (\w+), Locatie ([^,]+, \w+), Disponibil ([\d.]+)`),
	}
}

// Name returns the template name
func (t *TranzactieReusitaTemplate) Name() string {
	return "TranzactieReusita"
}

// Match returns true if this template matches the message
func (t *TranzactieReusitaTemplate) Match(content string) bool {
	return t.regex.MatchString(content)
}

// Parse extracts transaction data from TranzactieReusita message
func (t *TranzactieReusitaTemplate) Parse(content string) (*Transaction, error) {
	matches := t.regex.FindStringSubmatch(content)
	if matches == nil {
		return nil, errors.New("failed to parse TranzactieReusita message")
	}

	amount, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return nil, err
	}

	balance, err := strconv.ParseFloat(matches[6], 64)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		Operation:  "Tranzactie reusita",
		DateTime:   matches[1],
		Card:       matches[2],
		Original:   Amount{Value: amount, Currency: matches[4]},
		Address:    matches[5], // Location
		Balance:    balance,
		RawMessage: content,
	}, nil
}

// SuplinireTemplate handles card top-up/credit messages
type SuplinireTemplate struct {
	regex *regexp.Regexp
}

// NewSuplinireTemplate creates a new Suplinire template
func NewSuplinireTemplate() *SuplinireTemplate {
	return &SuplinireTemplate{
		// Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 93719.33 MDL, Detalii Plata salariala luna aprilie, Disponibil 88700.25 MDL
		// Also matches without Disponibil: Suplinire cont Card 9..7890, Data 13.01.2025 16:13:56, Suma 990 RUB, Detalii ONLINE SERVICE GAMMA> 44712345678, GBR
		regex: regexp.MustCompile(`Suplinire cont Card ([^,]+), Data ([^,]+), Suma ([\d.]+) (\w+), Detalii (.+?)(?:, Disponibil ([\d.]+) \w+)?$`),
	}
}

// Name returns the template name
func (t *SuplinireTemplate) Name() string {
	return "Suplinire"
}

// Match returns true if this template matches the message
func (t *SuplinireTemplate) Match(content string) bool {
	return t.regex.MatchString(content)
}

// Parse extracts transaction data from Suplinire message
func (t *SuplinireTemplate) Parse(content string) (*Transaction, error) {
	matches := t.regex.FindStringSubmatch(content)
	if matches == nil {
		return nil, errors.New("failed to parse Suplinire message")
	}

	amount, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return nil, err
	}

	var balance float64
	if len(matches) > 6 && matches[6] != "" {
		balance, err = strconv.ParseFloat(matches[6], 64)
		if err != nil {
			return nil, err
		}
	}

	return &Transaction{
		Operation:  "Suplinire",
		Card:       matches[1],
		DateTime:   matches[2],
		Original:   Amount{Value: amount, Currency: matches[4]},
		Address:    matches[5], // Using Address field for Detalii
		Balance:    balance,
		RawMessage: content,
	}, nil
}

// Matcher holds all templates and finds matching ones
type Matcher struct {
	templates      []Template
	ignorePatterns []string
}

// NewMatcher creates a new Matcher with all registered templates
func NewMatcher() *Matcher {
	return &Matcher{
		templates: []Template{
			NewMAIBTemplate(),
			NewEximTransactionTemplate(),
			NewDebitareTemplate(),
			NewTranzactieReusitaTemplate(),
			NewSuplinireTemplate(),
		},
		ignorePatterns: []string{
			// MAIB messages
			"Vas privetstvuet servis opoveshenia ot MAIB",
			"Oper.: Ostatok",
			// Eximbank auth/OTP messages
			"Autentificarea Dvs. in sistemul Eximbank Online a fost inregistrata la",
			"Parola de unica folosinta pentru tranzactia cu ID-ul",
			"OTP-ul pentru Plati din Exim Personal este",
			"Va multumim ca ati ales serviciul Eximbank SMS Info.",
			"Parola de Unica Folosinta (OTP) a Dvs. pentru logare este",
			"Parola:",     // Covers "Parola:XXXXXX Card X..XXXX" patterns
			"Parola Dvs.", // Covers "Parola Dvs. este XXXXXXXX"
			// Transaction status messages to ignore
			"Tranzactie esuata,",
			"Tranzactia din", // Eximbank transfer confirmations
			"Anulare tranzactie",
			// Promotional/marketing messages
			"Acesta este momentul pe care il asteptai!",
			"Vrei un card pentru copilul tau?",
			"Refinanteaza creditele de consum de la alte",
			"Profita acum! Credit PERSONAL sau MAGNIFIC",
			// Maintenance notifications
			"In data de",
			// Apple Pay notifications
			"Cardul Eximbank",
			// User's own messages
			"] Me:",
		},
	}
}

// ShouldIgnore returns true if the message matches an ignore pattern
func (m *Matcher) ShouldIgnore(content string) bool {
	for _, pattern := range m.ignorePatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
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

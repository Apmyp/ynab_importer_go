package template

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

type Amount struct {
	Value    float64
	Currency string
}

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

type Template interface {
	Match(content string) bool
	Parse(content string) (*Transaction, error)
	Name() string
}

type MAIBTemplate struct {
	opRegex     *regexp.Regexp
	fieldRegex  *regexp.Regexp
	amountRegex *regexp.Regexp
}

func NewMAIBTemplate() *MAIBTemplate {
	return &MAIBTemplate{
		opRegex:     regexp.MustCompile(`^Op: (.+)`),
		fieldRegex:  regexp.MustCompile(`^([^:]+): (.*)$`),
		amountRegex: regexp.MustCompile(`^([\d,]+)\s*(\w+)$`),
	}
}

func (t *MAIBTemplate) Name() string {
	return "MAIB"
}

func (t *MAIBTemplate) Match(content string) bool {
	return t.opRegex.MatchString(content)
}

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

func parseNumber(value string) (float64, error) {
	normalized := strings.Replace(value, ",", ".", -1)
	return strconv.ParseFloat(normalized, 64)
}

type EximTransactionTemplate struct {
	regex *regexp.Regexp
}

func NewEximTransactionTemplate() *EximTransactionTemplate {
	return &EximTransactionTemplate{
		regex: regexp.MustCompile(`Tranzactia din (\d{2}/\d{2}/\d{4}) din contul (\S+) in contul (\S+) in suma de ([\d.]+) (\w+) a fost (\w+)`),
	}
}

func (t *EximTransactionTemplate) Name() string {
	return "EximTransaction"
}

func (t *EximTransactionTemplate) Match(content string) bool {
	return t.regex.MatchString(content)
}

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

type DebitareTemplate struct {
	regex *regexp.Regexp
}

func NewDebitareTemplate() *DebitareTemplate {
	return &DebitareTemplate{
		// Example: Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii ..., Disponibil 38400.60 MDL
		regex: regexp.MustCompile(`Debitare cont Card ([^,]+), Data ([^,]+), Suma ([\d.]+) (\w+), Detalii (.+?), Disponibil ([\d.]+) \w+$`),
	}
}

func (t *DebitareTemplate) Name() string {
	return "Debitare"
}

func (t *DebitareTemplate) Match(content string) bool {
	return t.regex.MatchString(content)
}

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
		Address:    matches[5],
		Balance:    balance,
		RawMessage: content,
	}, nil
}

type TranzactieReusitaTemplate struct {
	regex *regexp.Regexp
}

func NewTranzactieReusitaTemplate() *TranzactieReusitaTemplate {
	return &TranzactieReusitaTemplate{
		// Example: Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie MAIB GROCERY STORE>CHISINAU, MDA, Disponibil 31200.80 MDL
		regex: regexp.MustCompile(`Tranzactie reusita, Data ([^,]+), Card ([^,]+), Suma ([\d.]+) (\w+), Locatie ([^,]+, \w+), Disponibil ([\d.]+)`),
	}
}

func (t *TranzactieReusitaTemplate) Name() string {
	return "TranzactieReusita"
}

func (t *TranzactieReusitaTemplate) Match(content string) bool {
	return t.regex.MatchString(content)
}

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
		Address:    matches[5],
		Balance:    balance,
		RawMessage: content,
	}, nil
}

type SuplinireTemplate struct {
	regex *regexp.Regexp
}

func NewSuplinireTemplate() *SuplinireTemplate {
	return &SuplinireTemplate{
		// Example: Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 93719.33 MDL, Detalii Plata salariala, Disponibil 88700.25 MDL
		regex: regexp.MustCompile(`Suplinire cont Card ([^,]+), Data ([^,]+), Suma ([\d.]+) (\w+), Detalii (.+?)(?:, Disponibil ([\d.]+) \w+)?$`),
	}
}

func (t *SuplinireTemplate) Name() string {
	return "Suplinire"
}

func (t *SuplinireTemplate) Match(content string) bool {
	return t.regex.MatchString(content)
}

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
		Address:    matches[5],
		Balance:    balance,
		RawMessage: content,
	}, nil
}

type Matcher struct {
	templates      []Template
	ignorePatterns []string
}

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
			"Vas privetstvuet servis opoveshenia ot MAIB",
			"Oper.: Ostatok",
			"Autentificarea Dvs. in sistemul Eximbank Online a fost inregistrata la",
			"Parola de unica folosinta pentru tranzactia cu ID-ul",
			"OTP-ul pentru Plati din Exim Personal este",
			"Va multumim ca ati ales serviciul Eximbank SMS Info.",
			"Parola de Unica Folosinta (OTP) a Dvs. pentru logare este",
			"Parola:",
			"Parola Dvs.",
			"Tranzactie esuata,",
			"Tranzactia din",
			"Anulare tranzactie",
			"Acesta este momentul pe care il asteptai!",
			"Vrei un card pentru copilul tau?",
			"Refinanteaza creditele de consum de la alte",
			"Profita acum! Credit PERSONAL sau MAGNIFIC",
			"In data de",
			"Cardul Eximbank",
			"] Me:",
		},
	}
}

func (m *Matcher) ShouldIgnore(content string) bool {
	for _, pattern := range m.ignorePatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

func (m *Matcher) FindTemplate(content string) Template {
	for _, tmpl := range m.templates {
		if tmpl.Match(content) {
			return tmpl
		}
	}
	return nil
}

func (m *Matcher) Parse(content string) (*Transaction, error) {
	tmpl := m.FindTemplate(content)
	if tmpl == nil {
		return nil, errors.New("no matching template found")
	}
	return tmpl.Parse(content)
}

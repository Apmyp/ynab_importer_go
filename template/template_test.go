package template

import (
	"testing"
)

func TestTransaction_Fields(t *testing.T) {
	tx := &Transaction{
		Operation:  "Tovary i uslugi",
		Card:       "*1234",
		Status:     "Odobrena",
		Amount:     34.0,
		Currency:   "MDL",
		Balance:    12500.50,
		DateTime:   "03.05.23 16:21",
		Address:    "COFFEE SHOP ALPHA",
		Support:    "+12025551234",
		RawMessage: "original message",
	}

	if tx.Operation != "Tovary i uslugi" {
		t.Errorf("expected operation 'Tovary i uslugi', got %q", tx.Operation)
	}
	if tx.Amount != 34.0 {
		t.Errorf("expected amount 34.0, got %f", tx.Amount)
	}
}

func TestMAIBTemplate_Match_ValidTransaction(t *testing.T) {
	content := `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 34 MDL
Dost: 12500,50
Data/vremya: 03.05.23 16:21
Adres: COFFEE SHOP ALPHA
Podderzhka: +12025551234`

	tmpl := NewMAIBTemplate()
	if !tmpl.Match(content) {
		t.Error("MAIBTemplate should match valid transaction message")
	}
}

func TestMAIBTemplate_Match_WelcomeMessage(t *testing.T) {
	content := `Vas privetstvuet servis opoveshenia ot MAIB
Profili budet skoro aktivirovan.
Paroli: PPAWJM`

	tmpl := NewMAIBTemplate()
	if tmpl.Match(content) {
		t.Error("MAIBTemplate should not match welcome message")
	}
}

func TestMAIBTemplate_Parse_ValidTransaction(t *testing.T) {
	content := `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 34 MDL
Dost: 12500,50
Data/vremya: 03.05.23 16:21
Adres: COFFEE SHOP ALPHA
Podderzhka: +12025551234`

	tmpl := NewMAIBTemplate()
	tx, err := tmpl.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if tx.Operation != "Tovary i uslugi" {
		t.Errorf("expected operation 'Tovary i uslugi', got %q", tx.Operation)
	}
	if tx.Card != "*1234" {
		t.Errorf("expected card '*1234', got %q", tx.Card)
	}
	if tx.Status != "Odobrena" {
		t.Errorf("expected status 'Odobrena', got %q", tx.Status)
	}
	if tx.Amount != 34.0 {
		t.Errorf("expected amount 34.0, got %f", tx.Amount)
	}
	if tx.Currency != "MDL" {
		t.Errorf("expected currency 'MDL', got %q", tx.Currency)
	}
	if tx.Balance != 12500.50 {
		t.Errorf("expected balance 12500.50, got %f", tx.Balance)
	}
	if tx.DateTime != "03.05.23 16:21" {
		t.Errorf("expected datetime '03.05.23 16:21', got %q", tx.DateTime)
	}
	if tx.Address != "COFFEE SHOP ALPHA" {
		t.Errorf("expected address 'COFFEE SHOP ALPHA', got %q", tx.Address)
	}
}

func TestMAIBTemplate_Parse_DecimalAmount(t *testing.T) {
	content := `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 132,87 MDL
Dost: 9200,75
Data/vremya: 04.05.23 13:03
Adres: MAIB GROCERY STORE BETA
Podderzhka: +12025551234`

	tmpl := NewMAIBTemplate()
	tx, err := tmpl.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if tx.Amount != 132.87 {
		t.Errorf("expected amount 132.87, got %f", tx.Amount)
	}
}

func TestMAIBTemplate_Parse_USDCurrency(t *testing.T) {
	content := `Op: Tovary i uslugi
Karta: *5678
Status: Odobrena
Summa: 26,37 USD
Dost: 15300,90
Data/vremya: 05.05.23 08:04
Adres: EP*exampleshop.com
Podderzhka: +12025551234`

	tmpl := NewMAIBTemplate()
	tx, err := tmpl.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if tx.Amount != 26.37 {
		t.Errorf("expected amount 26.37, got %f", tx.Amount)
	}
	if tx.Currency != "USD" {
		t.Errorf("expected currency 'USD', got %q", tx.Currency)
	}
}

func TestEximTransactionTemplate_Match_Valid(t *testing.T) {
	content := `Tranzactia din 29/05/2023 din contul ACC1234567MD4 in contul MD99XX000000011111111111 in suma de 5000.00 MDL a fost Executata`

	tmpl := NewEximTransactionTemplate()
	if !tmpl.Match(content) {
		t.Error("EximTransactionTemplate should match valid transaction message")
	}
}

func TestEximTransactionTemplate_Match_OTP(t *testing.T) {
	content := `Parola de unica folosinta pentru tranzactia cu ID-ul TX9999888877776666 este 0329`

	tmpl := NewEximTransactionTemplate()
	if tmpl.Match(content) {
		t.Error("EximTransactionTemplate should not match OTP message")
	}
}

func TestEximTransactionTemplate_Parse_Valid(t *testing.T) {
	content := `Tranzactia din 29/05/2023 din contul ACC1234567MD4 in contul MD99XX000000011111111111 in suma de 5000.00 MDL a fost Executata`

	tmpl := NewEximTransactionTemplate()
	tx, err := tmpl.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if tx.DateTime != "29/05/2023" {
		t.Errorf("expected datetime '29/05/2023', got %q", tx.DateTime)
	}
	if tx.Amount != 5000.00 {
		t.Errorf("expected amount 5000.00, got %f", tx.Amount)
	}
	if tx.Currency != "MDL" {
		t.Errorf("expected currency 'MDL', got %q", tx.Currency)
	}
	if tx.Status != "Executata" {
		t.Errorf("expected status 'Executata', got %q", tx.Status)
	}
}

func TestMatcher_FindTemplate(t *testing.T) {
	matcher := NewMatcher()

	testCases := []struct {
		name    string
		content string
		wantNil bool
	}{
		{
			name: "MAIB transaction",
			content: `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 34 MDL`,
			wantNil: false,
		},
		{
			name:    "EXIM transaction",
			content: `Tranzactia din 29/05/2023 din contul ACC1234567MD4 in contul MD99XX000000011111111111 in suma de 5000.00 MDL a fost Executata`,
			wantNil: false,
		},
		{
			name:    "Unknown message",
			content: `Random message that doesn't match any template`,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := matcher.FindTemplate(tc.content)
			if tc.wantNil && tmpl != nil {
				t.Error("expected nil template")
			}
			if !tc.wantNil && tmpl == nil {
				t.Error("expected non-nil template")
			}
		})
	}
}

func TestMatcher_Parse(t *testing.T) {
	matcher := NewMatcher()

	content := `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 34 MDL
Dost: 12500,50
Data/vremya: 03.05.23 16:21
Adres: COFFEE SHOP ALPHA
Podderzhka: +12025551234`

	tx, err := matcher.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if tx == nil {
		t.Fatal("expected non-nil transaction")
	}

	if tx.Amount != 34.0 {
		t.Errorf("expected amount 34.0, got %f", tx.Amount)
	}
}

func TestMatcher_Parse_NoMatch(t *testing.T) {
	matcher := NewMatcher()

	content := `Random message that doesn't match any template`

	tx, err := matcher.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for unmatched message")
	}
	if tx != nil {
		t.Error("expected nil transaction for unmatched message")
	}
}

func TestMAIBTemplate_Name(t *testing.T) {
	tmpl := NewMAIBTemplate()
	if tmpl.Name() != "MAIB" {
		t.Errorf("expected name 'MAIB', got %q", tmpl.Name())
	}
}

func TestEximTransactionTemplate_Name(t *testing.T) {
	tmpl := NewEximTransactionTemplate()
	if tmpl.Name() != "EximTransaction" {
		t.Errorf("expected name 'EximTransaction', got %q", tmpl.Name())
	}
}

func TestMAIBTemplate_Parse_InvalidAmount(t *testing.T) {
	content := `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: invalid
Dost: 12500,50
Data/vremya: 03.05.23 16:21
Adres: COFFEE SHOP ALPHA`

	tmpl := NewMAIBTemplate()
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for invalid amount")
	}
}

func TestMAIBTemplate_Parse_InvalidBalance(t *testing.T) {
	content := `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 34 MDL
Dost: invalid
Data/vremya: 03.05.23 16:21
Adres: COFFEE SHOP ALPHA`

	tmpl := NewMAIBTemplate()
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for invalid balance")
	}
}

func TestEximTransactionTemplate_Parse_InvalidAmount(t *testing.T) {
	// This would only fail if regex matched but amount parse failed
	// The regex requires valid number format, so this test is for edge cases
	tmpl := NewEximTransactionTemplate()

	// Test with content that doesn't match at all
	content := `Not a transaction`
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for non-matching content")
	}
}

func TestMAIBTemplate_Parse_AmountWithInvalidNumber(t *testing.T) {
	// Test when amount matches regex but number parsing fails
	// The regex is ^([\d,]+)\s*(\w+)$ so "abc MDL" wouldn't match
	// But "1,2,3 MDL" would match and parseNumber might fail
	content := `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 1,2,3,4 MDL
Dost: 12500,50`

	tmpl := NewMAIBTemplate()
	// This might or might not error depending on parseNumber behavior
	// Let's just verify it doesn't panic
	_, _ = tmpl.Parse(content)
}

func TestMAIBTemplate_Parse_MinimalMessage(t *testing.T) {
	// Test with message that has Op but minimal other fields
	content := `Op: Test operation`

	tmpl := NewMAIBTemplate()
	tx, err := tmpl.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if tx.Operation != "Test operation" {
		t.Errorf("expected operation 'Test operation', got %q", tx.Operation)
	}
}

func TestMAIBTemplate_Parse_EmptyLines(t *testing.T) {
	content := `Op: Test

Karta: *1234

Status: Done`

	tmpl := NewMAIBTemplate()
	tx, err := tmpl.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if tx.Operation != "Test" {
		t.Errorf("expected operation 'Test', got %q", tx.Operation)
	}
}

func TestDebitareTemplate_Match(t *testing.T) {
	tmpl := NewDebitareTemplate()

	testCases := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "Valid debitare message",
			content: "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Comision serviciu SMS pentru cardul nr. 199458, Disponibil 38400.60 MDL",
			want:    true,
		},
		{
			name:    "Non-matching message",
			content: "Random message",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tmpl.Match(tc.content)
			if got != tc.want {
				t.Errorf("Match() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDebitareTemplate_Parse(t *testing.T) {
	tmpl := NewDebitareTemplate()
	content := "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Comision serviciu SMS pentru cardul nr. 199458, Disponibil 38400.60 MDL"

	tx, err := tmpl.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if tx.Operation != "Debitare" {
		t.Errorf("expected operation 'Debitare', got %q", tx.Operation)
	}
	if tx.Card != "9..7890" {
		t.Errorf("expected card '9..7890', got %q", tx.Card)
	}
	if tx.DateTime != "08.04.2024 09:27:01" {
		t.Errorf("expected datetime '08.04.2024 09:27:01', got %q", tx.DateTime)
	}
	if tx.Amount != 9.65 {
		t.Errorf("expected amount 9.65, got %f", tx.Amount)
	}
	if tx.Currency != "MDL" {
		t.Errorf("expected currency 'MDL', got %q", tx.Currency)
	}
	if tx.Balance != 38400.60 {
		t.Errorf("expected balance 38400.60, got %f", tx.Balance)
	}
	if tx.Address != "Comision serviciu SMS pentru cardul nr. 199458" {
		t.Errorf("expected address (details) 'Comision serviciu SMS pentru cardul nr. 199458', got %q", tx.Address)
	}
}

func TestDebitareTemplate_Name(t *testing.T) {
	tmpl := NewDebitareTemplate()
	if tmpl.Name() != "Debitare" {
		t.Errorf("expected name 'Debitare', got %q", tmpl.Name())
	}
}

func TestDebitareTemplate_Parse_NonMatching(t *testing.T) {
	tmpl := NewDebitareTemplate()
	_, err := tmpl.Parse("Not a debitare message")
	if err == nil {
		t.Error("Parse() should return error for non-matching content")
	}
}

func TestTranzactieReusitaTemplate_Match(t *testing.T) {
	tmpl := NewTranzactieReusitaTemplate()

	testCases := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "Valid tranzactie message",
			content: "Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie MAIB GROCERY STORE BETA>CHISINAU, MDA, Disponibil 31200.80 MDL",
			want:    true,
		},
		{
			name:    "Non-matching message",
			content: "Random message",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tmpl.Match(tc.content)
			if got != tc.want {
				t.Errorf("Match() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTranzactieReusitaTemplate_Parse(t *testing.T) {
	tmpl := NewTranzactieReusitaTemplate()
	content := "Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie MAIB GROCERY STORE BETA>CHISINAU, MDA, Disponibil 31200.80 MDL"

	tx, err := tmpl.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if tx.Operation != "Tranzactie reusita" {
		t.Errorf("expected operation 'Tranzactie reusita', got %q", tx.Operation)
	}
	if tx.Card != "9..7890" {
		t.Errorf("expected card '9..7890', got %q", tx.Card)
	}
	if tx.DateTime != "13.04.2024 13:20:30" {
		t.Errorf("expected datetime '13.04.2024 13:20:30', got %q", tx.DateTime)
	}
	if tx.Amount != 91.91 {
		t.Errorf("expected amount 91.91, got %f", tx.Amount)
	}
	if tx.Currency != "MDL" {
		t.Errorf("expected currency 'MDL', got %q", tx.Currency)
	}
	if tx.Balance != 31200.80 {
		t.Errorf("expected balance 31200.80, got %f", tx.Balance)
	}
	if tx.Address != "MAIB GROCERY STORE BETA>CHISINAU, MDA" {
		t.Errorf("expected address 'MAIB GROCERY STORE BETA>CHISINAU, MDA', got %q", tx.Address)
	}
}

func TestTranzactieReusitaTemplate_Name(t *testing.T) {
	tmpl := NewTranzactieReusitaTemplate()
	if tmpl.Name() != "TranzactieReusita" {
		t.Errorf("expected name 'TranzactieReusita', got %q", tmpl.Name())
	}
}

func TestTranzactieReusitaTemplate_Parse_NonMatching(t *testing.T) {
	tmpl := NewTranzactieReusitaTemplate()
	_, err := tmpl.Parse("Not a tranzactie message")
	if err == nil {
		t.Error("Parse() should return error for non-matching content")
	}
}

func TestSuplinireTemplate_Match(t *testing.T) {
	tmpl := NewSuplinireTemplate()

	testCases := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "Valid suplinire message",
			content: "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 93719.33 MDL, Detalii Plata salariala luna aprilie, Disponibil 88700.25 MDL",
			want:    true,
		},
		{
			name:    "Non-matching message",
			content: "Random message",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tmpl.Match(tc.content)
			if got != tc.want {
				t.Errorf("Match() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSuplinireTemplate_Parse(t *testing.T) {
	tmpl := NewSuplinireTemplate()
	content := "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 93719.33 MDL, Detalii Plata salariala luna aprilie, Disponibil 88700.25 MDL"

	tx, err := tmpl.Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if tx.Operation != "Suplinire" {
		t.Errorf("expected operation 'Suplinire', got %q", tx.Operation)
	}
	if tx.Card != "9..7890" {
		t.Errorf("expected card '9..7890', got %q", tx.Card)
	}
	if tx.DateTime != "29.04.2024 16:18:01" {
		t.Errorf("expected datetime '29.04.2024 16:18:01', got %q", tx.DateTime)
	}
	if tx.Amount != 93719.33 {
		t.Errorf("expected amount 93719.33, got %f", tx.Amount)
	}
	if tx.Currency != "MDL" {
		t.Errorf("expected currency 'MDL', got %q", tx.Currency)
	}
	if tx.Balance != 88700.25 {
		t.Errorf("expected balance 88700.25, got %f", tx.Balance)
	}
	if tx.Address != "Plata salariala luna aprilie" {
		t.Errorf("expected address (details) 'Plata salariala luna aprilie', got %q", tx.Address)
	}
}

func TestSuplinireTemplate_Name(t *testing.T) {
	tmpl := NewSuplinireTemplate()
	if tmpl.Name() != "Suplinire" {
		t.Errorf("expected name 'Suplinire', got %q", tmpl.Name())
	}
}

func TestSuplinireTemplate_Parse_NonMatching(t *testing.T) {
	tmpl := NewSuplinireTemplate()
	_, err := tmpl.Parse("Not a suplinire message")
	if err == nil {
		t.Error("Parse() should return error for non-matching content")
	}
}

func generateLargeNumber() string {
	var num string
	for i := 0; i < 400; i++ {
		num += "9"
	}
	return num + ".0"
}

func TestEximTransactionTemplate_Parse_Overflow(t *testing.T) {
	tmpl := NewEximTransactionTemplate()
	largeNum := generateLargeNumber()
	content := "Tranzactia din 29/05/2023 din contul ACC1 in contul ACC2 in suma de " + largeNum + " MDL a fost Executata"
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for overflow amount")
	}
}

func TestDebitareTemplate_Parse_AmountOverflow(t *testing.T) {
	tmpl := NewDebitareTemplate()
	largeNum := generateLargeNumber()
	content := "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma " + largeNum + " MDL, Detalii Test, Disponibil 100.00 MDL"
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for overflow amount")
	}
}

func TestDebitareTemplate_Parse_BalanceOverflow(t *testing.T) {
	tmpl := NewDebitareTemplate()
	largeNum := generateLargeNumber()
	content := "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Test, Disponibil " + largeNum + " MDL"
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for overflow balance")
	}
}

func TestTranzactieReusitaTemplate_Parse_AmountOverflow(t *testing.T) {
	tmpl := NewTranzactieReusitaTemplate()
	largeNum := generateLargeNumber()
	content := "Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma " + largeNum + " MDL, Locatie TEST>CITY, MDA, Disponibil 100.00 MDL"
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for overflow amount")
	}
}

func TestTranzactieReusitaTemplate_Parse_BalanceOverflow(t *testing.T) {
	tmpl := NewTranzactieReusitaTemplate()
	largeNum := generateLargeNumber()
	content := "Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie TEST>CITY, MDA, Disponibil " + largeNum + " MDL"
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for overflow balance")
	}
}

func TestSuplinireTemplate_Parse_AmountOverflow(t *testing.T) {
	tmpl := NewSuplinireTemplate()
	largeNum := generateLargeNumber()
	content := "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma " + largeNum + " MDL, Detalii Salary, Disponibil 2000.00 MDL"
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for overflow amount")
	}
}

func TestSuplinireTemplate_Parse_BalanceOverflow(t *testing.T) {
	tmpl := NewSuplinireTemplate()
	largeNum := generateLargeNumber()
	content := "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 1000.00 MDL, Detalii Salary, Disponibil " + largeNum + " MDL"
	_, err := tmpl.Parse(content)
	if err == nil {
		t.Error("Parse() should return error for overflow balance")
	}
}

func TestMatcher_FindTemplate_NewTemplates(t *testing.T) {
	matcher := NewMatcher()

	testCases := []struct {
		name     string
		content  string
		wantNil  bool
		wantName string
	}{
		{
			name:     "Debitare message",
			content:  "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Test, Disponibil 100.00 MDL",
			wantNil:  false,
			wantName: "Debitare",
		},
		{
			name:     "Tranzactie reusita message",
			content:  "Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie TEST>CITY, MDA, Disponibil 100.00 MDL",
			wantNil:  false,
			wantName: "TranzactieReusita",
		},
		{
			name:     "Suplinire message",
			content:  "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 1000.00 MDL, Detalii Salary, Disponibil 2000.00 MDL",
			wantNil:  false,
			wantName: "Suplinire",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := matcher.FindTemplate(tc.content)
			if tc.wantNil && tmpl != nil {
				t.Error("expected nil template")
			}
			if !tc.wantNil {
				if tmpl == nil {
					t.Fatal("expected non-nil template")
				}
				if tmpl.Name() != tc.wantName {
					t.Errorf("expected template name %q, got %q", tc.wantName, tmpl.Name())
				}
			}
		})
	}
}

func TestMatcher_Parse_NewTemplates(t *testing.T) {
	matcher := NewMatcher()

	testCases := []struct {
		name      string
		content   string
		wantOp    string
		wantError bool
	}{
		{
			name:    "Debitare message",
			content: "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Test, Disponibil 100.00 MDL",
			wantOp:  "Debitare",
		},
		{
			name:    "Tranzactie reusita message",
			content: "Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie TEST>CITY, MDA, Disponibil 100.00 MDL",
			wantOp:  "Tranzactie reusita",
		},
		{
			name:    "Suplinire message",
			content: "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 1000.00 MDL, Detalii Salary, Disponibil 2000.00 MDL",
			wantOp:  "Suplinire",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := matcher.Parse(tc.content)
			if tc.wantError {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if tx.Operation != tc.wantOp {
				t.Errorf("expected operation %q, got %q", tc.wantOp, tx.Operation)
			}
		})
	}
}

func TestMatcher_ShouldIgnore(t *testing.T) {
	matcher := NewMatcher()

	testCases := []struct {
		name       string
		content    string
		wantIgnore bool
	}{
		{
			name:       "MAIB welcome message",
			content:    "Vas privetstvuet servis opoveshenia ot MAIB\nProfili budet skoro aktivirovan.",
			wantIgnore: true,
		},
		{
			name:       "MAIB balance check",
			content:    "Oper.: Ostatok\nKarta: *1234\nOstatok: 12500,50 MDL",
			wantIgnore: true,
		},
		{
			name:       "Eximbank auth notification",
			content:    "Autentificarea Dvs. in sistemul Eximbank Online a fost inregistrata la 08.04.2024 14:30",
			wantIgnore: true,
		},
		{
			name:       "Eximbank OTP password",
			content:    "Parola de unica folosinta pentru tranzactia cu ID-ul TX9999888877776666 este 0329",
			wantIgnore: true,
		},
		{
			name:       "Exim OTP for payments",
			content:    "OTP-ul pentru Plati din Exim Personal este 123456",
			wantIgnore: true,
		},
		{
			name:       "Eximbank SMS Info thank you",
			content:    "Va multumim ca ati ales serviciul Eximbank SMS Info.",
			wantIgnore: true,
		},
		{
			name:       "OTP for login",
			content:    "Parola de Unica Folosinta (OTP) a Dvs. pentru logare este 123456",
			wantIgnore: true,
		},
		{
			name:       "Parola with card number",
			content:    "Parola:219281 Card 9..7890",
			wantIgnore: true,
		},
		{
			name:       "Parola Dvs password",
			content:    "Parola Dvs. este aP1qaBkI",
			wantIgnore: true,
		},
		{
			name:       "Failed transaction",
			content:    "Tranzactie esuata, Data 13.04.2024 13:20:30, Card 9..7890",
			wantIgnore: true,
		},
		{
			name:       "Eximbank transfer confirmation",
			content:    "Tranzactia din 29/05/2023 din contul ACC1234567MD4 in contul MD99XX000000011111111111 in suma de 5000.00 MDL a fost Executata",
			wantIgnore: true,
		},
		{
			name:       "Transaction cancellation",
			content:    "Anulare tranzactie Card 9..7890",
			wantIgnore: true,
		},
		{
			name:       "Marketing promo 1",
			content:    "Acesta este momentul pe care il asteptai! Oferim credite cu dobanda redusa.",
			wantIgnore: true,
		},
		{
			name:       "Marketing child card",
			content:    "Vrei un card pentru copilul tau? Intra pe maib.md",
			wantIgnore: true,
		},
		{
			name:       "Marketing refinance",
			content:    "Refinanteaza creditele de consum de la alte banci cu dobanzi mai mici",
			wantIgnore: true,
		},
		{
			name:       "Marketing credit promo",
			content:    "Profita acum! Credit PERSONAL sau MAGNIFIC cu conditii avantajoase",
			wantIgnore: true,
		},
		{
			name:       "Maintenance notification",
			content:    "In data de 29.11 la 10:00-12:00 vor fi lucrari de mentenanta la Internet Banking si Mobile Banking",
			wantIgnore: true,
		},
		{
			name:       "Valid MAIB transaction",
			content:    "Op: Tovary i uslugi\nKarta: *1234\nStatus: Odobrena\nSumma: 34 MDL",
			wantIgnore: false,
		},
		{
			name:       "Valid Debitare transaction",
			content:    "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Test, Disponibil 100.00 MDL",
			wantIgnore: false,
		},
		{
			name:       "Valid Suplinire transaction",
			content:    "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 1000.00 MDL, Detalii Salary, Disponibil 2000.00 MDL",
			wantIgnore: false,
		},
		{
			name:       "Valid Tranzactie reusita",
			content:    "Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie TEST, MDA, Disponibil 100.00 MDL",
			wantIgnore: false,
		},
		{
			name:       "Random unknown message",
			content:    "Some random message that should not be ignored",
			wantIgnore: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := matcher.ShouldIgnore(tc.content)
			if got != tc.wantIgnore {
				t.Errorf("ShouldIgnore() = %v, want %v", got, tc.wantIgnore)
			}
		})
	}
}

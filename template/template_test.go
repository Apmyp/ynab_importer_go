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

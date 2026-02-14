package services

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// newImporter is a shortcut used in every test.
func newImporter() *XLSXImporter {
	return NewXLSXImporter()
}

// ---------------------------------------------------------------------------
// parseBillLabel tests
// ---------------------------------------------------------------------------

func TestParseBillLabel_CreditCardWithLabel(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name         string
		input        string
		wantName     string
		wantDueDay   int
		wantCardLbl  string
		wantStmtDay  int
		wantIssuer   string
		wantCategory string
	}{
		{
			name:         "standard cc with label",
			input:        "IzzCC - QS ***8186 :: (statement=7th, due=4th)",
			wantName:     "IzzCC",
			wantDueDay:   4,
			wantCardLbl:  "QS ***8186",
			wantStmtDay:  7,
			wantIssuer:   "IzzCC",
			wantCategory: "debt",
		},
		{
			name:         "cc with label different ordinals",
			input:        "AmexCC - Plat ***1234 :: (statement=21st, due=18th)",
			wantName:     "AmexCC",
			wantDueDay:   18,
			wantCardLbl:  "Plat ***1234",
			wantStmtDay:  21,
			wantIssuer:   "AmexCC",
			wantCategory: "debt",
		},
		{
			name:         "cc with label no comma between statement and due",
			input:        "BankCC - Gold :: (statement=1st due=28th)",
			wantName:     "BankCC",
			wantDueDay:   28,
			wantCardLbl:  "Gold",
			wantStmtDay:  1,
			wantIssuer:   "BankCC",
			wantCategory: "debt",
		},
		{
			name:         "cc with label extra spaces",
			input:        "IzzCC  -  QS ***8186  ::  (statement=7th, due=4th)",
			wantName:     "IzzCC",
			wantDueDay:   4,
			wantCardLbl:  "QS ***8186",
			wantStmtDay:  7,
			wantIssuer:   "IzzCC",
			wantCategory: "debt",
		},
		{
			name:         "cc with label case insensitive",
			input:        "MyCard - Label :: (STATEMENT=15th, DUE=12th)",
			wantName:     "MyCard",
			wantDueDay:   12,
			wantCardLbl:  "Label",
			wantStmtDay:  15,
			wantIssuer:   "MyCard",
			wantCategory: "debt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bill := imp.parseBillLabel(tt.input)
			if bill == nil {
				t.Fatal("parseBillLabel returned nil")
			}
			if bill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", bill.Name, tt.wantName)
			}
			if bill.DueDay == nil {
				t.Fatal("DueDay is nil")
			}
			if *bill.DueDay != tt.wantDueDay {
				t.Errorf("DueDay = %d, want %d", *bill.DueDay, tt.wantDueDay)
			}
			if bill.Category != tt.wantCategory {
				t.Errorf("Category = %q, want %q", bill.Category, tt.wantCategory)
			}
			if bill.CreditCard == nil {
				t.Fatal("CreditCard is nil")
			}
			if bill.CreditCard.CardLabel != tt.wantCardLbl {
				t.Errorf("CardLabel = %q, want %q", bill.CreditCard.CardLabel, tt.wantCardLbl)
			}
			if bill.CreditCard.StatementDay != tt.wantStmtDay {
				t.Errorf("StatementDay = %d, want %d", bill.CreditCard.StatementDay, tt.wantStmtDay)
			}
			if bill.CreditCard.DueDay != tt.wantDueDay {
				t.Errorf("CreditCard.DueDay = %d, want %d", bill.CreditCard.DueDay, tt.wantDueDay)
			}
			if bill.CreditCard.Issuer != tt.wantIssuer {
				t.Errorf("Issuer = %q, want %q", bill.CreditCard.Issuer, tt.wantIssuer)
			}
			if bill.IsAutopay {
				t.Error("IsAutopay should be false for credit cards")
			}
			if bill.DefaultAmt != nil {
				t.Error("DefaultAmt should be nil for credit cards")
			}
		})
	}
}

func TestParseBillLabel_SimpleCreditCard(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name        string
		input       string
		wantName    string
		wantDueDay  int
		wantStmtDay int
	}{
		{
			name:        "chase simple cc",
			input:       "Chase :: (statement=20th, due=17th)",
			wantName:    "Chase",
			wantDueDay:  17,
			wantStmtDay: 20,
		},
		{
			name:        "amex simple cc",
			input:       "Amex :: (statement=3rd, due=1st)",
			wantName:    "Amex",
			wantDueDay:  1,
			wantStmtDay: 3,
		},
		{
			name:        "simple cc case insensitive",
			input:       "Discover :: (STATEMENT=10th, DUE=7th)",
			wantName:    "Discover",
			wantDueDay:  7,
			wantStmtDay: 10,
		},
		{
			name:        "simple cc no comma",
			input:       "CapitalOne :: (statement=22nd due=19th)",
			wantName:    "CapitalOne",
			wantDueDay:  19,
			wantStmtDay: 22,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bill := imp.parseBillLabel(tt.input)
			if bill == nil {
				t.Fatal("parseBillLabel returned nil")
			}
			if bill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", bill.Name, tt.wantName)
			}
			if bill.DueDay == nil || *bill.DueDay != tt.wantDueDay {
				t.Errorf("DueDay = %v, want %d", bill.DueDay, tt.wantDueDay)
			}
			if bill.Category != "debt" {
				t.Errorf("Category = %q, want %q", bill.Category, "debt")
			}
			if bill.CreditCard == nil {
				t.Fatal("CreditCard is nil")
			}
			if bill.CreditCard.StatementDay != tt.wantStmtDay {
				t.Errorf("StatementDay = %d, want %d", bill.CreditCard.StatementDay, tt.wantStmtDay)
			}
			if bill.CreditCard.DueDay != tt.wantDueDay {
				t.Errorf("CreditCard.DueDay = %d, want %d", bill.CreditCard.DueDay, tt.wantDueDay)
			}
			if bill.CreditCard.Issuer != tt.wantName {
				t.Errorf("Issuer = %q, want %q", bill.CreditCard.Issuer, tt.wantName)
			}
			// CardLabel should be empty for simple CC
			if bill.CreditCard.CardLabel != "" {
				t.Errorf("CardLabel = %q, want empty", bill.CreditCard.CardLabel)
			}
			if bill.IsAutopay {
				t.Error("IsAutopay should be false")
			}
		})
	}
}

func TestParseBillLabel_AutopayWithAmount(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name       string
		input      string
		wantName   string
		wantDay    int
		wantAmount float64
	}{
		{
			name:       "saving autopay with amount",
			input:      "Saving (12th Auto - 25)",
			wantName:   "Saving",
			wantDay:    12,
			wantAmount: 25,
		},
		{
			name:       "autopay with larger amount",
			input:      "Investment (1st Auto - 500)",
			wantName:   "Investment",
			wantDay:    1,
			wantAmount: 500,
		},
		{
			name:       "autopay amount case insensitive",
			input:      "Savings (15th auto - 100)",
			wantName:   "Savings",
			wantDay:    15,
			wantAmount: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bill := imp.parseBillLabel(tt.input)
			if bill == nil {
				t.Fatal("parseBillLabel returned nil")
			}
			if bill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", bill.Name, tt.wantName)
			}
			if bill.DueDay == nil || *bill.DueDay != tt.wantDay {
				t.Errorf("DueDay = %v, want %d", bill.DueDay, tt.wantDay)
			}
			if !bill.IsAutopay {
				t.Error("IsAutopay should be true")
			}
			if bill.DefaultAmt == nil {
				t.Fatal("DefaultAmt is nil")
			}
			if !almostEqual(*bill.DefaultAmt, tt.wantAmount) {
				t.Errorf("DefaultAmt = %f, want %f", *bill.DefaultAmt, tt.wantAmount)
			}
			if bill.CreditCard != nil {
				t.Error("CreditCard should be nil")
			}
		})
	}
}

func TestParseBillLabel_AutopaySimple(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantDay  int
	}{
		{
			name:     "verizon autopay",
			input:    "Verizon (16th) - Auto",
			wantName: "Verizon",
			wantDay:  16,
		},
		{
			name:     "internet autopay",
			input:    "Internet (5th) - Auto",
			wantName: "Internet",
			wantDay:  5,
		},
		{
			name:     "autopay simple case insensitive",
			input:    "Electric (22nd) - auto",
			wantName: "Electric",
			wantDay:  22,
		},
		{
			name:     "autopay simple 1st",
			input:    "Netflix (1st) - Auto",
			wantName: "Netflix",
			wantDay:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bill := imp.parseBillLabel(tt.input)
			if bill == nil {
				t.Fatal("parseBillLabel returned nil")
			}
			if bill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", bill.Name, tt.wantName)
			}
			if bill.DueDay == nil || *bill.DueDay != tt.wantDay {
				t.Errorf("DueDay = %v, want %d", bill.DueDay, tt.wantDay)
			}
			if !bill.IsAutopay {
				t.Error("IsAutopay should be true")
			}
			if bill.DefaultAmt != nil {
				t.Errorf("DefaultAmt should be nil, got %f", *bill.DefaultAmt)
			}
			if bill.CreditCard != nil {
				t.Error("CreditCard should be nil")
			}
		})
	}
}

func TestParseBillLabel_DueEquals(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantDay  int
		wantCat  string
	}{
		{
			name:     "car payment",
			input:    "Car Payment (due=9th)",
			wantName: "Car Payment",
			wantDay:  9,
			wantCat:  "transportation",
		},
		{
			name:     "mortgage due",
			input:    "Mortgage (due=1st)",
			wantName: "Mortgage",
			wantDay:  1,
			wantCat:  "housing",
		},
		{
			name:     "insurance due",
			input:    "Car Payment (due=15th)",
			wantName: "Car Payment",
			wantDay:  15,
			wantCat:  "transportation",
		},
		{
			name:     "due equals case insensitive",
			input:    "Rent (DUE=3rd)",
			wantName: "Rent",
			wantDay:  3,
			wantCat:  "housing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bill := imp.parseBillLabel(tt.input)
			if bill == nil {
				t.Fatal("parseBillLabel returned nil")
			}
			if bill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", bill.Name, tt.wantName)
			}
			if bill.DueDay == nil || *bill.DueDay != tt.wantDay {
				t.Errorf("DueDay = %v, want %d", bill.DueDay, tt.wantDay)
			}
			if bill.IsAutopay {
				t.Error("IsAutopay should be false")
			}
			if bill.Category != tt.wantCat {
				t.Errorf("Category = %q, want %q", bill.Category, tt.wantCat)
			}
			if bill.CreditCard != nil {
				t.Error("CreditCard should be nil")
			}
			if bill.DefaultAmt != nil {
				t.Error("DefaultAmt should be nil")
			}
		})
	}
}

func TestParseBillLabel_BiweeklyAmount(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name       string
		input      string
		wantName   string
		wantAmount float64
	}{
		{
			name:       "house cleaning biweekly",
			input:      "House Cleaning ($160 bi-weekly)",
			wantName:   "House Cleaning",
			wantAmount: 160,
		},
		{
			name:       "biweekly without hyphen",
			input:      "Lawn Service ($80 biweekly)",
			wantName:   "Lawn Service",
			wantAmount: 80,
		},
		{
			name:       "biweekly case insensitive",
			input:      "Pet Walker ($50 Bi-Weekly)",
			wantName:   "Pet Walker",
			wantAmount: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bill := imp.parseBillLabel(tt.input)
			if bill == nil {
				t.Fatal("parseBillLabel returned nil")
			}
			if bill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", bill.Name, tt.wantName)
			}
			if bill.DueDay != nil {
				t.Errorf("DueDay should be nil for biweekly, got %d", *bill.DueDay)
			}
			if bill.DefaultAmt == nil {
				t.Fatal("DefaultAmt is nil")
			}
			if !almostEqual(*bill.DefaultAmt, tt.wantAmount) {
				t.Errorf("DefaultAmt = %f, want %f", *bill.DefaultAmt, tt.wantAmount)
			}
			if bill.Category != "personal" {
				t.Errorf("Category = %q, want %q", bill.Category, "personal")
			}
			if bill.IsAutopay {
				t.Error("IsAutopay should be false")
			}
			if bill.CreditCard != nil {
				t.Error("CreditCard should be nil")
			}
		})
	}
}

func TestParseBillLabel_SimpleDay(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name      string
		input     string
		wantName  string
		wantDay   int
		wantAuto  bool
		wantCat   string
	}{
		{
			name:     "hulu with day",
			input:    "Hulu (7th)",
			wantName: "Hulu",
			wantDay:  7,
			wantAuto: false,
			wantCat:  "subscriptions",
		},
		{
			name:     "al power with multiple days",
			input:    "AL Power (7th, 21st)",
			wantName: "AL Power",
			wantDay:  7, // first day captured
			wantAuto: false,
			wantCat:  "utilities",
		},
		{
			name:     "simple day 1st",
			input:    "Netflix (1st)",
			wantName: "Netflix",
			wantDay:  1,
			wantAuto: false,
			wantCat:  "subscriptions",
		},
		{
			name:     "simple day 2nd",
			input:    "Spire Gas (2nd)",
			wantName: "Spire Gas",
			wantDay:  2,
			wantAuto: false,
			wantCat:  "utilities",
		},
		{
			name:     "simple day 3rd",
			input:    "Water (3rd)",
			wantName: "Water",
			wantDay:  3,
			wantAuto: false,
			wantCat:  "utilities",
		},
		{
			name:     "simple day 22nd",
			input:    "Apple (22nd)",
			wantName: "Apple",
			wantDay:  22,
			wantAuto: false,
			wantCat:  "subscriptions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bill := imp.parseBillLabel(tt.input)
			if bill == nil {
				t.Fatal("parseBillLabel returned nil")
			}
			if bill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", bill.Name, tt.wantName)
			}
			if bill.DueDay == nil || *bill.DueDay != tt.wantDay {
				t.Errorf("DueDay = %v, want %d", bill.DueDay, tt.wantDay)
			}
			if bill.IsAutopay != tt.wantAuto {
				t.Errorf("IsAutopay = %v, want %v", bill.IsAutopay, tt.wantAuto)
			}
			if bill.Category != tt.wantCat {
				t.Errorf("Category = %q, want %q", bill.Category, tt.wantCat)
			}
			if bill.DefaultAmt != nil {
				t.Error("DefaultAmt should be nil")
			}
			if bill.CreditCard != nil {
				t.Error("CreditCard should be nil")
			}
		})
	}
}

func TestParseBillLabel_SimpleDayAutoDetection(t *testing.T) {
	// The simpleDay branch also checks for the word "auto" in the label.
	// If an input like "AutoPay Svc (10th)" slips past the autopaySimple
	// regex, the simpleDay branch will still detect "auto" in the name.
	imp := newImporter()

	bill := imp.parseBillLabel("AutoPay Club (10th)")
	if bill == nil {
		t.Fatal("parseBillLabel returned nil")
	}
	if !bill.IsAutopay {
		t.Error("IsAutopay should be true when label contains 'auto'")
	}
}

func TestParseBillLabel_Fallback(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantCat  string
	}{
		{
			name:     "plain name",
			input:    "Groceries",
			wantName: "Groceries",
			wantCat:  "other",
		},
		{
			name:     "plain name with spaces",
			input:    "  Random Bill  ",
			wantName: "Random Bill",
			wantCat:  "other",
		},
		{
			name:     "name matching a category keyword",
			input:    "Mortgage",
			wantName: "Mortgage",
			wantCat:  "housing",
		},
		{
			name:     "name matching insurance",
			input:    "Home Insurance",
			wantName: "Home Insurance",
			wantCat:  "insurance",
		},
		{
			name:     "name matching savings",
			input:    "Emergency Saving",
			wantName: "Emergency Saving",
			wantCat:  "savings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bill := imp.parseBillLabel(tt.input)
			if bill == nil {
				t.Fatal("parseBillLabel returned nil")
			}
			if bill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", bill.Name, tt.wantName)
			}
			if bill.Category != tt.wantCat {
				t.Errorf("Category = %q, want %q", bill.Category, tt.wantCat)
			}
			if bill.DueDay != nil {
				t.Errorf("DueDay should be nil for fallback, got %d", *bill.DueDay)
			}
			if bill.IsAutopay {
				t.Error("IsAutopay should be false")
			}
			if bill.DefaultAmt != nil {
				t.Error("DefaultAmt should be nil")
			}
			if bill.CreditCard != nil {
				t.Error("CreditCard should be nil")
			}
		})
	}
}

func TestParseBillLabel_NeverReturnsNil(t *testing.T) {
	imp := newImporter()

	inputs := []string{
		"anything",
		"!!special##chars",
		"123",
		"  ",
		"()",
		"test (invalid)",
	}

	for _, input := range inputs {
		bill := imp.parseBillLabel(input)
		if bill == nil {
			t.Errorf("parseBillLabel(%q) returned nil", input)
		}
	}
}

// ---------------------------------------------------------------------------
// parseBillLabel pattern priority tests
// ---------------------------------------------------------------------------

func TestParseBillLabel_PatternPriority(t *testing.T) {
	imp := newImporter()

	// ccWithLabel should match before ccSimple
	t.Run("ccWithLabel takes priority over ccSimple", func(t *testing.T) {
		bill := imp.parseBillLabel("IzzCC - QS ***8186 :: (statement=7th, due=4th)")
		if bill == nil {
			t.Fatal("returned nil")
		}
		if bill.CreditCard == nil {
			t.Fatal("CreditCard is nil")
		}
		if bill.CreditCard.CardLabel != "QS ***8186" {
			t.Errorf("Expected CardLabel from ccWithLabel match, got %q", bill.CreditCard.CardLabel)
		}
	})

	// autopayAmount should match before autopaySimple and simpleDay
	t.Run("autopayAmount takes priority over simpleDay", func(t *testing.T) {
		bill := imp.parseBillLabel("Saving (12th Auto - 25)")
		if bill == nil {
			t.Fatal("returned nil")
		}
		if bill.DefaultAmt == nil {
			t.Fatal("DefaultAmt should not be nil for autopayAmount")
		}
		if !almostEqual(*bill.DefaultAmt, 25) {
			t.Errorf("DefaultAmt = %f, want 25", *bill.DefaultAmt)
		}
	})

	// autopaySimple matches "Name (Nth) - Auto" before simpleDay
	t.Run("autopaySimple takes priority over simpleDay for Auto suffix", func(t *testing.T) {
		bill := imp.parseBillLabel("Verizon (16th) - Auto")
		if bill == nil {
			t.Fatal("returned nil")
		}
		if !bill.IsAutopay {
			t.Error("Should be detected as autopay via autopaySimple")
		}
		if bill.DefaultAmt != nil {
			t.Error("DefaultAmt should be nil for autopaySimple")
		}
	})

	// dueEquals should match before simpleDay
	t.Run("dueEquals takes priority over simpleDay", func(t *testing.T) {
		bill := imp.parseBillLabel("Car Payment (due=9th)")
		if bill == nil {
			t.Fatal("returned nil")
		}
		if bill.DueDay == nil || *bill.DueDay != 9 {
			t.Errorf("DueDay = %v, want 9", bill.DueDay)
		}
		if bill.IsAutopay {
			t.Error("IsAutopay should be false for dueEquals")
		}
	})
}

// ---------------------------------------------------------------------------
// ParseCellValue tests
// ---------------------------------------------------------------------------

func TestParseCellValue_PaidMarker(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name       string
		input      string
		wantStatus string
		wantAmount *float64
	}{
		{
			name:       "simple paid",
			input:      "**paid",
			wantStatus: "paid",
			wantAmount: nil,
		},
		{
			name:       "paid uppercase",
			input:      "**PAID",
			wantStatus: "paid",
			wantAmount: nil,
		},
		{
			name:       "paid mixed case",
			input:      "**Paid",
			wantStatus: "paid",
			wantAmount: nil,
		},
		{
			name:       "paid with amount before",
			input:      "150**paid",
			wantStatus: "paid",
			wantAmount: floatPtr(150),
		},
		{
			name:       "paid with dollar amount",
			input:      "$200**paid",
			wantStatus: "paid",
			wantAmount: floatPtr(200),
		},
		{
			name:       "paid with spaces around",
			input:      "  **paid  ",
			wantStatus: "paid",
			wantAmount: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := imp.ParseCellValue(tt.input)
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", result.Status, tt.wantStatus)
			}
			if tt.wantAmount == nil && result.Amount != nil {
				t.Errorf("Amount = %f, want nil", *result.Amount)
			}
			if tt.wantAmount != nil {
				if result.Amount == nil {
					t.Fatalf("Amount is nil, want %f", *tt.wantAmount)
				}
				if !almostEqual(*result.Amount, *tt.wantAmount) {
					t.Errorf("Amount = %f, want %f", *result.Amount, *tt.wantAmount)
				}
			}
		})
	}
}

func TestParseCellValue_DeferMarker(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name  string
		input string
	}{
		{"simple defer", "|-->"},
		{"defer with spaces", "  |-->  "},
		{"defer with text after", "|--> next month"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := imp.ParseCellValue(tt.input)
			if result.Status != "deferred" {
				t.Errorf("Status = %q, want %q", result.Status, "deferred")
			}
			if result.Amount != nil {
				t.Error("Amount should be nil for deferred")
			}
		})
	}
}

func TestParseCellValue_UncertainMarker(t *testing.T) {
	imp := newImporter()

	t.Run("exact uncertain marker", func(t *testing.T) {
		result := imp.ParseCellValue("??")
		if result.Status != "uncertain" {
			t.Errorf("Status = %q, want %q", result.Status, "uncertain")
		}
		if result.Amount != nil {
			t.Error("Amount should be nil for uncertain")
		}
	})

	t.Run("uncertain with surrounding spaces is still matched after trim", func(t *testing.T) {
		result := imp.ParseCellValue("  ??  ")
		if result.Status != "uncertain" {
			t.Errorf("Status = %q, want %q", result.Status, "uncertain")
		}
	})

	t.Run("not uncertain when other text present", func(t *testing.T) {
		result := imp.ParseCellValue("??extra")
		// The regex is ^\\?\\?$ so this should NOT match as uncertain
		if result.Status == "uncertain" {
			t.Error("Should not match as uncertain with extra text")
		}
	})

	t.Run("single question mark is not uncertain", func(t *testing.T) {
		result := imp.ParseCellValue("?")
		if result.Status == "uncertain" {
			t.Error("Single '?' should not match as uncertain")
		}
	})
}

func TestParseCellValue_NumericValues(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name       string
		input      string
		wantAmount float64
		wantStatus string
	}{
		{
			name:       "integer",
			input:      "100",
			wantAmount: 100,
			wantStatus: "pending",
		},
		{
			name:       "decimal",
			input:      "99.99",
			wantAmount: 99.99,
			wantStatus: "pending",
		},
		{
			name:       "with dollar sign",
			input:      "$250",
			wantAmount: 250,
			wantStatus: "pending",
		},
		{
			name:       "with dollar sign and decimals",
			input:      "$1,234.56",
			wantAmount: 1234.56,
			wantStatus: "pending",
		},
		{
			name:       "with commas",
			input:      "1,000",
			wantAmount: 1000,
			wantStatus: "pending",
		},
		{
			name:       "with surrounding spaces",
			input:      "  45.50  ",
			wantAmount: 45.50,
			wantStatus: "pending",
		},
		{
			name:       "zero",
			input:      "0",
			wantAmount: 0,
			wantStatus: "pending",
		},
		{
			name:       "negative number",
			input:      "-50",
			wantAmount: -50,
			wantStatus: "pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := imp.ParseCellValue(tt.input)
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", result.Status, tt.wantStatus)
			}
			if result.Amount == nil {
				t.Fatalf("Amount is nil, want %f", tt.wantAmount)
			}
			if !almostEqual(*result.Amount, tt.wantAmount) {
				t.Errorf("Amount = %f, want %f", *result.Amount, tt.wantAmount)
			}
			if result.Note != "" {
				t.Errorf("Note = %q, want empty", result.Note)
			}
		})
	}
}

func TestParseCellValue_PlainText(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name     string
		input    string
		wantNote string
	}{
		{
			name:     "plain text",
			input:    "some note",
			wantNote: "some note",
		},
		{
			name:     "text with numbers mixed in",
			input:    "check #123",
			wantNote: "check #123",
		},
		{
			name:     "alphabetic only",
			input:    "N/A",
			wantNote: "N/A",
		},
		{
			name:     "text with leading/trailing spaces",
			input:    "  pending review  ",
			wantNote: "pending review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := imp.ParseCellValue(tt.input)
			if result.Note != tt.wantNote {
				t.Errorf("Note = %q, want %q", result.Note, tt.wantNote)
			}
			if result.Amount != nil {
				t.Error("Amount should be nil for plain text")
			}
			if result.Status != "" {
				t.Errorf("Status = %q, want empty", result.Status)
			}
		})
	}
}

func TestParseCellValue_EmptyString(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only spaces", "   "},
		{"only tab", "\t"},
		{"mixed whitespace", " \t  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := imp.ParseCellValue(tt.input)
			if result.Amount != nil {
				t.Error("Amount should be nil for empty")
			}
			if result.Status != "" {
				t.Errorf("Status = %q, want empty", result.Status)
			}
			if result.Note != "" {
				t.Errorf("Note = %q, want empty", result.Note)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// guessCategory tests
// ---------------------------------------------------------------------------

func TestGuessCategory_Housing(t *testing.T) {
	imp := newImporter()

	for _, name := range []string{"Mortgage", "HOA Dues", "Rent", "My Mortgage Payment"} {
		cat := imp.guessCategory(name)
		if cat != "housing" {
			t.Errorf("guessCategory(%q) = %q, want %q", name, cat, "housing")
		}
	}
}

func TestGuessCategory_Utilities(t *testing.T) {
	imp := newImporter()

	for _, name := range []string{
		"AL Power", "Spire Gas", "Gas Bill", "Water", "Sewage",
		"H2O Bill", "Internet", "Trash", "Verizon", "Electric Bill",
	} {
		cat := imp.guessCategory(name)
		if cat != "utilities" {
			t.Errorf("guessCategory(%q) = %q, want %q", name, cat, "utilities")
		}
	}
}

func TestGuessCategory_Insurance(t *testing.T) {
	imp := newImporter()

	for _, name := range []string{"Home Insurance", "Life Insurance", "Insurance Premium"} {
		cat := imp.guessCategory(name)
		if cat != "insurance" {
			t.Errorf("guessCategory(%q) = %q, want %q", name, cat, "insurance")
		}
	}
}

func TestGuessCategory_Transportation(t *testing.T) {
	imp := newImporter()

	// "Car Payment" contains "car payment" → transportation
	// Note: "Car Insurance" contains both "car insurance" and "insurance" keywords.
	// Since Go maps have non-deterministic iteration, it may match either.
	// Only test the unambiguous case.
	cat := imp.guessCategory("Car Payment")
	if cat != "transportation" {
		t.Errorf("guessCategory(%q) = %q, want %q", "Car Payment", cat, "transportation")
	}
}

func TestGuessCategory_Subscriptions(t *testing.T) {
	imp := newImporter()

	for _, name := range []string{"Hulu", "Netflix", "Apple Music", "Disney+", "ESPN", "AWS"} {
		cat := imp.guessCategory(name)
		if cat != "subscriptions" {
			t.Errorf("guessCategory(%q) = %q, want %q", name, cat, "subscriptions")
		}
	}
}

func TestGuessCategory_Savings(t *testing.T) {
	imp := newImporter()

	for _, name := range []string{"Saving", "Emergency Saving", "Saving Account"} {
		cat := imp.guessCategory(name)
		if cat != "savings" {
			t.Errorf("guessCategory(%q) = %q, want %q", name, cat, "savings")
		}
	}
}

func TestGuessCategory_Debt(t *testing.T) {
	imp := newImporter()

	for _, name := range []string{"Loan", "Student Loan", "Credit Line", "Chase", "IzzCC", "Anna"} {
		cat := imp.guessCategory(name)
		if cat != "debt" {
			t.Errorf("guessCategory(%q) = %q, want %q", name, cat, "debt")
		}
	}
}

func TestGuessCategory_Personal(t *testing.T) {
	imp := newImporter()

	for _, name := range []string{"Haircut", "House Cleaning", "Pest Control", "Landscaping"} {
		cat := imp.guessCategory(name)
		if cat != "personal" {
			t.Errorf("guessCategory(%q) = %q, want %q", name, cat, "personal")
		}
	}
}

func TestGuessCategory_Other(t *testing.T) {
	imp := newImporter()

	for _, name := range []string{"Groceries", "Random Bill", "XYZ Service", "Misc", ""} {
		cat := imp.guessCategory(name)
		if cat != "other" {
			t.Errorf("guessCategory(%q) = %q, want %q", name, cat, "other")
		}
	}
}

func TestGuessCategory_CaseInsensitive(t *testing.T) {
	imp := newImporter()

	tests := []struct {
		input    string
		wantCat  string
	}{
		{"MORTGAGE", "housing"},
		{"netflix", "subscriptions"},
		{"Car Payment", "transportation"},
		{"SAVING", "savings"},
		{"hulu", "subscriptions"},
		{"VERIZON", "utilities"},
		{"Chase", "debt"},
	}

	for _, tt := range tests {
		cat := imp.guessCategory(tt.input)
		if cat != tt.wantCat {
			t.Errorf("guessCategory(%q) = %q, want %q", tt.input, cat, tt.wantCat)
		}
	}
}

// ---------------------------------------------------------------------------
// parseNumber tests
// ---------------------------------------------------------------------------

func TestParseNumber_PlainNumbers(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   float64
	}{
		{"integer", "100", 100},
		{"decimal", "99.99", 99.99},
		{"zero", "0", 0},
		{"negative", "-50", -50},
		{"large number", "999999", 999999},
		{"small decimal", "0.01", 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNumber(tt.input)
			if result == nil {
				t.Fatalf("parseNumber(%q) returned nil", tt.input)
			}
			if !almostEqual(*result, tt.want) {
				t.Errorf("parseNumber(%q) = %f, want %f", tt.input, *result, tt.want)
			}
		})
	}
}

func TestParseNumber_WithDollarSign(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{"dollar integer", "$100", 100},
		{"dollar decimal", "$99.99", 99.99},
		{"dollar with commas", "$1,234.56", 1234.56},
		{"dollar large", "$10,000", 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNumber(tt.input)
			if result == nil {
				t.Fatalf("parseNumber(%q) returned nil", tt.input)
			}
			if !almostEqual(*result, tt.want) {
				t.Errorf("parseNumber(%q) = %f, want %f", tt.input, *result, tt.want)
			}
		})
	}
}

func TestParseNumber_WithCommas(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{"one comma", "1,000", 1000},
		{"two commas", "1,000,000", 1000000},
		{"comma with decimal", "1,234.56", 1234.56},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNumber(tt.input)
			if result == nil {
				t.Fatalf("parseNumber(%q) returned nil", tt.input)
			}
			if !almostEqual(*result, tt.want) {
				t.Errorf("parseNumber(%q) = %f, want %f", tt.input, *result, tt.want)
			}
		})
	}
}

func TestParseNumber_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only spaces", "   "},
		{"text", "abc"},
		{"text with numbers", "abc123"},
		{"special chars", "!@#"},
		{"only dollar", "$"},
		{"only comma", ","},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNumber(tt.input)
			if result != nil {
				t.Errorf("parseNumber(%q) = %f, want nil", tt.input, *result)
			}
		})
	}
}

func TestParseNumber_WithWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{"leading spaces", "  100", 100},
		{"trailing spaces", "100  ", 100},
		{"both spaces", "  100  ", 100},
		{"tabs", "\t100\t", 100},
		{"dollar with spaces", "  $50  ", 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNumber(tt.input)
			if result == nil {
				t.Fatalf("parseNumber(%q) returned nil", tt.input)
			}
			if !almostEqual(*result, tt.want) {
				t.Errorf("parseNumber(%q) = %f, want %f", tt.input, *result, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewXLSXImporter test
// ---------------------------------------------------------------------------

func TestNewXLSXImporter(t *testing.T) {
	imp := NewXLSXImporter()
	if imp == nil {
		t.Fatal("NewXLSXImporter() returned nil")
	}

	// Verify all regex patterns are compiled (non-nil)
	if imp.ccWithLabel == nil {
		t.Error("ccWithLabel regex is nil")
	}
	if imp.ccSimple == nil {
		t.Error("ccSimple regex is nil")
	}
	if imp.autopayAmount == nil {
		t.Error("autopayAmount regex is nil")
	}
	if imp.autopaySimple == nil {
		t.Error("autopaySimple regex is nil")
	}
	if imp.dueEquals == nil {
		t.Error("dueEquals regex is nil")
	}
	if imp.biweeklyAmount == nil {
		t.Error("biweeklyAmount regex is nil")
	}
	if imp.simpleDay == nil {
		t.Error("simpleDay regex is nil")
	}
	if imp.citiAuto == nil {
		t.Error("citiAuto regex is nil")
	}
	if imp.dayOrdinal == nil {
		t.Error("dayOrdinal regex is nil")
	}
	if imp.paidMarker == nil {
		t.Error("paidMarker regex is nil")
	}
	if imp.deferMarker == nil {
		t.Error("deferMarker regex is nil")
	}
	if imp.uncertainMark == nil {
		t.Error("uncertainMark regex is nil")
	}
}

// ---------------------------------------------------------------------------
// Edge case & integration-style tests
// ---------------------------------------------------------------------------

func TestParseCellValue_StatusPriorityOverNumber(t *testing.T) {
	imp := newImporter()

	// Paid marker takes priority even if there is a number
	result := imp.ParseCellValue("250**paid")
	if result.Status != "paid" {
		t.Errorf("Status = %q, want %q", result.Status, "paid")
	}
	if result.Amount == nil {
		t.Fatal("Amount should not be nil when number precedes **paid")
	}
	if !almostEqual(*result.Amount, 250) {
		t.Errorf("Amount = %f, want 250", *result.Amount)
	}
}

func TestParseCellValue_DeferMarkerVariations(t *testing.T) {
	imp := newImporter()

	// The defer regex is `\|-->`, it matches the literal string |-->
	result := imp.ParseCellValue("|-->")
	if result.Status != "deferred" {
		t.Errorf("Status = %q, want %q", result.Status, "deferred")
	}

	// It should match anywhere in the string
	result2 := imp.ParseCellValue("amount |--> deferred")
	if result2.Status != "deferred" {
		t.Errorf("Status = %q, want %q", result2.Status, "deferred")
	}
}

func TestParseBillLabel_EndToEnd_RealisticLabels(t *testing.T) {
	imp := newImporter()

	// Test a comprehensive set of realistic spreadsheet labels
	type expected struct {
		name      string
		hasDay    bool
		day       int
		isAuto    bool
		hasAmt    bool
		amount    float64
		hasCC     bool
		category  string
	}

	tests := []struct {
		label string
		want  expected
	}{
		{
			label: "IzzCC - QS ***8186 :: (statement=7th, due=4th)",
			want:  expected{name: "IzzCC", hasDay: true, day: 4, hasCC: true, category: "debt"},
		},
		{
			label: "Chase :: (statement=20th, due=17th)",
			want:  expected{name: "Chase", hasDay: true, day: 17, hasCC: true, category: "debt"},
		},
		{
			label: "Saving (12th Auto - 25)",
			want:  expected{name: "Saving", hasDay: true, day: 12, isAuto: true, hasAmt: true, amount: 25, category: "savings"},
		},
		{
			label: "Verizon (16th) - Auto",
			want:  expected{name: "Verizon", hasDay: true, day: 16, isAuto: true, category: "utilities"},
		},
		{
			label: "Car Payment (due=9th)",
			want:  expected{name: "Car Payment", hasDay: true, day: 9, category: "transportation"},
		},
		{
			label: "House Cleaning ($160 bi-weekly)",
			want:  expected{name: "House Cleaning", hasAmt: true, amount: 160, category: "personal"},
		},
		{
			label: "Hulu (7th)",
			want:  expected{name: "Hulu", hasDay: true, day: 7, category: "subscriptions"},
		},
		{
			label: "Groceries",
			want:  expected{name: "Groceries", category: "other"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			bill := imp.parseBillLabel(tt.label)
			if bill == nil {
				t.Fatal("returned nil")
			}
			if bill.Name != tt.want.name {
				t.Errorf("Name = %q, want %q", bill.Name, tt.want.name)
			}
			if tt.want.hasDay {
				if bill.DueDay == nil {
					t.Fatalf("DueDay is nil, want %d", tt.want.day)
				}
				if *bill.DueDay != tt.want.day {
					t.Errorf("DueDay = %d, want %d", *bill.DueDay, tt.want.day)
				}
			} else {
				if bill.DueDay != nil {
					t.Errorf("DueDay = %d, want nil", *bill.DueDay)
				}
			}
			if bill.IsAutopay != tt.want.isAuto {
				t.Errorf("IsAutopay = %v, want %v", bill.IsAutopay, tt.want.isAuto)
			}
			if tt.want.hasAmt {
				if bill.DefaultAmt == nil {
					t.Fatalf("DefaultAmt is nil, want %f", tt.want.amount)
				}
				if !almostEqual(*bill.DefaultAmt, tt.want.amount) {
					t.Errorf("DefaultAmt = %f, want %f", *bill.DefaultAmt, tt.want.amount)
				}
			} else {
				if bill.DefaultAmt != nil {
					t.Errorf("DefaultAmt = %f, want nil", *bill.DefaultAmt)
				}
			}
			if tt.want.hasCC != (bill.CreditCard != nil) {
				t.Errorf("CreditCard present = %v, want %v", bill.CreditCard != nil, tt.want.hasCC)
			}
			if bill.Category != tt.want.category {
				t.Errorf("Category = %q, want %q", bill.Category, tt.want.category)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// guessCategory edge cases
// ---------------------------------------------------------------------------

func TestGuessCategory_PartialMatches(t *testing.T) {
	imp := newImporter()

	// "power" substring appears in "AL Power" - should match utilities
	cat := imp.guessCategory("AL Power")
	if cat != "utilities" {
		t.Errorf("guessCategory('AL Power') = %q, want 'utilities'", cat)
	}

	// "spire" should match utilities
	cat = imp.guessCategory("Spire Natural Gas")
	if cat != "utilities" {
		t.Errorf("guessCategory('Spire Natural Gas') = %q, want 'utilities'", cat)
	}
}

func TestGuessCategory_EmptyString(t *testing.T) {
	imp := newImporter()
	cat := imp.guessCategory("")
	if cat != "other" {
		t.Errorf("guessCategory('') = %q, want 'other'", cat)
	}
}

// ---------------------------------------------------------------------------
// ParseCellValue interaction with parseNumber
// ---------------------------------------------------------------------------

func TestParseCellValue_DollarAmounts(t *testing.T) {
	imp := newImporter()

	result := imp.ParseCellValue("$1,500.75")
	if result.Status != "pending" {
		t.Errorf("Status = %q, want 'pending'", result.Status)
	}
	if result.Amount == nil {
		t.Fatal("Amount is nil")
	}
	if !almostEqual(*result.Amount, 1500.75) {
		t.Errorf("Amount = %f, want 1500.75", *result.Amount)
	}
}

func TestParseCellValue_PaidWithDollarAmount(t *testing.T) {
	imp := newImporter()

	result := imp.ParseCellValue("$75.50**paid")
	if result.Status != "paid" {
		t.Errorf("Status = %q, want 'paid'", result.Status)
	}
	if result.Amount == nil {
		t.Fatal("Amount is nil")
	}
	if !almostEqual(*result.Amount, 75.50) {
		t.Errorf("Amount = %f, want 75.50", *result.Amount)
	}
}

package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ParsedBill is the result of parsing a row label from column A.
type ParsedBill struct {
	Name         string  `json:"name"`
	DueDay       *int    `json:"due_day"`
	IsAutopay    bool    `json:"is_autopay"`
	DefaultAmt   *float64 `json:"default_amount"`
	Category     string  `json:"category"`
	CreditCard   *ParsedCreditCard `json:"credit_card,omitempty"`
}

type ParsedCreditCard struct {
	CardLabel    string `json:"card_label"`
	StatementDay int    `json:"statement_day"`
	DueDay       int    `json:"due_day"`
	Issuer       string `json:"issuer"`
}

type ParsedPeriod struct {
	Month string
	Day   int
}

type ParsedCellValue struct {
	Amount  *float64
	Status  string // "paid", "deferred", "uncertain", ""
	Note    string
}

type ImportPreview struct {
	Bills       []ParsedBill  `json:"bills"`
	PeriodCount int           `json:"period_count"`
	Warnings    []string      `json:"warnings"`
}

type XLSXImporter struct {
	// Regex patterns for parsing column A bill descriptions
	ccWithLabel    *regexp.Regexp // "IssuerName - CardLabel :: (statement=Nth, due=Nth)"
	ccSimple       *regexp.Regexp // "Name :: (statement=Nth, due=Nth)"
	autopayAmount  *regexp.Regexp // "Name (Nth Auto - Amount)"
	autopaySimple  *regexp.Regexp // "Name (Nth) - Auto"
	dueEquals      *regexp.Regexp // "Name (due=Nth)"
	biweeklyAmount *regexp.Regexp // "Name ($Amount bi-weekly)"
	simpleDay      *regexp.Regexp // "Name (Nth)"
	citiAuto       *regexp.Regexp // "Name (Nth) - Auto" (at end)
	dayOrdinal     *regexp.Regexp // Nth, 1st, 2nd, 3rd, etc.
	paidMarker     *regexp.Regexp
	deferMarker    *regexp.Regexp
	uncertainMark  *regexp.Regexp
}

func NewXLSXImporter() *XLSXImporter {
	return &XLSXImporter{
		ccWithLabel:    regexp.MustCompile(`(?i)^(.+?)\s*-\s*(.+?)\s*::\s*\(statement=(\d+)\w*,?\s*due=(\d+)\w*\)`),
		ccSimple:       regexp.MustCompile(`(?i)^(.+?)\s*::\s*\(statement=(\d+)\w*,?\s*due=(\d+)\w*\)`),
		autopayAmount:  regexp.MustCompile(`(?i)^(.+?)\s*\((\d+)\w*\s+Auto\s*-\s*(\d+)\)`),
		autopaySimple:  regexp.MustCompile(`(?i)^(.+?)\s*\((\d+)\w*\)\s*-\s*Auto`),
		dueEquals:      regexp.MustCompile(`(?i)^(.+?)\s*\(due=(\d+)\w*\)`),
		biweeklyAmount: regexp.MustCompile(`(?i)^(.+?)\s*\(\$(\d+)\s*bi-?weekly\)`),
		simpleDay:      regexp.MustCompile(`(?i)^(.+?)\s*\((\d+)\w*(?:,\s*\d+\w*)?\)`),
		citiAuto:       regexp.MustCompile(`(?i)^(.+?)\s*\((\d+)\w*\)\s*-\s*Auto$`),
		dayOrdinal:     regexp.MustCompile(`(\d+)\w*`),
		paidMarker:     regexp.MustCompile(`(?i)\*\*paid`),
		deferMarker:    regexp.MustCompile(`\|-->`),
		uncertainMark:  regexp.MustCompile(`^\?\?$`),
	}
}

func (imp *XLSXImporter) ParseFile(filePath string) (*ImportPreview, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening xlsx: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in xlsx")
	}

	// Find "Budget" sheet
	sheetName := sheets[0]
	for _, s := range sheets {
		if strings.EqualFold(s, "Budget") {
			sheetName = s
			break
		}
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("reading sheet %s: %w", sheetName, err)
	}

	if len(rows) < 3 {
		return nil, fmt.Errorf("sheet has too few rows")
	}

	preview := &ImportPreview{}

	// Parse bills from column A (rows 3 onwards until we hit "Est. Pay" or "TOTAL")
	for i := 2; i < len(rows); i++ { // 0-indexed, row 3 = index 2
		if len(rows[i]) == 0 {
			continue
		}
		label := strings.TrimSpace(rows[i][0])
		if label == "" {
			continue
		}

		// Stop conditions
		labelLower := strings.ToLower(label)
		if labelLower == "est. pay" || labelLower == "total" || labelLower == "left" || labelLower == "paid" {
			continue
		}

		bill := imp.parseBillLabel(label)
		if bill != nil {
			preview.Bills = append(preview.Bills, *bill)
		}
	}

	// Count pay periods from header row (row 1, every 3 columns starting from B)
	if len(rows) > 0 {
		periodCount := 0
		for j := 1; j < len(rows[0]); j += 3 {
			if rows[0][j] != "" {
				periodCount++
			}
		}
		preview.PeriodCount = periodCount
	}

	return preview, nil
}

func (imp *XLSXImporter) parseBillLabel(label string) *ParsedBill {
	// Try credit card with label: "IzzCC - QS ***8186 :: (statement=7th, due=4th)"
	if m := imp.ccWithLabel.FindStringSubmatch(label); m != nil {
		issuer := strings.TrimSpace(m[1])
		cardLabel := strings.TrimSpace(m[2])
		stmtDay, _ := strconv.Atoi(m[3])
		dueDay, _ := strconv.Atoi(m[4])
		return &ParsedBill{
			Name:     issuer,
			DueDay:   &dueDay,
			Category: "debt",
			CreditCard: &ParsedCreditCard{
				CardLabel:    cardLabel,
				StatementDay: stmtDay,
				DueDay:       dueDay,
				Issuer:       issuer,
			},
		}
	}

	// Try simple credit card: "Chase :: (statement=20th, due=17th)"
	if m := imp.ccSimple.FindStringSubmatch(label); m != nil {
		name := strings.TrimSpace(m[1])
		stmtDay, _ := strconv.Atoi(m[2])
		dueDay, _ := strconv.Atoi(m[3])
		return &ParsedBill{
			Name:     name,
			DueDay:   &dueDay,
			Category: "debt",
			CreditCard: &ParsedCreditCard{
				StatementDay: stmtDay,
				DueDay:       dueDay,
				Issuer:       name,
			},
		}
	}

	// Try autopay with amount: "Saving (12th Auto - 25)"
	if m := imp.autopayAmount.FindStringSubmatch(label); m != nil {
		name := strings.TrimSpace(m[1])
		day, _ := strconv.Atoi(m[2])
		amount, _ := strconv.ParseFloat(m[3], 64)
		cat := imp.guessCategory(name)
		return &ParsedBill{
			Name:       name,
			DueDay:     &day,
			IsAutopay:  true,
			DefaultAmt: &amount,
			Category:   cat,
		}
	}

	// Try autopay simple: "Verizon (16th) - Auto"
	if m := imp.autopaySimple.FindStringSubmatch(label); m != nil {
		name := strings.TrimSpace(m[1])
		day, _ := strconv.Atoi(m[2])
		cat := imp.guessCategory(name)
		return &ParsedBill{
			Name:      name,
			DueDay:    &day,
			IsAutopay: true,
			Category:  cat,
		}
	}

	// Try due=Nth: "Car Payment (due=9th)"
	if m := imp.dueEquals.FindStringSubmatch(label); m != nil {
		name := strings.TrimSpace(m[1])
		day, _ := strconv.Atoi(m[2])
		cat := imp.guessCategory(name)
		return &ParsedBill{
			Name:     name,
			DueDay:   &day,
			Category: cat,
		}
	}

	// Try biweekly: "House Cleaning ($160 bi-weekly)"
	if m := imp.biweeklyAmount.FindStringSubmatch(label); m != nil {
		name := strings.TrimSpace(m[1])
		amount, _ := strconv.ParseFloat(m[2], 64)
		return &ParsedBill{
			Name:       name,
			DefaultAmt: &amount,
			Category:   "personal",
		}
	}

	// Try simple day: "Hulu (7th)" or "AL Power (7th, 21st)"
	if m := imp.simpleDay.FindStringSubmatch(label); m != nil {
		name := strings.TrimSpace(m[1])
		day, _ := strconv.Atoi(m[2])
		// Check if also has "- Auto" at the end (handled above, but just in case)
		isAuto := strings.Contains(strings.ToLower(label), "auto")
		cat := imp.guessCategory(name)
		return &ParsedBill{
			Name:      name,
			DueDay:    &day,
			IsAutopay: isAuto,
			Category:  cat,
		}
	}

	// Fallback: anything else is just a name with no parsed day
	name := strings.TrimSpace(label)
	cat := imp.guessCategory(name)
	return &ParsedBill{
		Name:     name,
		Category: cat,
	}
}

func (imp *XLSXImporter) ParseCellValue(value string) ParsedCellValue {
	value = strings.TrimSpace(value)
	if value == "" {
		return ParsedCellValue{}
	}

	// Check for status markers
	if imp.paidMarker.MatchString(value) {
		// Try to extract amount from the marker text
		numPart := imp.paidMarker.ReplaceAllString(value, "")
		numPart = strings.TrimSpace(numPart)
		amount := parseNumber(numPart)
		return ParsedCellValue{Amount: amount, Status: "paid"}
	}

	if imp.deferMarker.MatchString(value) {
		return ParsedCellValue{Status: "deferred"}
	}

	if imp.uncertainMark.MatchString(value) {
		return ParsedCellValue{Status: "uncertain"}
	}

	// Pure number
	if amount := parseNumber(value); amount != nil {
		return ParsedCellValue{Amount: amount, Status: "pending"}
	}

	// Otherwise it's a note
	return ParsedCellValue{Note: value}
}

func (imp *XLSXImporter) guessCategory(name string) string {
	lower := strings.ToLower(name)

	categories := map[string][]string{
		"housing":        {"mortgage", "hoa", "rent"},
		"utilities":      {"power", "spire", "gas", "water", "sewage", "h2o", "internet", "trash", "verizon", "electric"},
		"insurance":      {"insurance"},
		"transportation": {"car payment", "car insurance"},
		"subscriptions":  {"hulu", "netflix", "apple", "disney", "espn", "aws"},
		"savings":        {"saving"},
		"debt":           {"loan", "credit", "chase", "izzcc", "anna"},
		"personal":       {"haircut", "cleaning", "pest control", "landscaping"},
	}

	for cat, keywords := range categories {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return cat
			}
		}
	}
	return "other"
}

func parseNumber(s string) *float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return &f
	}
	return nil
}

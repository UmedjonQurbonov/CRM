package types

import "github.com/shopspring/decimal"

// Decimal is a type alias for shopspring/decimal.Decimal to represent monetary values.
type Decimal = decimal.Decimal

// NewDecimal creates a new Decimal from int64 and exponent.
var NewDecimal = decimal.New

// NewDecimalFromString parses a decimal from string.
var NewDecimalFromString = decimal.NewFromString

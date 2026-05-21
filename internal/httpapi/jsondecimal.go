package httpapi

import (
	"github.com/shopspring/decimal"
)

// JSONDecimal wraps decimal.Decimal so that response payloads emit the value
// as a JSON number (e.g. 500.5) rather than the quoted-string form that
// shopspring/decimal uses by default.
type JSONDecimal decimal.Decimal

// MarshalJSON writes the decimal as an unquoted JSON number.
func (d JSONDecimal) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(d).String()), nil
}

// UnmarshalJSON delegates to decimal.Decimal so JSONDecimal can also be used
// for request bodies; it accepts both numeric and quoted forms.
func (d *JSONDecimal) UnmarshalJSON(b []byte) error {
	var v decimal.Decimal
	if err := v.UnmarshalJSON(b); err != nil {
		return err
	}
	*d = JSONDecimal(v)
	return nil
}

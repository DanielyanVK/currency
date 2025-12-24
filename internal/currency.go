package internal

import (
	"bytes"
	"fmt"
	"strings"
)

type CurrencyCode string

func NewCurrencyCode(s string) (CurrencyCode, error) {
	ccy := CurrencyCode(strings.ToUpper(strings.TrimSpace(s)))
	if !ccy.IsSupported() {
		return "", fmt.Errorf("unsupported currency %q", s)
	}
	return ccy, nil
}

const (
	RUB CurrencyCode = "RUB"
	USD CurrencyCode = "USD"
	EUR CurrencyCode = "EUR"
	JPY CurrencyCode = "JPY"
)

var supportedSet = map[CurrencyCode]struct{}{
	RUB: {}, USD: {}, EUR: {}, JPY: {},
}

func (c *CurrencyCode) IsSupported() bool {
	_, ok := supportedSet[*c]
	return ok
}

func (c *CurrencyCode) String() string { return string(*c) }

func (c *CurrencyCode) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", c.String())), nil
}

func (c *CurrencyCode) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	s := strings.Trim(string(b), "\"")
	ccy, err := NewCurrencyCode(s)
	if err != nil {
		return err
	}
	*c = ccy
	return nil
}

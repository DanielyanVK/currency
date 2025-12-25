package internal

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

type Date struct{ time.Time }

const dateTimeLayout = "2006-01-02 15:04:05Z07"
const dateLayout = "2006-01-02"

func (d *Date) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		d.Time = time.Time{}
		return nil
	}

	s := strings.Trim(string(b), "\"")
	s = strings.TrimSpace(s)
	if s == "" {
		d.Time = time.Time{}
		return nil
	}

	t, err := time.Parse(dateLayout, s)
	if err != nil {
		t, err = time.Parse(dateTimeLayout, s)
		if err != nil {
			return fmt.Errorf("parse date %q: %w", s, err)
		}
	}

	tt := t.UTC()
	d.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, time.UTC)
	return nil
}

func (d *Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("%q", d.Time.Format(dateLayout))), nil
}

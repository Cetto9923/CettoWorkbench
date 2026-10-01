package datefmt

import (
	"testing"
	"time"
)

func TestDateAndRaw(t *testing.T) {
	zero := time.Time{}
	nilT := (*time.Time)(nil)
	if got := Date(nilT); got != Empty {
		t.Errorf("nil => %q", got)
	}
	if got := Date(&zero); got != Unset {
		t.Errorf("zero => %q", got)
	}
	d := time.Date(2026, 9, 18, 10, 30, 0, 0, time.Local)
	if got := Date(&d); got != "2026-09-18" {
		t.Errorf("valid => %q", got)
	}
	if got := Raw("0000-00-00 00:00:00"); got != Unset {
		t.Errorf("raw zero => %q", got)
	}
	if got := Raw("2026-09-18 10:30:00"); got != "2026-09-18" {
		t.Errorf("raw ts => %q", got)
	}
	if got := Raw(""); got != Empty {
		t.Errorf("raw empty => %q", got)
	}
	if got := Raw("   "); got != Empty {
		t.Errorf("raw blank => %q", got)
	}
	if got := Raw("1999-01-01"); got != Unset {
		t.Errorf("raw <2000 => %q", got)
	}
	if got := Raw("garbage"); got != Empty {
		t.Errorf("raw garbage => %q", got)
	}
}

package cli

import "testing"

func TestValidRangeSpec(t *testing.T) {
	valid := []string{"0-1023", "1024-", "-512", "0-0", "5-"}
	for _, s := range valid {
		if !validRangeSpec(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	invalid := []string{"", "-", "abc", "1-2-3", "0x10-", "-", "1.5-2", " 0-1"}
	for _, s := range invalid {
		if validRangeSpec(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}

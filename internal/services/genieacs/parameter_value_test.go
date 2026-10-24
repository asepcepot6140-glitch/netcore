package genieacs

import (
	"errors"
	"testing"
)

// ZTE F609/F660 report X_ZTE-COM_VLANEnable as xsd:unsignedInt, so a boolean from the UI must become 0/1.
func TestBooleanAsNumberFeedsUnsignedField(t *testing.T) {
	cases := map[string]uint64{"true": 1, "TRUE": 1, "1": 1, "false": 0, "0": 0}
	for raw, want := range cases {
		text, err := booleanAsNumber("enable_vlan", raw)
		if err != nil {
			t.Fatalf("%q: unexpected error %v", raw, err)
		}
		got, err := normalizeValue("enable_vlan", typeUnsigned, text)
		if err != nil {
			t.Fatalf("%q: unexpected error %v", raw, err)
		}
		if got != want {
			t.Fatalf("%q: got %v, want %v", raw, got, want)
		}
	}
}

func TestBooleanAsNumberRejectsOtherText(t *testing.T) {
	if _, err := booleanAsNumber("enable_vlan", "maybe"); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("expected ErrInvalidParameter, got %v", err)
	}
}

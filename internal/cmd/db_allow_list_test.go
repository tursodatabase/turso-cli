package cmd

import (
	"reflect"
	"testing"
)

func TestNormalizeAllowListEntries(t *testing.T) {
	got := normalizeAllowListEntries([]string{" 10.0.0.1 ", "", "10.0.0.1", "vpce-123", "  "})
	want := []string{"10.0.0.1", "vpce-123"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeAllowListEntries() = %v, want %v", got, want)
	}
}

func TestValidateAllowedIPs(t *testing.T) {
	valid := []string{"10.0.0.1", "203.0.113.7", "10.0.0.0/8", "192.168.0.0/24", "::1", "2001:db8::/32"}
	if err := validateAllowedIPs(valid); err != nil {
		t.Errorf("validateAllowedIPs(%v) = %v, want nil", valid, err)
	}

	for _, entry := range []string{"not-an-ip", "10.0.0.256", "10.0.0.0/33", "10.0.0.1/", "vpce-123"} {
		if err := validateAllowedIPs([]string{entry}); err == nil {
			t.Errorf("validateAllowedIPs(%q) = nil, want error", entry)
		}
	}
}

func TestValidateAllowedVpcIDs(t *testing.T) {
	if err := validateAllowedVpcIDs([]string{"vpce-0fe6c8807461bba49"}); err != nil {
		t.Errorf("validateAllowedVpcIDs() = %v, want nil", err)
	}

	for _, entry := range []string{"vpc-123", "vpce-", "10.0.0.1", "VPCE-123"} {
		if err := validateAllowedVpcIDs([]string{entry}); err == nil {
			t.Errorf("validateAllowedVpcIDs(%q) = nil, want error", entry)
		}
	}
}

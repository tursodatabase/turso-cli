package cmd

import (
	"fmt"
	"testing"
)

func Test_extractSemver(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"sqld sqld 0.22.5 (b924756f 2023-11-21)", "0.22.5"},
		{"sqld sqld 0.22.5 (b924756f 2023-11-21) (HEAD -> master, origin/master, origin/HEAD)", "0.22.5"},
		{"sqld sqld 1.22.5 (b924756f 2023-11-21)", "1.22.5"},
		{"sqld 1.22.5 (b924756f 2023-11-21)", "1.22.5"},
		{"sqld sqld 1.22.0 (b924756f 2023-11-21)", "1.22.0"},
		{"sqld sqld 0.23.0 (b924756f 2023-11-21)", "0.23.0"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if got := extractSemver(tt.version); got != tt.want {
				t.Errorf("extractSemver() = %v, want %v", got, tt.want)
			}
		})
	}
}

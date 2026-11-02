package flags

import (
	"reflect"
	"testing"
)

func TestParseFineGrainedPermissions(t *testing.T) {
	got, err := parseFineGrainedPermissions([]string{"all:data_read,data_add", "comments:schema_update"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []FineGrainedPermissions{
		{TableNames: nil, AllowedOperations: []string{"data_read", "data_add"}},
		{TableNames: []string{"comments"}, AllowedOperations: []string{"schema_update"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseFineGrainedPermissionsInvalid(t *testing.T) {
	for _, value := range []string{"all:data_insert", "all:data_read,", "all", "comments:"} {
		if _, err := parseFineGrainedPermissions([]string{value}); err == nil {
			t.Errorf("expected error for %q", value)
		}
	}
}

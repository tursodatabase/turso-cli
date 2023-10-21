package cmd

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func Test_setDatabasesCache(t *testing.T) {
	dbNames := []turso.Database{
		{
			Name:          "name",
			ID:            "id",
			Regions:       []string{"waw"},
			PrimaryRegion: "waw",
			Hostname:      "hostname",
		},
		{
			Name:          "name2",
			ID:            "id2",
			Regions:       []string{"waw", "ams"},
			PrimaryRegion: "ams",
			Hostname:      "hostname2",
		},
	}
	setDatabasesCache(dbNames)
	if !reflect.DeepEqual(getDatabasesCache(), dbNames) {
		t.Errorf("setDatabasesCache() = %v, want %v", getDatabasesCache(), dbNames)
	}
}

func Test_closestLocationCache(t *testing.T) {
	loc := "waw"
	setClosestLocationCache(loc)
	if got := closestLocationCache(); got != loc {
		t.Errorf("closestLocationCache() = %v, want %v", got, loc)
	}
}

func Test_setDbTokenCache(t *testing.T) {
	dbId := fmt.Sprintf("dbId-%s", uuid.NewString())
	token := fmt.Sprintf("token-%s", uuid.NewString())
	setDbTokenCache(dbId, token, time.Now().Add(10*time.Minute).Unix())
	if got := dbTokenCache(dbId); got != token {
		t.Errorf("getDbTokenCache() = %v, want %v", got, token)
	}
}

func Test_setDbTokenCache_expired(t *testing.T) {
	dbId := fmt.Sprintf("dbId-%s", uuid.NewString())
	token := fmt.Sprintf("token-%s", uuid.NewString())
	setDbTokenCache(dbId, token, time.Now().Add(-10*time.Minute).Unix())
	if got := dbTokenCache(dbId); got != "" {
		t.Errorf("getDbTokenCache() = %v", got)
	}
}

func Test_setLocationsCache(t *testing.T) {
	locs := map[string]string{
		"ams": "Amsterdam, Netherlands",
		"arn": "Stockholm, Sweden",
	}
	setLocationsCache(locs)
	if !reflect.DeepEqual(locationsCache(), locs) {
		t.Errorf("locationsCache() = %v, want %v", locationsCache(), locs)
	}
}

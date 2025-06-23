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

func Test_updateDatabasesCache(t *testing.T) {
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
	updateDatabaseCache(map[string]turso.Database{"name2": {
		Name:          "name2-upd",
		ID:            "id2-upd",
		Regions:       []string{"lhr"},
		PrimaryRegion: "ams",
		Hostname:      "hostname2-upd",
	}, "name3": {
		Name:          "name3",
		ID:            "id3",
		Regions:       []string{"aws"},
		PrimaryRegion: "waw",
		Hostname:      "hostname3",
	}})
	dbNamesUpdated := []turso.Database{
		{
			Name:          "name",
			ID:            "id",
			Regions:       []string{"waw"},
			PrimaryRegion: "waw",
			Hostname:      "hostname",
		},
		{
			Name:          "name2-upd",
			ID:            "id2-upd",
			Regions:       []string{"lhr"},
			PrimaryRegion: "ams",
			Hostname:      "hostname2-upd",
		},
		{
			Name:          "name3",
			ID:            "id3",
			Regions:       []string{"aws"},
			PrimaryRegion: "waw",
			Hostname:      "hostname3",
		},
	}
	if !reflect.DeepEqual(getDatabasesCache(), dbNamesUpdated) {
		t.Errorf("setDatabasesCache() = %v, want %v", getDatabasesCache(), dbNamesUpdated)
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

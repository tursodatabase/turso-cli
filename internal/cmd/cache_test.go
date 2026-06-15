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

// Regression test: the orgs-list cache and the per-org groups cache must not
// share a viper namespace, otherwise writing one clobbers the other (viper
// stores dotted keys as nested maps). See ORGS_CACHE_KEY.
func Test_orgsAndGroupsCacheCoexist(t *testing.T) {
	orgs := []turso.Organization{{ID: "org-id-123", Slug: "org-a", Type: "team"}}
	groups := []turso.Group{{UUID: "group-uuid-abc", Name: "group-a"}}

	assertCoexist := func(t *testing.T) {
		t.Helper()
		if got := getOrgsCache(); len(got) != 1 || got[0].ID != "org-id-123" {
			t.Errorf("orgs cache clobbered: getOrgsCache() = %#v", got)
		}
		if got := getGroupsCache("org-a"); len(got) != 1 || got[0].UUID != "group-uuid-abc" {
			t.Errorf("groups cache clobbered: getGroupsCache() = %#v", got)
		}
	}

	t.Run("orgs then groups", func(t *testing.T) {
		setOrgsCache(orgs)
		setGroupsCache("org-a", groups)
		assertCoexist(t)
	})

	t.Run("groups then orgs", func(t *testing.T) {
		setGroupsCache("org-a", groups)
		setOrgsCache(orgs)
		assertCoexist(t)
	})
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

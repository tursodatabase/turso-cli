package settings

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type Entry[T any] struct {
	Expiration int64 `json:"expiration"`
	Data       T     `json:"data"`
}

var ErrExpired = errors.New("cache entry expired")

func cacheKey(key string) string {
	return "cache." + key
}

func setCache[T any](key string, ttl int64, value T) error {
	exp := time.Now().Unix() + ttl
	return setCacheWithExp(key, exp, value)
}

func setCacheWithExp[T any](key string, exp int64, value T) error {
	entry := Entry[T]{Data: value}
	entry.Expiration = exp
	viper.Set(cacheKey(key), entry)
	settings.changed = true
	return nil
}

func getCache[T any](key string) (T, error) {
	entry := Entry[T]{}
	value := viper.Get(cacheKey(key))
	if err := mapstructure.Decode(value, &entry); err != nil {
		return entry.Data, fmt.Errorf("failed to get cache data for %s", key)
	}

	if entry.Expiration < time.Now().Unix() {
		return entry.Data, ErrExpired
	}

	return entry.Data, nil
}

func invalidateCache[T any](key string) error {
	return setCacheWithExp[T](key, 0, *new(T))
}

const DB_CACHE_KEY = "database_names"
const DB_CACHE_TTL_SECONDS = 30 * 60

// Database is exact same struct as turso.Database, but recreated to
// avoid cyclic dependency.
type Database struct {
	ID            string
	Name          string
	Regions       []string
	PrimaryRegion string
	Hostname      string
}

func (s *Settings) SetDatabasesCache(dbNames []Database) {
	setCache(DB_CACHE_KEY, DB_CACHE_TTL_SECONDS, dbNames)
}

func (s *Settings) GetDatabasesCache() []Database {
	data, err := getCache[[]Database](DB_CACHE_KEY)
	if err != nil {
		return nil
	}
	return data
}

func (s *Settings) InvalidateDatabasesCache() {
	invalidateCache[[]Database](DB_CACHE_KEY)
}

const REGIONS_CACHE_KEY = "locations"
const REGIONS_CACHE_TTL_SECONDS = 8 * 60 * 60

func (s *Settings) SetLocationsCache(locations map[string]string) {
	setCache(REGIONS_CACHE_KEY, REGIONS_CACHE_TTL_SECONDS, locations)
}

func (s *Settings) LocationsCache() map[string]string {
	locations, err := getCache[map[string]string](REGIONS_CACHE_KEY)
	if err != nil {
		return nil
	}
	return locations
}

const CLOSEST_LOCATION_CACHE_KEY = "closestLocation"

func (s *Settings) SetClosestLocationCache(closest string) {
	setCache(CLOSEST_LOCATION_CACHE_KEY, REGIONS_CACHE_TTL_SECONDS, closest)
}

func (s *Settings) ClosestLocationCache() string {
	defaultLocation, err := getCache[string](CLOSEST_LOCATION_CACHE_KEY)
	if err != nil {
		return ""
	}
	return defaultLocation
}

const TOKEN_VALID_CACHE_KEY_PREFIX = "token_valid."

func (s *Settings) SetTokenValidCache(token string, exp int64) {
	key := TOKEN_VALID_CACHE_KEY_PREFIX + strings.ReplaceAll(token, ".", "_")
	setCacheWithExp(key, exp, true)
}

func (s *Settings) TokenValidCache(token string) bool {
	key := TOKEN_VALID_CACHE_KEY_PREFIX + strings.ReplaceAll(token, ".", "_")
	ok, err := getCache[bool](key)
	return err == nil && ok
}

const DATABASE_TOKEN_KEY_PREFIX = "database_token."

func (s *Settings) SetDbTokenCache(dbID, token string, exp int64) {
	key := DATABASE_TOKEN_KEY_PREFIX + dbID
	setCacheWithExp(key, exp, token)
}

func (s *Settings) DbTokenCache(dbID string) string {
	key := DATABASE_TOKEN_KEY_PREFIX + dbID
	token, err := getCache[string](key)
	if err != nil {
		return ""
	}
	return token
}

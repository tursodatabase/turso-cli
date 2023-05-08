package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	entry := Entry[T]{Data: value}
	if ttl > 0 {
		entry.Expiration = time.Now().Unix() + ttl
	}
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

	if entry.Expiration != 0 && entry.Expiration < time.Now().Unix() {
		return entry.Data, ErrExpired
	}

	return entry.Data, nil
}

func invalidateCache(key string) error {
	configMap := viper.AllSettings()
	delete(configMap, cacheKey(key))
	encodedConfig, err := json.MarshalIndent(configMap, "", " ")
	if err != nil {
		return err
	}
	if err := viper.ReadConfig(bytes.NewReader(encodedConfig)); err != nil {
		return err
	}
	settings.changed = true
	return nil
}

const DB_NAMES_CACHE_KEY = "database_names"
const DB_NAMES_CACHE_TTL_SECONDS = 30 * 60

func (s *Settings) SetDbNamesCache(dbNames []string) {
	setCache(DB_NAMES_CACHE_KEY, DB_NAMES_CACHE_TTL_SECONDS, dbNames)
}

func (s *Settings) GetDbNamesCache() []string {
	data, err := getCache[[]string](DB_NAMES_CACHE_KEY)
	if err != nil {
		return nil
	}
	return data
}

func (s *Settings) InvalidateDbNamesCache() {
	invalidateCache(DB_NAMES_CACHE_KEY)
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

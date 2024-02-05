package settings

import (
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

func SetCacheRaw[T any](key string, value T) error {
	if _, err := ReadSettings(); err != nil {
		return err
	}
	viper.Set(cacheKey(key), value)
	settings.changed = true
	return nil
}

func ClearCache() error {
	if _, err := ReadSettings(); err != nil {
		return err
	}
	viper.Set("cache", struct{}{})
	settings.changed = true
	return nil
}

func SetCache[T any](key string, ttl int64, value T) error {
	exp := time.Now().Unix() + ttl
	return SetCacheWithExp(key, exp, value)
}

func SetCacheWithExp[T any](key string, exp int64, value T) error {
	if _, err := ReadSettings(); err != nil {
		return err
	}
	entry := Entry[T]{Data: value, Expiration: exp}
	viper.Set(cacheKey(key), entry)
	settings.changed = true
	return nil
}

func GetCache[T any](key string) (T, error) {
	entry := Entry[T]{}
	if _, err := ReadSettings(); err != nil {
		return entry.Data, err
	}
	value := viper.Get(cacheKey(key))
	if err := mapstructure.Decode(value, &entry); err != nil {
		return entry.Data, fmt.Errorf("failed to get cache data for %s", key)
	}
	if entry.Expiration < time.Now().Unix() {
		return entry.Data, ErrExpired
	}
	return entry.Data, nil
}

func InvalidateCache[T any](key string) error {
	return SetCacheWithExp[T](key, 0, *new(T))
}

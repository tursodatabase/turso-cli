package settings

import (
	"fmt"
	"os"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/kirsle/configdir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type DatabaseSettings struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"Password"`
}

type Settings struct{}

func ReadSettings() (*Settings, error) {
	configPath := configdir.LocalConfig("turso")
	configPathFlag := viper.GetString("config-path")
	if len(configPathFlag) > 0 {
		configPath = configPathFlag
	}
	err := configdir.MakePath(configPath)
	if err != nil {
		return nil, err
	}

	viper.SetConfigName("settings")
	viper.SetConfigType("json")
	viper.AddConfigPath(configPath)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Force config creation
			if err := viper.SafeWriteConfig(); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &Settings{}, nil
}

const DB_NAMES_CACHE_KEY = "cached_db_names"
const DB_NAMES_CACHE_TTL_SECONDS = 30 * 60
const DB_NAMES_CACHE_VALUE_FIELD_NAME = "db_names"

func (s *Settings) SetDbNamesCache(dbNames []string) {
	setCache(DB_NAMES_CACHE_KEY, DB_NAMES_CACHE_VALUE_FIELD_NAME, DB_NAMES_CACHE_TTL_SECONDS, dbNames)
}

func (s *Settings) GetDbNamesCache() []string {
	return getCache(DB_NAMES_CACHE_KEY, DB_NAMES_CACHE_VALUE_FIELD_NAME)
}

func (s *Settings) InvalidateDbNamesCache() {
	invalidateCache(DB_NAMES_CACHE_KEY, DB_NAMES_CACHE_VALUE_FIELD_NAME)
}

const REGIONS_CACHE_KEY = "cached_region_names"
const REGIONS_CACHE_TTL_SECONDS = 24 * 60 * 60
const REGIONS_CACHE_VALUE_FIELD_NAME = "region_names"

func (s *Settings) SetRegionsCache(regions []string) {
	setCache(REGIONS_CACHE_KEY, REGIONS_CACHE_VALUE_FIELD_NAME, REGIONS_CACHE_TTL_SECONDS, regions)
}

func (s *Settings) GetRegionsCache() []string {
	return getCache(REGIONS_CACHE_KEY, REGIONS_CACHE_VALUE_FIELD_NAME)
}

func setCache(cacheKey string, valueFieldName string, ttl int64, value []string) {
	viper.Set(cacheKey+"."+valueFieldName, value)
	viper.Set(cacheKey+".expiration_time", time.Now().Unix()+ttl)
	err := viper.WriteConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error saving settings: ", err)
	}
}

func getCache(cacheKey string, valueFieldName string) []string {
	expirationTime := viper.GetInt64(cacheKey + ".expiration_time")
	if expirationTime == 0 {
		return nil
	}
	if expirationTime <= time.Now().Unix() {
		invalidateCache(cacheKey, valueFieldName)
		return nil
	}
	return viper.GetStringSlice(cacheKey + "." + valueFieldName)
}

func invalidateCache(cacheKey string, valueFieldName string) {
	viper.Set(DB_NAMES_CACHE_KEY+".expiration_time", 0)
	viper.Set(DB_NAMES_CACHE_KEY+"."+valueFieldName, []string{})
	err := viper.WriteConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error saving settings: ", err)
	}
}

func (s *Settings) AddDatabase(id string, dbSettings *DatabaseSettings) {
	databases := viper.GetStringMap("databases")
	databases[id] = dbSettings
	viper.Set("databases", databases)
	err := viper.WriteConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error saving settings: ", err)
	}
}

func (s *Settings) DeleteDatabase(name string) {
	databases := viper.GetStringMap("databases")
	for id, rawSettings := range databases {
		settings := DatabaseSettings{}
		mapstructure.Decode(rawSettings, &settings)
		if settings.Name == name {
			delete(databases, id)
		}
	}
	err := viper.WriteConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error saving settings: ", err)
	}
}

func (s Settings) FindDatabaseByName(name string) (*DatabaseSettings, error) {

	databases := viper.GetStringMap("databases")
	for _, rawSettings := range databases {
		settings := DatabaseSettings{}
		mapstructure.Decode(rawSettings, &settings)
		if settings.Name == name {
			return &settings, nil
		}
	}
	return nil, fmt.Errorf("database %s not found. List known databases using %s", turso.Emph(name), turso.Emph("turso db list"))
}

func (s *Settings) GetDatabaseSettings(id string) *DatabaseSettings {
	databases := viper.GetStringMap("databases")
	rawSettings, ok := databases[id]
	if !ok {
		return nil
	}
	settings := DatabaseSettings{}
	mapstructure.Decode(rawSettings, &settings)
	return &settings
}

func (s *Settings) SetDatabasePassword(id string, password string) error {
	viper.Set(fmt.Sprintf("databases.%s.password", id), password)
	err := viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("error saving settings: %s", err)
	}
	return nil
}

func (s *Settings) SetToken(token string) error {
	viper.Set("token", token)
	return viper.WriteConfig()
}

func (s *Settings) GetToken() string {
	return viper.GetString("token")
}

func (s *Settings) SetUsername(username string) error {
	viper.Set("username", username)
	return viper.WriteConfig()
}

func (s *Settings) GetUsername() string {
	return viper.GetString("username")
}

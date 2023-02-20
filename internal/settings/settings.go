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

type DbNamesCache struct {
	ExpirationTime int64    `json:"expiration_time"`
	DbNames        []string `json:"db_names"`
}

type DatabaseSettings struct {
	Name     string  `json:"name"`
	Host     string  `json:"host"`
	Hostname *string `json:"hostname"`
	Username string  `json:"username"`
	Password string  `json:"Password"`
}

func (s *DatabaseSettings) GetURL() string {
	var hostname string
	if s.Hostname != nil {
		hostname = *s.Hostname
	} else {
		hostname = s.Host
	}
	return fmt.Sprintf("https://%s:%s@%s", s.Username, s.Password, hostname)
}

type Settings struct{}

func ReadSettings() (*Settings, error) {
	configPath := configdir.LocalConfig("turso")
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

func (s *Settings) SetDbNamesCache(dbNames []string) {
	viper.Set(DB_NAMES_CACHE_KEY, DbNamesCache{time.Now().Unix() + DB_NAMES_CACHE_TTL_SECONDS, dbNames})
	err := viper.WriteConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error saving settings: ", err)
	}
}

func (s *Settings) GetDbNamesCache() []string {
	expirationTime := viper.GetInt64(DB_NAMES_CACHE_KEY + ".expiration_time")
	if expirationTime == 0 {
		return nil
	}
	if expirationTime <= time.Now().Unix() {
		s.InvalidateDbNamesCache()
		return nil
	}
	return viper.GetStringSlice(DB_NAMES_CACHE_KEY + ".db_names")
}

func (s *Settings) InvalidateDbNamesCache() {
	viper.Set(DB_NAMES_CACHE_KEY, DbNamesCache{0, []string{}})
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

func (s *Settings) SetToken(token string) error {
	viper.Set("token", token)
	return viper.WriteConfig()
}

func (s *Settings) GetToken() string {
	return viper.GetString("token")
}

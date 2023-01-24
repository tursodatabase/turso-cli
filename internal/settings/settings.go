package settings

import (
	"fmt"

	"github.com/kirsle/configdir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type DatabaseSettings struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"Password"`
}

func (s *DatabaseSettings) GetURL() string {
	return fmt.Sprintf("http://%s:%s@%s", s.Username, s.Password, s.Host)
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
			// Config file not found; ignore error because one will be created if necessary later.
		} else {
			return nil, err
		}
	}
	return &Settings{}, nil
}

func (s *Settings) AddDatabase(name string, dbSettings *DatabaseSettings) {
	databases := viper.GetStringMap("databases")
	databases[name] = dbSettings
	viper.Set("databases", databases)
	viper.WriteConfig()
}

func (s *Settings) GetDatabaseSettings(name string) *DatabaseSettings {
	databases := viper.GetStringMap("databases")
	rawSettings, ok := databases[name]
	if !ok {
		return nil
	}
	settings := DatabaseSettings{}
	mapstructure.Decode(rawSettings, &settings)
	return &settings
}

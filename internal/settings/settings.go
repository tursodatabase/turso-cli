package settings

import (
	"fmt"

	"github.com/kirsle/configdir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type DatabaseSettings struct {
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
	return fmt.Sprintf("http://%s:%s@%s", s.Username, s.Password, hostname)
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
			viper.SafeWriteConfig()
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

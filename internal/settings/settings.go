package settings

import (
	"errors"

	"github.com/kirsle/configdir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type DatabaseSettings struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"Password"`
}

type Settings struct {
	changed bool
}

var settings *Settings

func ReadSettings() (*Settings, error) {
	if settings != nil {
		return settings, nil
	}

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

	settings = &Settings{}
	return settings, nil
}

func PersistChanges() {
	if settings.changed {
		viper.WriteConfig()
	}
}

func (s *Settings) AddDatabase(id string, dbSettings *DatabaseSettings) {
	databases := viper.GetStringMap("databases")
	databases[id] = dbSettings
	viper.Set("databases", databases)
	s.changed = true
}

func (s *Settings) RegisterUse(cmd string) bool {
	commands := viper.GetStringMap("usedCommands")
	firstTime := true
	if used, ok := commands[cmd].(bool); ok {
		firstTime = !used
	}
	commands[cmd] = true
	viper.Set("usedCommands", commands)
	s.changed = true
	return firstTime
}

func (s *Settings) SetOrganization(org string) {
	viper.Set("organization", org)
	s.changed = true
}

func (s *Settings) Organization() string {
	return viper.GetString("organization")
}

func (s *Settings) DeleteDatabase(name string) {
	databases := viper.GetStringMap("databases")
	for id, rawSettings := range databases {
		settings := DatabaseSettings{}
		mapstructure.Decode(rawSettings, &settings)
		if settings.Name == name {
			delete(databases, id)
			s.changed = true
		}
	}
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
	databases := viper.GetStringMap("databases")
	rawSettings, ok := databases[id]
	if !ok {
		return errors.New("database not found")
	}
	settings := DatabaseSettings{}
	mapstructure.Decode(rawSettings, &settings)
	settings.Password = password
	databases[id] = settings
	viper.Set("databases", databases)
	s.changed = true
	return nil
}

func (s *Settings) SetToken(token string) {
	viper.Set("token", token)
	s.changed = true
}

func (s *Settings) GetToken() string {
	return viper.GetString("token")
}

func (s *Settings) SetUsername(username string) {
	viper.Set("username", username)
	s.changed = true
}

func (s *Settings) GetUsername() string {
	return viper.GetString("username")
}

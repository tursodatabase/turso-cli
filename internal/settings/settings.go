package settings

import (
	"sync"

	"github.com/kirsle/configdir"
	"github.com/spf13/viper"
)

type Settings struct {
	changed bool
}

var settings *Settings
var mu sync.Mutex

func ReadSettings() (*Settings, error) {
	mu.Lock()
	defer mu.Unlock()

	if settings != nil {
		return settings, nil
	}
	settings = &Settings{}

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

	return settings, nil
}

func PersistChanges() {
	if settings != nil && settings.changed {
		viper.WriteConfig()
	}
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

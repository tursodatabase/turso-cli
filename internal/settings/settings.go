package settings

import (
	"fmt"
	"path"
	"path/filepath"
	"sync"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/flags"
	"github.com/kirsle/configdir"
	"github.com/spf13/viper"
)

type Settings struct {
	changed bool
}

var (
	settings *Settings
	mu       sync.Mutex
)

func ReadSettings() (*Settings, error) {
	mu.Lock()
	defer mu.Unlock()

	if settings != nil {
		return settings, nil
	}
	settings = &Settings{}

	configPath := configdir.LocalConfig("turso")
	viper.BindEnv("config-path", "TURSO_CONFIG_FOLDER")

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
	configFile := path.Join(configPath, "settings.json")
	if abs, err := filepath.Abs(configFile); err == nil {
		configFile = abs
	}

	if err := viper.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			// Force config creation
			if err := viper.SafeWriteConfig(); err != nil {
				return nil, err
			}
		case viper.ConfigParseError:
			if flags.ResetConfig() {
				viper.WriteConfig()
				break
			}
			warning := internal.Warn("Warning")
			flag := internal.Emph("--reset-config")
			fmt.Printf("%s: could not parse JSON config from file %s\n", warning, internal.Emph(configFile))
			fmt.Printf("Fix the syntax errors on the file, or use the %s flag to replace it with a fresh one.\n", flag)
			return nil, err
		default:
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

func (s *Settings) SetAutoupdate(autoupdate bool) {
	viper.Set("autoupdate", autoupdate)
	s.changed = true
}

func (s *Settings) SetLastUpdateCheck(t int64) {
	viper.Set("lastUpdateCheck", t)
	s.changed = true
}

func (s *Settings) GetLastUpdateCheck() int64 {
	return viper.GetInt64("lastUpdateCheck")
}

func (s *Settings) GetAutoupdate() bool {
	return viper.GetBool("autoupdate")
}

func (s *Settings) GetUsername() string {
	return viper.GetString("username")
}

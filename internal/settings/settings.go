package settings

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/kirsle/configdir"
	"github.com/spf13/viper"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/flags"
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
	viper.BindEnv("baseURL", "TURSO_API_BASEURL")

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
			fmt.Printf("E.g. turso auth login --reset-config\n")
			return nil, err
		default:
			return nil, err
		}
	}

	return settings, nil
}

func Path() string {
	return viper.ConfigFileUsed()
}

func PersistChanges() {
	if settings == nil || !settings.changed {
		return
	}

	if err := TryToPersistChanges(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

func TryToPersistChanges() error {
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to persist turso settings file: %w", err)
	}
	return nil
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

func (s *Settings) GetBaseURL() string {
	return viper.GetString("baseURL")
}

func (s *Settings) SetAutoupdate(autoupdate string) {
	config := viper.GetStringMap("config")
	if config == nil {
		config = make(map[string]interface{})
	}
	config["autoupdate"] = autoupdate
	viper.Set("config", config)
	s.changed = true
}

func (s *Settings) SetLastUpdateCheck(t int64) {
	config := viper.GetStringMap("config")
	if config == nil {
		config = make(map[string]interface{})
	}
	config["last_update_check"] = t
	viper.Set("config", config)
	s.changed = true
}

func (s *Settings) GetLastUpdateCheck() int64 {
	config := viper.GetStringMap("config")
	if config == nil || config["last_update_check"] == nil {
		return 0
	}
	lastUpdateCheck, ok := config["last_update_check"]
	if !ok {
		return 0
	}

	switch lastUpdateCheck := lastUpdateCheck.(type) {
	case float64:
		return int64(lastUpdateCheck)
	case int64:
		return lastUpdateCheck
	default:
		return 0
	}
}

func (s *Settings) GetAutoupdate() string {
	config := viper.GetStringMap("config")
	if config == nil || config["autoupdate"] == nil || config["autoupdate"] == "" {
		return "on"
	}
	value := config["autoupdate"]
	return value.(string)
}

// Copyright (c) The Tellor Authors.
// Licensed under the MIT License.

package util

type LogConfig struct {
	levels map[string]LogLevel
}

var (
	sharedConfig *LogConfig
)

// ParseLoggingConfig parses the given JSON log level config file for use in log configuration.
func SetupLoggingConfig(entries map[string]string) error {

	cfg := &LogConfig{make(map[string]LogLevel)}
	for component, level := range entries {
		lvl, err := StringToLevel(level)
		if err != nil {
			return err
		}
		cfg.levels[component] = lvl
	}
	sharedConfig = cfg

	// Initialize all the loggers that have already been declared as global vars.
	initLoggers(sharedConfig)
	return nil
}

// GetLoggingConfig retrieves a shared logging config.
func GetLoggingConfig() *LogConfig {
	return sharedConfig
}

// GetLevel the log level.
func (cfg *LogConfig) GetLevel(pkg string, component string) LogLevel {
	key := pkg + "." + component
	lvl := cfg.levels[key]
	if lvl == 0 {
		return InfoLogLevel
	}
	return lvl
}

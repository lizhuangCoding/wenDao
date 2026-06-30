package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var defaultConfigPaths = []string{
	"./config",
	"../config",
}

type Loader struct {
	v *viper.Viper
}

func NewLoader(paths ...string) *Loader {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if len(paths) == 0 {
		paths = defaultConfigPaths
	}
	for _, path := range paths {
		if strings.TrimSpace(path) != "" {
			v.AddConfigPath(path)
		}
	}

	return &Loader{v: v}
}

// LoadConfig 加载默认路径中的配置文件。
func LoadConfig() (*Config, error) {
	return NewLoader().Load()
}

// LoadConfigFromPaths 按给定目录顺序查找并加载配置文件。
func LoadConfigFromPaths(paths ...string) (*Config, error) {
	return NewLoader(paths...).Load()
}

func (l *Loader) Load() (*Config, error) {
	if l == nil || l.v == nil {
		return nil, fmt.Errorf("config loader is required")
	}
	if err := registerConfigSections(l.v); err != nil {
		return nil, err
	}
	if err := l.v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := l.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	normalizeConfig(&cfg)
	if cfg.JWT.Secret == placeholderJWTSecret {
		return nil, fmt.Errorf("invalid placeholder JWT secret: configure a non-placeholder JWT secret before startup")
	}
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

package config

import (
	"os"
	"fmt"
	"log/slog"
	"path/filepath"
)

import (
	"gopkg.in/yaml.v2"
)

var (
	LOG_LEVELS = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
)

func Load(path string) (*Config, error) {
	c_path := filepath.Clean(path)

	b, err := os.ReadFile(c_path)
	if err != nil {
		return nil, err
	}

	if len(b) < 1{
		return nil, fmt.Errorf("file is empty: %s", path)
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.check(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Config struct {
	Database *Database `yaml:"database"`
	Http     *Http     `yaml:"http"`
	Feed     *Feed     `yaml:"feed"`
	Log      *Log      `yaml:"log"`
}

func (self *Config) check() error {
	if err := self.Database.check(); err != nil{
		return err
	}
	if err := self.Http.check(); err != nil {
		return err
	}
	if err := self.Feed.check(); err != nil {
		return err
	}
	if err := self.Log.check(); err != nil {
		return err
	}
	return nil
}

type Log struct {
	Level string `yaml:"level"`
}

func (self *Log) check() error {
	if self.Level == "" {
		return fmt.Errorf("Log.Level is Empty")
	}
	if _, ok := LOG_LEVELS[self.Level]; !ok {
		return fmt.Errorf("unknown log level: '%s'", self.Level)
	}
	return nil
}

func (self *Log) GetSlogLevel() slog.Level {
	return LOG_LEVELS[self.Level]
}

type Database struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	DB   string `yaml:"database"`
	User string `yaml:"user"`
	Pass string `yaml:"password"`
}

func (self *Database) check() error {
	if self.Host == "" {
		return fmt.Errorf("host is empty.")
	}
	if 0 >= self.Port || self.Port > 65535 {
		return fmt.Errorf("http port number out of range.")
	}
	if self.DB == "" {
		return fmt.Errorf("database is empty.")
	}
	if self.User == "" {
		return fmt.Errorf("user is empty.")
	}
	if self.Pass == "" {
		return fmt.Errorf("pass is empty.")
	}
	return nil
}

type Http struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Root string `yaml:"app_root"`
}

func (self *Http) check() error {
	if 0 >= self.Port || self.Port > 65535 {
		return fmt.Errorf("http port number out of range.")
	}
	return nil
}

func (self *Http) GetAddr() string {
	return fmt.Sprintf("%s:%d", self.Host, self.Port)
}

type Feed struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Link        string `yaml:"link"`
	AuthorName  string `yaml:"author_name"`
	AuthorEmail string `yaml:"author_email"`

	DefaultType string `yaml:"default_type"`
}

func (self *Feed) check() error {
	if self.DefaultType == "rss" {
		return nil
	}
	if self.DefaultType == "atom" {
		return nil
	}
	if self.DefaultType == "json" {
		return nil
	}
	return fmt.Errorf("Feed.Default: unsupported type: %s", self.DefaultType)
}

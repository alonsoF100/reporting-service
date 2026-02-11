package config

import "time"

type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Logger      LoggerConfig      `mapstructure:"logger"`
	Migration   MigrationsConfig  `mapstructure:"migrations"`
	Application ApplicationConfig `mapstructure:"application"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type LoggerConfig struct {
	Level string `mapstructure:"level"`
	JSON  bool   `mapstructure:"json"`
}

type MigrationsConfig struct {
	Dir string `mapstructure:"dir"`
}

type ApplicationConfig struct {
	Input      string        `mapstructure:"input_dir"`
	Output     string        `mapstructure:"output_dir"`
	Period     time.Duration `mapstructure:"scan_period"`
	QueueSize  int           `mapstructure:"queue_size"`
	Workers    int           `mapstructure:"workers"`
	MaxRetries int           `mapstructure:"max_retries"`
}

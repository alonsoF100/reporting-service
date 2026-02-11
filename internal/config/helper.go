package config

import "fmt"

func (cfg ServerConfig) PortStr() string {
	return fmt.Sprintf(":%d", cfg.Port)
}

func (cfg *DatabaseConfig) ConStr() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)
}

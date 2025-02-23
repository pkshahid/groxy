package utils

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port int `mapstructure:"port"`
		TLS  struct {
			Enabled  bool   `mapstructure:"enabled"`
			CertFile string `mapstructure:"cert_file"`
			KeyFile  string `mapstructure:"key_file"`
		} `mapstructure:"tls"`
	} `mapstructure:"server"`
	LoadBalancer struct {
		Strategy string   `mapstructure:"strategy"`
		Backends []string `mapstructure:"backends"`
	} `mapstructure:"load_balancer"`
}

// LoadConfig loads config from YAML
func LoadConfig() *Config {
	var config Config
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv() // Enable reading from environment variables

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Error unmarshaling config: %v", err)
	}

	return &config
}

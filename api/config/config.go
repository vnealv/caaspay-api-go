package config

import (
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	MetricsEnabled bool
	DatadogAddr    string
	LogLevel       string
	Env            string
	Port           int
	Host           string
	RPCTimeout     time.Duration

	Redis   RedisConfig
	RPCPool RPCPoolConfig
	JWT     JWTConfig
	OAuth   OAuthConfig
}

type RedisConfig struct {
	IsCluster bool
	Prefix    string
	Address   []string
}

type RPCPoolConfig struct {
	InitialClients       int
	MaxClients           int
	MaxRequestsPerClient int
	MonitorInterval      time.Duration
	ScaleDown            bool
}

type JWTConfig struct {
	TokenExpiry time.Duration
	JWTSecret   string
}

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Endpoint     OAuthEndpoint
}

type OAuthEndpoint struct {
	AuthURL  string
	TokenURL string
}

func LoadAPIConfig() (*Config, error) {
	viper.SetConfigName("api")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Apply defaults if not set in the YAML file
	if config.Host == "" {
		config.Host = "127.0.0.1"
	}
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.RPCTimeout == 0 {
		config.RPCTimeout = 60 * time.Second
	}
	if config.RPCPool.InitialClients == 0 {
		config.RPCPool.InitialClients = 4
	}
	if config.RPCPool.MaxClients == 0 {
		config.RPCPool.MaxClients = 20
	}
	if config.RPCPool.MaxRequestsPerClient == 0 {
		config.RPCPool.MaxRequestsPerClient = 10
	}
	if config.RPCPool.MonitorInterval == 0 {
		config.RPCPool.MonitorInterval = 15 * time.Second
	}
	if config.JWT.TokenExpiry == 0 {
		config.JWT.TokenExpiry = 30 * time.Minute
	}
	if config.Redis.Prefix == "" {
		config.Redis.Prefix = "myriad"
	}

	return &config, nil
}

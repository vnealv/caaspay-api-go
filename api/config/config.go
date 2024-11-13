package config

import (
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	MetricsEnabled        bool            `mapstructure:"metrics_enabled"`
	DatadogAddr           string          `mapstructure:"datadog_addr"`
	LogLevel              string          `mapstructure:"log_level"`
	Env                   string          `mapstructure:"env"`
	Port                  int             `mapstructure:"port"`
	Host                  string          `mapstructure:"host"`
	RPCTimeout            time.Duration   `mapstructure:"rpc_timeout"`
	TrustedProxies        []string        `mapstructure:"trusted_proxies"`
	RateLimit             RateLimitConfig `mapstructure:"rate_limit"`
	StatusRouteEnabled    bool            `mapstructure:"status_route_enabled"`
	HealthRouteEnabled    bool            `mapstructure:"health_route_enabled"`
	SelfJWTEnabled        bool            `mapstructure:"self_jwt_enabled"`
	EnableSecurityHeaders bool            `mapstructure:"enable_security_headers"`
	EnableCloudflare      bool            `mapstructure:"enable_cloudflare"`
	EnableCORS            bool            `mapstructure:"enable_cors"`
	EnableRBAC            bool            `mapstructure:"enable_rbac"`
	EnableOpenapiSwagger  bool            `mapstructure:"enable_openapi_swagger"`
	TrustedOrigins        []string        `mapstructure:"trusted_origins"`

	Redis   RedisConfig   `mapstructure:"redis"`
	RPCPool RPCPoolConfig `mapstructure:"rpc_pool"`
	JWT     JWTConfig     `mapstructure:"jwt"`
	OAuth   OAuthConfig   `mapstructure:"oauth"`
}

type RedisConfig struct {
	IsCluster bool     `mapstructure:"is_cluster"`
	Prefix    string   `mapstructure:"prefix"`
	Address   []string `mapstructure:"address"`
}

type RPCPoolConfig struct {
	InitialClients       int           `mapstructure:"initial_clients"`
	MaxClients           int           `mapstructure:"max_clients"`
	MaxRequestsPerClient int           `mapstructure:"max_requests_per_client"`
	MonitorInterval      time.Duration `mapstructure:"monitor_interval"`
	ScaleDown            bool          `mapstructure:"scale_down"`
}

type JWTConfig struct {
	TokenExpiry time.Duration `mapstructure:"token_expiry"`
	JWTSecret   string        `mapstructure:"jwt_secret"`
}

type OAuthConfig struct {
	ClientID     string        `mapstructure:"client_id"`
	ClientSecret string        `mapstructure:"client_secret"`
	RedirectURL  string        `mapstructure:"redirect_url"`
	Endpoint     OAuthEndpoint `mapstructure:"endpoint"`
}

type OAuthEndpoint struct {
	AuthURL  string `mapstructure:"auth_url"`
	TokenURL string `mapstructure:"token_url"`
}

type RateLimitConfig struct {
	Enabled      bool `mapstructure:"enabled"`
	DefaultLimit int  `mapstructure:"default_limit"`
	DefaultBurst int  `mapstructure:"default_burst"`
}

func LoadAPIConfig() (*Config, error) {
	viper.SetConfigName("api")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
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
	if config.RateLimit.DefaultLimit == 0 {
		config.RateLimit.DefaultLimit = 5
	}
	if config.RateLimit.DefaultBurst == 0 {
		config.RateLimit.DefaultBurst = 10
	}

	return &config, nil
}

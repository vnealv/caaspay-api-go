package config

import (
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	AppName               string          `mapstructure:"app_name"`
	API_Title             string          `mapstructure:"api_title"`
	API_Description       string          `mapstructure:"api_description"`
	API_Version           string          `mapstructure:"api_version"`
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

	Redis         RedisConfig         `mapstructure:"redis"`
	RPCPool       RPCPoolConfig       `mapstructure:"rpc_pool"`
	JWT           JWTConfig           `mapstructure:"jwt"`
	OAuth         OAuthConfig         `mapstructure:"oauth"`
	JWTCloudflare JWTCloudflareConfig `mapstructure:"jwt_cloudflare"`
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
	TokenExpiry        time.Duration `mapstructure:"token_expiry"`
	JWTSecret          string        `mapstructure:"jwt_secret"`
	TokenRenewalWindow time.Duration `mapstructure:"token_renewal_window"`
	AllowedUsers       []AllowedUser `mapstructure:"allowed_users"`
}

type JWTCloudflareConfig struct {
	PublicKeyURL  string        `mapstructure:"public_key_url"`
	Issuer        string        `mapstructure:"issuer"`
	CacheDuration time.Duration `mapstructure:"cache_duration"`
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

type AllowedUser struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Role     string `mapstructure:"role"`
}

func LoadAPIConfig() (*Config, error) {
	viper.AddConfigPath("./config")
	viper.SetConfigType("yaml")

	// Set environment variable prefix to "GOAPI" for consistency
	viper.SetEnvPrefix("GOAPI")
	viper.AutomaticEnv()

	// Load main API config
	viper.SetConfigName("api")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading api.yaml: %w", err)
	}

	// Initialize config struct with loaded values
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshalling api config: %w", err)
	}

	// Load credentials if available and overlay on top of the API config
	viper.SetConfigName("credentials")
	if err := viper.MergeInConfig(); err == nil {
		if err := viper.Unmarshal(&config); err != nil {
			return nil, fmt.Errorf("error unmarshalling credentials config: %w", err)
		}
	}

	// Map environment variables to struct fields, giving them precedence
	bindEnvironmentVariables()
	// Reload into config to override with environment variables
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshalling final config: %w", err)
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
	if config.JWT.TokenRenewalWindow == 0 {
		config.JWT.TokenRenewalWindow = 15 * time.Minute
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
	if config.JWTCloudflare.CacheDuration == 0 {
		config.JWTCloudflare.CacheDuration = time.Hour
	}

	return &config, nil
}

func bindEnvironmentVariables() {
	// Map specific environment variables to config fields
	viper.BindEnv("metrics_enabled", "GOAPI_METRICS_ENABLED")
	viper.BindEnv("datadog_addr", "GOAPI_DATADOG_ADDR")
	viper.BindEnv("log_level", "GOAPI_LOG_LEVEL")
	viper.BindEnv("env", "GOAPI_ENV")
	viper.BindEnv("port", "GOAPI_PORT")
	viper.BindEnv("host", "GOAPI_HOST")
	viper.BindEnv("rpc_timeout", "GOAPI_RPC_TIMEOUT")
	viper.BindEnv("trusted_proxies", "GOAPI_TRUSTED_PROXIES")
	viper.BindEnv("status_route_enabled", "GOAPI_STATUS_ROUTE_ENABLED")
	viper.BindEnv("health_route_enabled", "GOAPI_HEALTH_ROUTE_ENABLED")
	viper.BindEnv("self_jwt_enabled", "GOAPI_SELF_JWT_ENABLED")
	viper.BindEnv("enable_security_headers", "GOAPI_ENABLE_SECURITY_HEADERS")
	viper.BindEnv("enable_cloudflare", "GOAPI_ENABLE_CLOUDFLARE")
	viper.BindEnv("enable_cors", "GOAPI_ENABLE_CORS")
	viper.BindEnv("enable_rbac", "GOAPI_ENABLE_RBAC")
	viper.BindEnv("enable_openapi_swagger", "GOAPI_ENABLE_OPENAPI_SWAGGER")
	viper.BindEnv("redis.is_cluster", "GOAPI_REDIS_IS_CLUSTER")
	viper.BindEnv("redis.prefix", "GOAPI_REDIS_PREFIX")
	viper.BindEnv("redis.address", "GOAPI_REDIS_ADDRESS")
	viper.BindEnv("redis.initial_clients", "GOAPI_REDIS_INITIAL_CLIENTS")
	viper.BindEnv("redis.max_clients", "GOAPI_REDIS_MAX_CLIENTS")
	viper.BindEnv("redis.max_requests_per_client", "GOAPI_REDIS_MAX_REQUESTS_PER_CLIENT")
	viper.BindEnv("jwt.token_expiry", "GOAPI_JWT_TOKEN_EXPIRY")
	viper.BindEnv("jwt.jwt_secret", "GOAPI_JWT_SECRET")
	viper.BindEnv("jwt_cloudflare.public_key_url", "GOAPI_JWT_CLOUDFLARE_PUBLIC_KEY_URL")
	viper.BindEnv("jwt_cloudflare.issuer", "GOAPI_JWT_CLOUDFLARE_ISSUER")
}

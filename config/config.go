package config

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerConfig    ServerConfig
	AuthConfig      AuthConfig
	OCRConfig       OCRConfig
	EmbeddingConfig EmbeddingConfig
	OIDCProvider    OIDCProvider
	Webservice      Webservice
	MongoDB         MongoDB
}

type MongoDB struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

type ServerConfig struct {
	App       string
	Port      string
	Proxy     string
	BaseUrl   string
	SecretKey string
}

type AuthConfig struct {
	Username       string
	Password       string
	PrivateKeySeed string
	ExpiresIn      time.Duration
}

type OCRConfig struct {
	APIURL    string
	APIKey    string
	APIHeader string
	IPs       []string
}

type EmbeddingConfig struct {
	VectorsFile  string
	OpenaiApiKey string
}

type OIDCProvider struct {
	Authority        string
	ExpectedAudience string
	RequiredScopes   string
}

type Webservice struct {
	HeaderKey  string
	ApiKey     string
	AllowedIPs []string
}

var (
	cfg  *Config
	once sync.Once
)

func InitConfig() *Config {
	once.Do(func() {
		viper.SetConfigFile(".env")
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Error reading config file: %v", err)
		}

		server := &ServerConfig{
			App:       viper.GetString("APP"),
			Port:      viper.GetString("PORT"),
			Proxy:     viper.GetString("PROXY_URL"),
			BaseUrl:   viper.GetString("BASE_URL"),
			SecretKey: viper.GetString("SECRET_KEY"),
		}

		mongoDB := &MongoDB{
			Host:     viper.GetString("MONGODB_HOST"),
			Port:     viper.GetInt("MONGODB_PORT"),
			Username: viper.GetString("MONGODB_USERNAME"),
			Password: viper.GetString("MONGODB_PASSWORD"),
			Database: viper.GetString("MONGODB_DATABASE"),
		}

		auth := &AuthConfig{
			Username:       viper.GetString("USERNAME"),
			Password:       viper.GetString("PASSWORD"),
			PrivateKeySeed: viper.GetString("PRIVATE_KEY_SEED"),
			ExpiresIn:      viper.GetDuration("EXPIRES_IN"),
		}

		ocr := &OCRConfig{
			APIURL:    viper.GetString("OCR_API_URL"),
			APIKey:    viper.GetString("OCR_API_KEY"),
			APIHeader: viper.GetString("OCR_API_HEADER"),
			IPs:       viper.GetStringSlice("OCR_IPS"),
		}

		embedding := &EmbeddingConfig{
			VectorsFile:  viper.GetString("VECTORS_FILE"),
			OpenaiApiKey: viper.GetString("OPENAI_API_KEY"),
		}

		oidc := &OIDCProvider{
			Authority:        viper.GetString("OIDC_AUTHORITY"),
			ExpectedAudience: viper.GetString("OIDC_EXPECTED_AUDIENCE"),
			RequiredScopes:   viper.GetString("OIDC_REQUIRED_SCOPES"),
		}

		Webservice := &Webservice{
			HeaderKey:  viper.GetString("WEBSERVICE_HEADER_KEY"),
			ApiKey:     viper.GetString("WEBSERVICE_API_KEY"),
			AllowedIPs: viper.GetStringSlice("WEBSERVICE_ALLOWED_IPS"),
		}

		cfg = &Config{
			ServerConfig:    *server,
			AuthConfig:      *auth,
			OCRConfig:       *ocr,
			EmbeddingConfig: *embedding,
			OIDCProvider:    *oidc,
			Webservice:      *Webservice,
			MongoDB:         *mongoDB,
		}

		fmt.Println("Config initialized successfully")
	})

	return cfg
}

func GetConfig() *Config {
	if cfg == nil {
		InitConfig()
	}

	return cfg
}

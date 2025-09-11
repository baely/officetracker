package config

import (
	"github.com/kelseyhightower/envconfig"
)

type AppConfigurer interface {
	GetApp() App
}

type IntegratedApp struct {
	App        App      `envconfig:"APP"`
	Domain     Domain   `envconfig:"DOMAIN"`
	Postgres   Postgres `envconfig:"POSTGRES"`
	Redis      Redis    `envconfig:"REDIS"`
	Github     Github   `envconfig:"GITHUB"`
	Auth0      Auth0    `envconfig:"AUTH0"`
	SigningKey string   `envconfig:"SIGNING_KEY"`
}

func (a IntegratedApp) GetApp() App {
	return a.App
}

type StandaloneApp struct {
	App    App    `envconfig:"APP"`
	SQLite SQLite `envconfig:"SQLITE"`
}

func (a StandaloneApp) GetApp() App {
	return a.App
}

type App struct {
	Env  string `envconfig:"ENV"`
	Port string `envconfig:"PORT"`
	Demo bool   `envconfig:"DEMO"`
}

type Domain struct {
	Protocol  string `envconfig:"PROTOCOL"`
	Subdomain string `envconfig:"SUBDOMAIN"`
	Domain    string `envconfig:"DOMAIN"`
	BasePath  string `envconfig:"BASE_PATH"`
}

type SQLite struct {
	Location string `envconfig:"LOCATION"`
}

type Postgres struct {
	Host     string `envconfig:"HOST"`
	Port     string `envconfig:"PORT"`
	User     string `envconfig:"USER"`
	Password string `envconfig:"PASSWORD"`
	DBName   string `envconfig:"DBNAME"`
}

type Redis struct {
	Host     string `envconfig:"HOST"`
	Username string `envconfig:"USERNAME"`
	Password string `envconfig:"PASSWORD"`
	DB       int    `envconfig:"DB"`
}

type Github struct {
	ClientID string `envconfig:"CLIENT_ID"`
	Secret   string `envconfig:"SECRET"`
}

type Auth0 struct {
	Domain       string `envconfig:"DOMAIN"`
	ClientID     string `envconfig:"CLIENT_ID"`
	ClientSecret string `envconfig:"CLIENT_SECRET"`
}

func LoadIntegratedApp() (IntegratedApp, error) {
	var cfg IntegratedApp
	err := envconfig.Process("", &cfg)
	if err != nil {
		return IntegratedApp{}, err
	}
	return cfg, nil
}

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
	Github     Github   `envconfig:"GITHUB"`
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

type Firestore struct {
	ProjectID    string `envconfig:"PROJECT_ID"`
	CollectionID string `envconfig:"COLLECTION_ID"`
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

type Github struct {
	ClientID string `envconfig:"CLIENT_ID"`
	Secret   string `envconfig:"SECRET"`
}

func LoadIntegratedApp() (IntegratedApp, error) {
	var cfg IntegratedApp
	err := envconfig.Process("", &cfg)
	if err != nil {
		return IntegratedApp{}, err
	}
	return cfg, nil
}

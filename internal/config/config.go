package config

import (
	"github.com/kelseyhightower/envconfig"
)

type IntegratedApp struct {
	App        App
	Domain     Domain
	Firestore  Firestore
	Github     Github
	SigningKey string
}

type StandaloneApp struct {
	App    App
	SQLite SQLite
}

type App struct {
	Env  string
	Port string
	Demo bool
}

type Domain struct {
	Protocol  string
	Subdomain string
	Domain    string
	BasePath  string
}

type Firestore struct {
	ProjectID    string
	CollectionID string
}

type SQLite struct {
	Location string
}

type Github struct {
	ClientID string
	Secret   string
}

func LoadIntegratedApp() (IntegratedApp, error) {
	var cfg IntegratedApp
	err := envconfig.Process("", &cfg)
	if err != nil {
		return IntegratedApp{}, err
	}
	return cfg, nil
}

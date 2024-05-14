package config

import (
	"github.com/kelseyhightower/envconfig"
)

type IntegratedApp struct {
	App
	Domain    Domain
	Firestore Firestore
	Github    Github
}

type StandaloneApp struct {
	App
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

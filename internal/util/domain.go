package util

import (
	"fmt"

	"github.com/baely/officetracker/internal/config"
)

func QualifiedDomain(cfg config.Domain) string {
	domain := cfg.Domain
	if cfg.Subdomain != "" {
		domain = fmt.Sprintf("%s.%s", cfg.Subdomain, domain)
	}

	return domain
}

func BaseUri(cfg config.IntegratedApp) string {
	domain := QualifiedDomain(cfg.Domain)
	if domain == "localhost" {
		domain = fmt.Sprintf("%s:%s", domain, cfg.Port)
	}

	return fmt.Sprintf("%s://%s%s", cfg.Domain.Protocol, domain, cfg.Domain.BasePath)
}

func BasePath(cfg config.Domain) string {
	return cfg.BasePath
}

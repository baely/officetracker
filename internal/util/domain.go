package util

import (
	"fmt"
	"os"
)

func QualifiedDomain() string {
	domain := os.Getenv("DOMAIN")

	subdomain := os.Getenv("SUBDOMAIN")
	if subdomain != "" {
		domain = fmt.Sprintf("%s.%s", subdomain, domain)
	}

	return domain
}

func BaseUri() string {
	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("APP_PORT")
	path := os.Getenv("BASE_PATH")
	domain := QualifiedDomain()
	if domain == "localhost" {
		domain = fmt.Sprintf("%s:%s", domain, port)
	}

	return fmt.Sprintf("%s://%s%s", protocol, domain, path)
}

func BasePath() string {
	return os.Getenv("BASE_PATH")
}

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
	port := os.Getenv("PORT")
	path := os.Getenv("BASE_PATH")
	return fmt.Sprintf("%s://%s:%s%s", protocol, QualifiedDomain(), port, path)
}

func BasePath() string {
	return os.Getenv("BASE_PATH")
}

package auth

import "net/http"

type Type int

const (
	NoAuth Type = iota
	SSOAuth
	SecretAuth
)

func InjectAuth(r *http.Request, authType Type) {
	switch authType {
	case NoAuth:
		injectNoAuth(r)
	case SSOAuth:
		injectSSOAuth(r)
	case SecretAuth:
		injectSecretAuth(r)
	}
}

func injectNoAuth(r *http.Request) {
}

func injectSSOAuth(r *http.Request) {
}

func injectSecretAuth(r *http.Request) {

}

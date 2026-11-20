package auth

type AuthType string

const (
	AuthNone     AuthType = "none"
	AuthInternal AuthType = "internal"
	AuthKeycloak AuthType = "keycloak"
)

type Config struct {
	AuthType         AuthType
	ClientID         string
	ClientSecret     string
	RealmURL         string
	InternalRealmURL string
	RedirectPath     string
	ClientBaseURL    string
}

func NoneConfig() *Config {
	return &Config{
		AuthType: AuthNone,
	}
}

func InternalConfig() *Config {
	return &Config{
		AuthType: AuthInternal,
	}
}

func KeycloakConfig(clientID, clientSecret, realm, internalRealm, redirectPath, clientBaseURL string) *Config {
	return &Config{
		AuthType:         AuthKeycloak,
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		RealmURL:         realm,
		InternalRealmURL: internalRealm,
		RedirectPath:     redirectPath,
		ClientBaseURL:    clientBaseURL,
	}
}

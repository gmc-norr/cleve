package auth

type AuthType string

const (
	AuthNone     AuthType = "none"
	AuthInternal AuthType = "internal"
	AuthKeycloak AuthType = "keycloak"
)

type Config struct {
	AuthType     AuthType
	ClientID     string
	ClientSecret string
	RealmURL     string
	RedirectURL  string
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

func KeycloakConfig(clientID, clientSecret, realm, redirectURL string) *Config {
	return &Config{
		AuthType:     AuthKeycloak,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RealmURL:     realm,
		RedirectURL:  redirectURL,
	}
}

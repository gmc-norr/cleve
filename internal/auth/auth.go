package auth

type AuthType string

const (
	AuthNone     AuthType = "none"
	AuthInternal AuthType = "internal"
	AuthKeyCloak AuthType = "keycloak"
)

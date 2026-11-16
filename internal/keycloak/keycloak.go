package keycloak

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/internal/auth"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const (
	oauthCookieName      = "CLEVE_OAUTH_STATE"
	LoginErrorCookieName = "CLEVE_LOGIN_ERROR"
)

type Renderer interface {
	Execute(io.Writer, any) error
	ExecuteTemplate(io.Writer, string, any) error
}

// TODO add a *slog.Logger and use this for all logging
type Client struct {
	oauth2Config oauth2.Config
	userinfoURL  string
	renderer     Renderer
}

func NewClient(client, secret, realm, internalRealm, redirectURL string, renderer Renderer) *Client {
	if internalRealm == "" {
		internalRealm = realm
	}
	authURL, _ := url.JoinPath(realm, "protocol/openid-connect/auth")
	tokenURL, _ := url.JoinPath(internalRealm, "protocol/openid-connect/token")
	userinfoURL, _ := url.JoinPath(internalRealm, "protocol/openid-connect/userinfo")
	return &Client{
		oauth2Config: oauth2.Config{
			ClientID:     client,
			ClientSecret: secret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
		},
		userinfoURL: userinfoURL,
		renderer:    renderer,
	}
}

func NewClientFromConfig(config *auth.Config, renderer Renderer) *Client {
	return NewClient(config.ClientID, config.ClientSecret, config.RealmURL, config.InternalRealmURL, config.RedirectURL, renderer)
}

func (c *Client) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState(32)
	http.SetCookie(w, &http.Cookie{
		Name:     oauthCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	authURL := c.oauth2Config.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

type userStore interface {
	GetUserByEmail(ctx context.Context, email string) (cleve.User, bool, error)
	CreateSession(ctx context.Context, userID uuid.UUID, authType auth.AuthType) (auth.Session, error)
}

func (c *Client) redirectWithError(w http.ResponseWriter, r *http.Request, redirectURL string, message string) {
	cookie := &http.Cookie{
		Name:     LoginErrorCookieName,
		Value:    url.QueryEscape(message),
		Path:     "/",
		MaxAge:   60,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (c *Client) redirectToLoginWithError(w http.ResponseWriter, r *http.Request, message string) {
	c.redirectWithError(w, r, "/login", message)
}

func (c *Client) HandleCallback(w http.ResponseWriter, r *http.Request, store userStore) {
	stateCookie, err := r.Cookie(oauthCookieName)
	if err != nil || r.URL.Query().Get("state") != stateCookie.Value {
		if err != nil {
			slog.Error("failed to get oauth cookie", "error", err)
		} else {
			slog.Error("mismatching session state", "cookie", stateCookie.Value, "request", r.URL.Query().Get("state"))
		}
		c.redirectToLoginWithError(w, r, "State mismatch between Cleve and Keycloak")
		return
	}

	code := r.URL.Query().Get("code")
	token, err := c.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("failed to exchange keycloak authentication code", "error", err)
		c.redirectToLoginWithError(w, r, "Failed to exchange keycloak authentication code")
		return
	}

	user, err := GetUserInfo(r.Context(), c.userinfoURL, token)
	if err != nil {
		slog.Error("failed to get user info", "url", c.userinfoURL, "error", err, "token", token)
		c.redirectToLoginWithError(w, r, "Authentication successful, but failed to get user info")
		return
	}

	storeUser, found, err := store.GetUserByEmail(r.Context(), user.Email)
	if err != nil {
		slog.Error("failed to get local user info", "error", err)
		c.redirectToLoginWithError(w, r, "Unexpected error when looking up user")
		return
	}
	if !found {
		c.redirectToLoginWithError(w, r, "Authentication successful, but user is not allowed to view this content")
		return
	}

	session, err := store.CreateSession(r.Context(), storeUser.ID, auth.AuthKeycloak)
	if err != nil {
		slog.Error("failed to create session", "error", err)
		c.redirectToLoginWithError(w, r, "Authentication successful, but user is not allowed to view this content")
		return
	}

	auth.SetSessionCookie(w, session)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (c *Client) RedirectURL(url string) {
	c.oauth2Config.RedirectURL = url
}

func generateState(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

type UserInfo struct {
	Sub               string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	Email             string `json:"email"`
}

func GetUserInfo(ctx context.Context, url string, token *oauth2.Token) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: %s", res.Status)
	}
	decoder := json.NewDecoder(res.Body)
	user := UserInfo{}
	if err := decoder.Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

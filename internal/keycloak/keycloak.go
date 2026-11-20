package keycloak

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	NextURLCookieName    = "CLEVE_NEXT_URL"
)

// TODO add a *slog.Logger and use this for all logging
type Client struct {
	oauth2Config          oauth2.Config
	userinfoURL           string
	externalLogoutURL     string
	internalLogoutURL     string
	postLogoutRedirectURL string
	clientBaseURL         string
}

func NewClient(client, secret, realm, internalRealm, clientBaseURL, redirectPath string) *Client {
	if internalRealm == "" {
		internalRealm = realm
	}
	authURL, _ := url.JoinPath(realm, "protocol/openid-connect/auth")
	tokenURL, _ := url.JoinPath(internalRealm, "protocol/openid-connect/token")
	userinfoURL, _ := url.JoinPath(internalRealm, "protocol/openid-connect/userinfo")
	logoutURL, _ := url.JoinPath(realm, "protocol/openid-connect/logout")
	internalLogoutURL, _ := url.JoinPath(internalRealm, "protocol/openid-connect/logout")
	return &Client{
		oauth2Config: oauth2.Config{
			ClientID:     client,
			ClientSecret: secret,
			RedirectURL:  clientBaseURL + redirectPath,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
		},
		userinfoURL:           userinfoURL,
		externalLogoutURL:     logoutURL,
		internalLogoutURL:     internalLogoutURL,
		postLogoutRedirectURL: clientBaseURL + "/logged-out",
		clientBaseURL:         clientBaseURL,
	}
}

func NewClientFromConfig(config *auth.Config) *Client {
	return NewClient(config.ClientID, config.ClientSecret, config.RealmURL, config.InternalRealmURL, config.ClientBaseURL, config.RedirectPath)
}

func (c *Client) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState(32)
	setStateCookie(w, state)

	authURL := c.oauth2Config.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

type userStore interface {
	GetUserByEmail(ctx context.Context, email string) (cleve.User, bool, error)
	CreateSession(ctx context.Context, userID uuid.UUID, userAgent string, authType auth.AuthType) (auth.Session, error)
}

func setStateCookie(w http.ResponseWriter, state string) {
	cookie := &http.Cookie{
		Name:     oauthCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

func deleteStateCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   oauthCookieName,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

func SetNextURLCookie(w http.ResponseWriter, path string) {
	cookie := &http.Cookie{
		Name:     NextURLCookieName,
		Value:    auth.SafeRedirectURL(path),
		Path:     "/",
		MaxAge:   60,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

func DeleteNextURLCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   NextURLCookieName,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

func setLoginErrorCookie(w http.ResponseWriter, message string) {
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
}

func DeleteLoginErrorCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   LoginErrorCookieName,
		MaxAge: -1,
	})
}

func (c *Client) redirectWithError(w http.ResponseWriter, r *http.Request, redirectURL string, message string) {
	setLoginErrorCookie(w, message)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (c *Client) redirectToLoginWithError(w http.ResponseWriter, r *http.Request, message string) {
	c.redirectWithError(w, r, "/login", message)
}

func (c *Client) LogoutURL(redirectURL string) (string, error) {
	return c.logoutURL(redirectURL, "", false)
}

func (c *Client) logoutURL(redirectURL, idToken string, internal bool) (string, error) {
	var (
		err       error
		revokeURL *url.URL
	)
	if internal {
		revokeURL, err = url.Parse(c.internalLogoutURL)
	} else {
		revokeURL, err = url.Parse(c.externalLogoutURL)
	}
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Set("client_id", c.oauth2Config.ClientID)
	if redirectURL != "" {
		params.Set("post_logout_redirect_uri", redirectURL)
	}
	if idToken != "" {
		params.Set("id_token_hint", idToken)
	}
	revokeURL.RawQuery = params.Encode()
	return revokeURL.String(), nil
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
	deleteStateCookie(w)

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
		setLoginErrorCookie(w, "Authentication successful, but user is not allowed to view this content")
		revokeURL, err := c.logoutURL(c.clientBaseURL+"/login", token.Extra("id_token").(string), false)
		if err != nil {
			slog.Error("failed to create logout url", "error", err)
		}
		http.Redirect(w, r, revokeURL, http.StatusSeeOther)
		return
	}

	session, err := store.CreateSession(r.Context(), storeUser.ID, r.Header.Get("User-Agent"), auth.AuthKeycloak)
	if err != nil {
		slog.Error("failed to create session", "error", err)
		c.redirectToLoginWithError(w, r, "Authentication successful, but user is not allowed to view this content")
		return
	}

	nextURL := "/"
	if nextURLCookie, err := r.Cookie(NextURLCookieName); err == nil {
		nextURL = nextURLCookie.Value
		DeleteNextURLCookie(w)
	}

	auth.SetSessionCookie(w, session)
	http.Redirect(w, r, nextURL, http.StatusFound)
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

package gin

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/internal/auth"
	"github.com/gmc-norr/cleve/internal/keycloak"
	"github.com/gmc-norr/cleve/mongo"
)

func userAuth(db *mongo.DB, config *auth.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if config == nil || config.AuthType == auth.AuthNone {
			session := auth.UserSession{
				Session: auth.Session{
					AuthType: auth.AuthNone,
				},
			}
			sessionCtx := auth.WithSession(ctx.Request.Context(), session)
			ctx.Request = ctx.Request.WithContext(sessionCtx)
			ctx.Next()
			return
		}
		session, valid, err := auth.CurrentSession(ctx.Request, db)
		if err != nil {
			ctx.HTML(http.StatusInternalServerError, "error500", nil)
			ctx.Abort()
			return
		}
		if valid {
			sessionCtx := auth.WithSession(ctx.Request.Context(), session)
			ctx.Request = ctx.Request.WithContext(sessionCtx)
			ctx.Next()
			return
		}

		q := make(url.Values, 1)
		q.Set("next_url", ctx.Request.URL.Path)
		ctx.Redirect(http.StatusFound, fmt.Sprintf("/login?%s", q.Encode()))
	}
}

func logoutHandler(db *mongo.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session, found, err := auth.CurrentSession(ctx.Request, db)
		if err != nil {
			ctx.HTML(http.StatusInternalServerError, "error500", gin.H{"message": "unexpected error when fetching user session"})
			return
		}
		if !found {
			auth.DeleteSessionCookie(ctx.Writer)
			ctx.Redirect(http.StatusSeeOther, "/login")
			return
		}
		err = auth.DeleteSession(ctx.Request.Context(), db, session)
		if err != nil {
			ctx.HTML(http.StatusInternalServerError, "error500", gin.H{"message": "unexpected error when deleting user session"})
			return
		}
		auth.DeleteSessionCookie(ctx.Writer)
		ctx.Redirect(http.StatusSeeOther, "/login")
	}
}

func loginHandler(db *mongo.DB, config *auth.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := ctx.Request.ParseForm(); err != nil {
			_ = ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid form data: %w", err))
			return
		}
		nextURL := auth.SafeRedirectURL(ctx.Request.FormValue("next_url"))
		email := ctx.Request.FormValue("email")
		password := ctx.Request.FormValue("password")

		user, found, err := db.GetUserByEmail(ctx.Request.Context(), email)
		if err != nil {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if !found || !auth.VerifyPassword(user.PasswordHash, password) {
			ctx.HTML(http.StatusUnauthorized, "login", gin.H{"email": email, "password": password, "message": UserMessage{
				Text:  "Incorrect email or password",
				Class: "warning",
			}})
			return
		}

		session, err := db.CreateSession(ctx.Request.Context(), user.ID, config.AuthType)
		if err != nil {
			slog.Error("auth", "message", "failed to create session", "error", err)
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		auth.SetSessionCookie(ctx.Writer, session)

		ctx.Redirect(http.StatusFound, nextURL)
	}
}

func keycloakLoginHandler(config *auth.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		client := keycloak.NewClientFromConfig(config)
		// TODO don't hard-code the redirect URL
		client.RedirectURL("https://vs1909.vll.se:8080/logincallback")
		client.HandleLogin(ctx.Writer, ctx.Request)
	}
}

func loginPageHandler(db *mongo.DB, config *auth.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if config.AuthType == auth.AuthNone {
			ctx.HTML(http.StatusNotFound, "error404", gin.H{
				"error": "Authentication is disabled",
			})
			return
		}

		_, valid, err := auth.CurrentSession(ctx.Request, db)
		if err != nil {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		}
		nextURL := auth.SafeRedirectURL(ctx.Query("next_url"))
		if valid {
			ctx.Redirect(http.StatusFound, nextURL)
			return
		}

		switch config.AuthType {
		case auth.AuthInternal:
			ctx.HTML(http.StatusOK, "login", gin.H{"next_url": nextURL, "cleve_version": cleve.GetVersion(), "auth_type": config.AuthType})
		case auth.AuthKeycloak:
			ctx.HTML(http.StatusOK, "login", gin.H{"cleve_version": cleve.GetVersion(), "auth_type": config.AuthType})
			return
		default:
			ctx.HTML(http.StatusInternalServerError, "error500", gin.H{
				"error": "Invalid authentication provider",
			})
		}
	}
}

func loginCallbackHandler(db *mongo.DB, config *auth.Config) gin.HandlerFunc {
	client := keycloak.NewClientFromConfig(config)
	// TODO don't hard-code the redirect URL
	client.RedirectURL("https://vs1909.vll.se:8080/logincallback")
	return func(ctx *gin.Context) {
		client.HandleCallback(ctx.Writer, ctx.Request, db)
	}
}

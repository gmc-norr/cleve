package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/google/uuid"
)

const sessionCookieName = "session_id"

type contextSessionKey struct{}

type Session struct {
	UserID              uuid.UUID `bson:"user_id"`
	ExpiresAt           time.Time `bson:"expires_at"`
	RevokedAt           time.Time `bson:"revoked_at"`
	LastAuthenticatedAt time.Time `bson:"last_authenticated_at"`
	CreatedAt           time.Time `bson:"created_at"`
	AuthType            AuthType  `bson:"auth_type"`
	Token               string    `bson:"token"`
}

type UserSession struct {
	Session Session
	User    cleve.User
}

type userStore interface {
	GetUserByID(ctx context.Context, userID uuid.UUID) (cleve.User, error)
	GetUserByEmail(ctx context.Context, email string) (cleve.User, bool, error)
	GetCurrentSession(ctx context.Context, token string) (UserSession, bool, error)
	CreateSession(ctx context.Context, userID uuid.UUID, authType AuthType) (Session, error)
	DeleteSessionByUserAndToken(ctx context.Context, userID uuid.UUID, token string) error
}

func CurrentSession(r *http.Request, store userStore) (UserSession, bool, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return UserSession{}, false, nil
	}
	return store.GetCurrentSession(r.Context(), cookie.Value)
}

func DeleteSession(ctx context.Context, store userStore, session UserSession) error {
	return store.DeleteSessionByUserAndToken(ctx, session.User.ID, session.Session.Token)
}

func SetSessionCookie(w http.ResponseWriter, session Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func DeleteSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		MaxAge: -1,
	})
}

func WithSession(ctx context.Context, session UserSession) context.Context {
	return context.WithValue(ctx, contextSessionKey{}, session)
}

func SessionFrom(ctx context.Context) (UserSession, bool) {
	session, ok := ctx.Value(contextSessionKey{}).(UserSession)
	return session, ok
}

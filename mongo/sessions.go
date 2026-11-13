package mongo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gmc-norr/cleve/internal/auth"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const sessionTTL = 7 * 24 * time.Hour

// CreateSession creates a new user session and stores it in the database.
// The session token is hashed when stored in the database, but the session
// that is returned contains the unhashed token. Returns a non-nil error if
// the session cannot be added to the database.
func (db DB) CreateSession(ctx context.Context, userID uuid.UUID, authType auth.AuthType) (auth.Session, error) {
	now := time.Now()
	expiresAt := now.Add(sessionTTL)

	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	session := auth.Session{
		UserID:              userID,
		Token:               hashToken(token),
		CreatedAt:           now,
		LastAuthenticatedAt: now,
		ExpiresAt:           expiresAt,
		AuthType:            authType,
	}

	_, err := db.SessionCollection().InsertOne(ctx, session)
	return auth.Session{
		UserID:              userID,
		Token:               token,
		CreatedAt:           now,
		LastAuthenticatedAt: now,
		ExpiresAt:           expiresAt,
		AuthType:            authType,
	}, err
}

func (db DB) GetCurrentSession(ctx context.Context, token string) (auth.UserSession, bool, error) {
	session, err := db.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return auth.UserSession{}, false, nil
		}
		return auth.UserSession{}, false, err
	}
	if time.Now().After(session.ExpiresAt) {
		return auth.UserSession{}, false, nil
	}
	if !session.RevokedAt.IsZero() {
		return auth.UserSession{}, false, nil
	}

	user, err := db.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return auth.UserSession{}, false, nil
		}
		return auth.UserSession{}, false, err
	}
	return auth.UserSession{
		User: user,
		Session: auth.Session{
			UserID:              session.UserID,
			ExpiresAt:           session.ExpiresAt,
			RevokedAt:           session.RevokedAt,
			LastAuthenticatedAt: session.LastAuthenticatedAt,
			CreatedAt:           session.CreatedAt,
			Token:               token,
			AuthType:            session.AuthType,
		},
	}, true, nil
}

func (db DB) GetSessionByToken(ctx context.Context, token string) (auth.Session, error) {
	res := db.SessionCollection().FindOne(ctx, bson.D{
		{Key: "token", Value: hashToken(token)},
	})
	var session auth.Session
	if err := res.Decode(&session); err != nil {
		return auth.Session{}, err
	}
	return session, nil
}

// DeleteSessionByToken deletes a session based on the user and token.
// If such a session cannot be found a nil error is still returned as the
// session doesn't exist and there's nothing to do. If any other error is
// returned from the mongo driver, then this is propagated.
func (db DB) DeleteSessionByUserAndToken(ctx context.Context, userID uuid.UUID, token string) error {
	_, err := db.SessionCollection().DeleteOne(ctx, bson.D{
		{Key: "token", Value: hashToken(token)},
		{Key: "user_id", Value: userID},
	})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil
		}
		return err
	}
	return nil
}

func (db DB) SessionIndex() ([]map[string]string, error) {
	cursor, err := db.SessionCollection().Indexes().List(context.TODO())
	if err != nil {
		return []map[string]string{}, err
	}
	defer closeCursor(cursor, context.TODO())

	var indexes []map[string]string

	var result []bson.M
	if err = cursor.All(context.TODO(), &result); err != nil {
		return []map[string]string{}, err
	}

	for _, v := range result {
		i := map[string]string{}
		for k, val := range v {
			i[k] = fmt.Sprintf("%v", val)
		}
		indexes = append(indexes, i)
	}

	return indexes, nil
}

func (db DB) SetSessionIndex() (string, error) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "token", Value: 1},
				{Key: "user_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}

	res, err := db.SessionCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := db.SessionCollection().Indexes().CreateMany(context.TODO(), indexModels)
	return fmt.Sprintf("%v", name), err
}

func hashToken(token string) string {
	b := sha256.Sum256([]byte(token))
	return hex.EncodeToString(b[:])
}

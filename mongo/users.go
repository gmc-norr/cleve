package mongo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db DB) CreateUser(ctx context.Context, user cleve.User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	_, err := db.UserCollection().InsertOne(ctx, user)
	return err
}

func (db DB) Users(ctx context.Context) ([]cleve.User, error) {
	cur, err := db.UserCollection().Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer closeCursor(cur, ctx)
	var users []cleve.User
	for cur.Next(ctx) {
		var user cleve.User
		if err := cur.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (db DB) GetUserByID(ctx context.Context, id uuid.UUID) (cleve.User, error) {
	res := db.UserCollection().FindOne(ctx, bson.D{
		{Key: "id", Value: id},
	})
	user := cleve.User{}
	if err := res.Decode(&user); err != nil {
		return cleve.User{}, err
	}
	return user, nil
}

func (db DB) GetUserByEmail(ctx context.Context, email string) (cleve.User, bool, error) {
	res := db.UserCollection().FindOne(ctx, bson.D{
		{Key: "email", Value: email},
	})
	user := cleve.User{}
	if err := res.Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return cleve.User{}, false, nil
		}
		return cleve.User{}, false, err
	}
	return user, true, nil
}

// DeleteUser deletes a user from the database based on its ID. Returns
// true if the user was successfully deleted and false if such a user could
// not be found. Any underlying database errors are returned as well.
func (db DB) DeleteUser(ctx context.Context, id uuid.UUID) (bool, error) {
	res, err := db.UserCollection().DeleteOne(ctx, bson.D{
		{Key: "id", Value: id},
	})
	return res != nil && res.DeletedCount == 1, err
}

func (db DB) UserIndex() ([]map[string]string, error) {
	cursor, err := db.UserCollection().Indexes().List(context.TODO())
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

func (db DB) SetUserIndex() (string, error) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "email", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}

	res, err := db.UserCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := db.UserCollection().Indexes().CreateMany(context.TODO(), indexModels)
	return fmt.Sprintf("%v", name), err
}

package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Panels returns all gene panels in the collection, but without the
// genes that they contain. Only the most recent panel for each ID
// is returned, based on creation date. Control what panels are returned
// with the filter that is passed in.
func (db DB) Panels(filter cleve.PanelFilter) ([]cleve.GenePanel, error) {
	var (
		pipeline mongo.Pipeline
		panels   []cleve.GenePanel
	)

	matchFields := bson.D{}

	if !filter.Archived {
		matchFields = append(matchFields, bson.E{
			Key: "archived", Value: false,
		})
	}
	if filter.Category != "" {
		matchFields = append(matchFields, bson.E{
			Key: "categories", Value: bson.D{
				{Key: "$elemMatch", Value: bson.D{{Key: "$eq", Value: filter.Category}}},
			},
		})
	}
	if filter.GeneQuery != "" {
		matchFields = append(matchFields, bson.E{
			Key: "genes", Value: bson.D{
				{Key: "$elemMatch", Value: bson.D{
					{Key: "symbol", Value: bson.D{
						{Key: "$regex", Value: filter.GeneQuery},
						{Key: "$options", Value: "i"},
					}},
				}},
			},
		})
	}
	if filter.Gene != "" {
		matchFields = append(matchFields, bson.E{
			Key: "genes", Value: bson.D{
				{Key: "$elemMatch", Value: bson.D{
					{Key: "symbol", Value: bson.D{
						{Key: "$regex", Value: "^" + filter.Gene + "$"},
						{Key: "$options", Value: "i"},
					}},
				}},
			},
		})
	}
	if filter.NameQuery != "" {
		matchFields = append(matchFields, bson.E{
			Key: "name", Value: bson.D{
				{Key: "$regex", Value: filter.NameQuery},
				{Key: "$options", Value: "i"},
			},
		})
	}

	pipeline = mongo.Pipeline([]bson.D{
		{{Key: "$match", Value: matchFields}},
		{{Key: "$project", Value: bson.D{
			{Key: "genes", Value: 0},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$id"},
			{Key: "panel", Value: bson.D{
				{Key: "$top", Value: bson.D{
					{Key: "output", Value: "$$ROOT"},
					{Key: "sortBy", Value: bson.D{
						{Key: "date", Value: -1},
					}},
				}},
			}},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$panel"},
		}}},

		{{Key: "$replaceRoot", Value: bson.D{
			{Key: "newRoot", Value: "$panel"},
		}}},

		{{Key: "$sort", Value: bson.D{
			{Key: "name", Value: 1},
		}}},
	})

	cursor, err := db.PanelCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return panels, err
	}

	for cursor.Next(context.TODO()) {
		p := cleve.GenePanel{}
		if err := cursor.Decode(&p); err != nil {
			return panels, err
		}
		panels = append(panels, p)
	}

	return panels, nil
}

// Panel returns a specific gene panel given an ID and a version. If the panel
// does not exist, an error is returned.
func (db DB) Panel(id string, version string) (cleve.GenePanel, error) {
	var p cleve.GenePanel
	err := db.PanelCollection().FindOne(context.TODO(), bson.D{{Key: "id", Value: id}, {Key: "version", Value: version}}).Decode(&p)
	return p, err
}

// PanelVersions returns a slice of panel versions that exist for a given panel ID.
// The versions are sorted in reverse chronological order based on creation date.
func (db DB) PanelVersions(id string) ([]cleve.GenePanelVersion, error) {
	var versions []cleve.GenePanelVersion

	pipeline := mongo.Pipeline([]bson.D{
		{{Key: "$match", Value: bson.D{
			{Key: "id", Value: id},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "version", Value: 1},
			{Key: "date", Value: 1},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "date", Value: -1},
		}}},
	})

	cursor, err := db.PanelCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return versions, err
	}

	for cursor.Next(context.TODO()) {
		var v cleve.GenePanelVersion
		if err := cursor.Decode(&v); err != nil {
			return versions, err
		}
		versions = append(versions, v)
	}

	if len(versions) == 0 {
		return versions, ErrNoDocuments
	}

	return versions, nil
}

// PanelCategories returns a slice of strings representing all panel categories in the
// database. Only unique entries are returned, and they are sorted lexicographically.
func (db DB) PanelCategories() ([]string, error) {
	categories := make([]string, 0)
	pipeline := mongo.Pipeline([]bson.D{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$id"},
			{Key: "panel", Value: bson.D{
				{Key: "$top", Value: bson.D{
					{Key: "output", Value: "$$ROOT"},
					{Key: "sortBy", Value: bson.D{{Key: "date", Value: -1}}},
				}},
			}},
		}}},
		{{Key: "$replaceRoot", Value: bson.D{{Key: "newRoot", Value: "$panel"}}}},
		{{Key: "$project", Value: bson.D{{Key: "category", Value: "$categories"}}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$category"},
			{Key: "preserveNullAndEmptyArrays", Value: false},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "categories", Value: bson.D{
				{Key: "$addToSet", Value: "$category"},
			}},
		}}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "categories", Value: bson.D{
				{Key: "$sortArray", Value: bson.D{
					{Key: "input", Value: "$categories"},
					{Key: "sortBy", Value: 1},
				}},
			}},
		}}},
	})
	cursor, err := db.PanelCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return categories, err
	}
	defer cursor.Close(context.TODO())
	cursor.Next(context.TODO())

	type aux struct {
		Categories []string
	}
	var c aux
	if err := cursor.Decode(&c); err != nil {
		return categories, err
	}

	return c.Categories, nil
}

// CreatePanel takes adds a new gene panel to the database. If a panel with the
// same ID and version already exists, an error is returned.
func (db DB) CreatePanel(p cleve.GenePanel) error {
	auxPanel := struct {
		ImportedAt      time.Time
		cleve.GenePanel `bson:",inline"`
	}{
		ImportedAt: time.Now().UTC(),
		GenePanel:  p,
	}
	_, err := db.PanelCollection().InsertOne(context.TODO(), auxPanel)
	return err
}

// ArchivePanel archives a panel given an ID. All existing versions of the panel are
// archived.
func (db DB) ArchivePanel(id string) error {
	res, err := db.PanelCollection().UpdateMany(context.TODO(), bson.D{{Key: "id", Value: id}}, bson.D{{Key: "$set", Value: bson.D{{Key: "archived", Value: true}}}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNoDocuments
	}
	return nil
}

// UnarchivePanel unarchives a panel given an ID. All existing versions of the panel are
// unarchived.
func (db DB) UnarchivePanel(id string) error {
	res, err := db.PanelCollection().UpdateMany(context.TODO(), bson.D{{Key: "id", Value: id}}, bson.D{{Key: "$set", Value: bson.D{{Key: "archived", Value: false}}}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNoDocuments
	}
	return nil
}

// PanelIndex returns the the current indexes for the panel collection.
func (db DB) PanelIndex() ([]map[string]string, error) {
	cursor, err := db.RunCollection().Indexes().List(context.TODO())
	if err != nil {
		return []map[string]string{}, err
	}
	defer cursor.Close(context.TODO())

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

// SetPanelIndex sets all indexes for the panel collection. Existing indexes
// are removed before new indexes are created.
func (db *DB) SetPanelIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "id", Value: 1},
			{Key: "version", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	_, err := db.PanelCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	// log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := db.PanelCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

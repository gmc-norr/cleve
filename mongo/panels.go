package mongo

import (
	"context"
	"errors"
	"fmt"
	"io"
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

	panels = make([]cleve.GenePanel, 0)
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
		var p struct {
			Version         string
			cleve.GenePanel `bson:",inline"`
		}
		if err := cursor.Decode(&p); err != nil {
			return panels, err
		}
		p.GenePanel.Version, err = cleve.ParseVersion(p.Version)
		if err != nil {
			return panels, err
		}
		panels = append(panels, p.GenePanel)
	}

	return panels, nil
}

// Panel returns a specific gene panel given an ID and a version. If the ID
// is the empty string, the most recent version of the panel is returned.
// If the panel does not exist, an error is returned.
func (db DB) Panel(id string, version string) (cleve.GenePanel, error) {
	var p struct {
		Version         string
		cleve.GenePanel `bson:",inline"`
	}
	matchFields := bson.D{
		{Key: "id", Value: id},
	}
	if version != "" {
		matchFields = append(matchFields, bson.E{Key: "version", Value: version})
	}
	pipeline := mongo.Pipeline([]bson.D{
		{{Key: "$match", Value: matchFields}},
	})
	if version == "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$sort", Value: bson.D{
				{Key: "date", Value: -1},
			}},
		}, bson.D{{Key: "$limit", Value: 1}})
	}
	cursor, err := db.PanelCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return p.GenePanel, err
	}
	defer cursor.Close(context.TODO())
	if ok := cursor.Next(context.TODO()); !ok {
		return p.GenePanel, ErrNoDocuments
	}
	if cursor.Next(context.TODO()) {
		return p.GenePanel, fmt.Errorf("expected one document, found more than that")
	}
	if err := cursor.Decode(&p); err != nil {
		return p.GenePanel, err
	}
	p.GenePanel.Version, err = cleve.ParseVersion(p.Version)
	return p.GenePanel, err
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
		var v struct {
			Version                string
			cleve.GenePanelVersion `bson:",inline"`
		}
		if err := cursor.Decode(&v); err != nil {
			return versions, err
		}
		v.GenePanelVersion.Version, err = cleve.ParseVersion(v.Version)
		if err != nil {
			return versions, err
		}
		versions = append(versions, v.GenePanelVersion)
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
	err = cursor.Decode(&c)

	if errors.Is(err, io.EOF) {
		return c.Categories, nil
	}
	return c.Categories, err
}

// CreatePanel takes adds a new gene panel to the database. If a panel with the
// same ID and version already exists, an error is returned.
func (db DB) CreatePanel(p cleve.GenePanel) error {
	existingPanel, err := db.Panel(p.Id, "")
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	if existingPanel.Archived {
		return fmt.Errorf("%w: panel is archived", ConflictError)
	}
	if existingPanel.Date.After(p.Date) {
		return fmt.Errorf("%w: a newer version of this panel already exists", ConflictError)
	}
	if time.Now().Before(p.Date) {
		return fmt.Errorf("%w: panel cannot have a creation date in the future", ConflictError)
	}
	auxPanel := struct {
		ImportedAt      time.Time
		Version         string
		cleve.GenePanel `bson:",inline"`
	}{
		ImportedAt: time.Now().UTC(),
		Version:    p.Version.String(),
		GenePanel:  p,
	}
	_, err = db.PanelCollection().InsertOne(context.TODO(), auxPanel)
	return err
}

// ArchivePanel archives a panel given an ID. All existing versions of the panel are
// archived.
func (db DB) ArchivePanel(id string) error {
	res, err := db.PanelCollection().UpdateMany(
		context.TODO(),
		bson.D{{Key: "id", Value: id}},
		[]bson.D{
			{{Key: "$addFields", Value: bson.D{
				{Key: "archived", Value: true},
				{Key: "archivedat", Value: time.Now()},
			}}},
		},
	)
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
	res, err := db.PanelCollection().UpdateMany(
		context.TODO(),
		bson.D{{Key: "id", Value: id}},
		[]bson.D{
			{{Key: "$addFields", Value: bson.D{
				{Key: "archived", Value: false},
			}}},
			{{Key: "$project", Value: bson.D{
				{Key: "archivedat", Value: 0},
			}}},
		})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNoDocuments
	}
	return nil
}

// DeletePanel deletes either a single version of a panel if both `name` and `version` are,
// given. If `version` is the empty string, then all versions of a panel ID are deleted.
// Returns the number of documents deleted and an error.
func (db DB) DeletePanel(id string, version string) (int, error) {
	var err error
	var res *mongo.DeleteResult
	deleteAll := version == ""
	if deleteAll {
		res, err = db.PanelCollection().DeleteMany(
			context.TODO(),
			bson.D{{Key: "id", Value: id}},
		)
	} else {
		res, err = db.PanelCollection().DeleteOne(
			context.TODO(),
			bson.D{
				{Key: "id", Value: id},
				{Key: "version", Value: version},
			},
		)
	}
	return int(res.DeletedCount), err
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

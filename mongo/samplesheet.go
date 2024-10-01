package mongo

import (
	"context"
	"fmt"
	"log"

	"github.com/gmc-norr/cleve"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type sampleSheetOptions struct {
	runId *string
	uuid  *uuid.UUID
}

type SampleSheetOption func(*sampleSheetOptions) error

// SampleSheetWithRunId associates the sample sheet with a run ID.
func SampleSheetWithRunId(runId string) SampleSheetOption {
	return func(o *sampleSheetOptions) error {
		if runId == "" {
			return fmt.Errorf("run id must not be empty")
		}
		if len(runId) > 128 {
			return fmt.Errorf("run id cannot be longer than 128 characters")
		}
		o.runId = &runId
		return nil
	}
}

// SampleSheetWithUuid associates the sample sheet with a UUID.
func SampleSheetWithUuid(id string) SampleSheetOption {
	return func(o *sampleSheetOptions) error {
		ssUuid, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		o.uuid = &ssUuid
		return nil
	}
}

// Add a sample sheet to the database. If the same sample sheet already
// exists, it will be updated, but only if the modification time is newer than
// the existing sample sheet. The UUID is the main identifier for the sample
// sheet, but if that is missing, the run ID from the options is then used.
// If neither a UUID nor a run ID can be found, an error is returned.
func (db DB) CreateSampleSheet(sampleSheet cleve.SampleSheet, opts ...SampleSheetOption) (*cleve.UpdateResult, error) {
	var ssOptions sampleSheetOptions
	for _, opt := range opts {
		if err := opt(&ssOptions); err != nil {
			return nil, err
		}
	}

	if sampleSheet.UUID == nil && ssOptions.runId == nil {
		return nil, fmt.Errorf("run id not supplied, and samplesheet has no uuid")
	}

	var updateKey bson.D
	if sampleSheet.UUID != nil {
		updateKey = bson.D{{Key: "uuid", Value: sampleSheet.UUID}}
	} else {
		updateKey = bson.D{{Key: "run_id", Value: ssOptions.runId}}
	}

	if ssOptions.runId != nil {
		sampleSheet.RunID = ssOptions.runId
	}

	updatedSampleSheet := &sampleSheet
	var existingSampleSheet cleve.SampleSheet
	var err error
	// Either UUID or RunID are non-nil, prioritise UUID for merging
	if sampleSheet.UUID != nil {
		existingSampleSheet, err = db.SampleSheet(SampleSheetWithUuid(sampleSheet.UUID.String()))
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}
	} else {
		existingSampleSheet, err = db.SampleSheet(SampleSheetWithRunId(*sampleSheet.RunID))
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}
	}

	if err == nil {
		updatedSampleSheet, err = existingSampleSheet.Merge(&sampleSheet)
		if err != nil {
			return nil, err
		}
	}

	return db.SampleSheetCollection().ReplaceOne(context.TODO(), updateKey, updatedSampleSheet, options.Replace().SetUpsert(true))
}

func (db DB) DeleteSampleSheet(runID string) error {
	res, err := db.SampleSheetCollection().DeleteOne(context.TODO(), bson.D{{Key: "run_id", Value: runID}})
	if err == nil && res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (db DB) SampleSheets() ([]cleve.SampleSheet, error) {
	var sampleSheets []cleve.SampleSheet
	cursor, err := db.SampleSheetCollection().Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &sampleSheets); err != nil {
		return nil, err
	}
	return sampleSheets, nil
}

// Get a samplesheet either by run ID or UUID, passed by options.
func (db DB) SampleSheet(opts ...SampleSheetOption) (cleve.SampleSheet, error) {
	var sampleSheet cleve.SampleSheet

	if len(opts) == 0 {
		return sampleSheet, fmt.Errorf("at least one option must be supplied")
	}

	var ssOptions sampleSheetOptions
	for _, opt := range opts {
		if err := opt(&ssOptions); err != nil {
			return sampleSheet, err
		}
	}

	var key bson.D
	if ssOptions.uuid != nil {
		key = bson.D{{Key: "uuid", Value: ssOptions.uuid}}
	} else if ssOptions.runId != nil {
		key = bson.D{{Key: "run_id", Value: ssOptions.runId}}
	}

	err := db.SampleSheetCollection().FindOne(context.TODO(), key).Decode(&sampleSheet)
	return sampleSheet, err
}

func (db DB) SampleSheetIndex() ([]map[string]string, error) {
	cursor, err := db.SampleSheetCollection().Indexes().List(context.TODO())
	defer cursor.Close(context.TODO())

	var indexes []map[string]string
	if err != nil {
		return []map[string]string{}, err
	}

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

func (db DB) SetSampleSheetIndex() (string, error) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "run_id", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(
				bson.D{
					{
						Key: "run_id",
						Value: bson.D{{
							Key:   "$type",
							Value: "string",
						}},
					},
				},
			),
		},
		{
			Keys: bson.D{
				{Key: "uuid", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(
				bson.D{
					{
						Key: "uuid",
						Value: bson.D{{
							Key:   "$type",
							Value: "binData",
						}},
					},
				},
			),
		},
	}

	// TODO: do this as a transaction and roll back if anything fails
	res, err := db.SampleSheetCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := db.SampleSheetCollection().Indexes().CreateMany(context.TODO(), indexModels)
	return fmt.Sprintf("%v", name), err
}

package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db DB) CreateRunQC(runId string, qc interop.InteropSummary) error {
	type auxQc struct {
		Version int                    `bson:"schema_version"`
		Qc      interop.InteropSummary `bson:",inline"`
	}
	aqc := auxQc{
		Version: 2,
		Qc:      qc,
	}
	_, err := db.RunQCCollection().InsertOne(context.TODO(), aqc)
	return err
}

func (db DB) DeleteRunQC(runId string) error {
	_, err := db.RunQCCollection().DeleteOne(context.TODO(), bson.D{
		{Key: "run_id", Value: runId},
	})
	return err
}

func (db DB) UpdateRunQC(qc interop.InteropSummary) error {
	type auxQc struct {
		Version int                    `bson:"schema_version"`
		Qc      interop.InteropSummary `bson:",inline"`
	}
	aqc := auxQc{
		Version: 2,
		Qc:      qc,
	}
	_, err := db.RunQCCollection().ReplaceOne(context.TODO(), bson.D{
		{Key: "run_id", Value: qc.RunId},
	}, aqc, options.Replace().SetUpsert(true))
	return err
}

func (db DB) RunQCs(filter cleve.QcFilter) (cleve.QcResult, error) {
	var pipeline mongo.Pipeline
	var qc cleve.QcResult

	pipeline = append(pipeline, bson.D{
		{
			Key: "$set", Value: bson.D{
				{
					Key: "schema_version",
					Value: bson.D{
						{Key: "$ifNull", Value: bson.A{"$schema_version", 1}},
					},
				},
			},
		},
	})

	// Run ID filter
	if filter.RunId != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "run_id", Value: filter.RunId},
			}},
		})
	}

	// Run ID query
	if filter.RunIdQuery != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$expr", Value: bson.D{
					{Key: "$regexMatch", Value: bson.M{
						"input":   "$run_id",
						"regex":   filter.RunIdQuery,
						"options": "i",
					}},
				}},
			}},
		})
	}

	// Platform filter
	if filter.Platform != "" {
		platform, err := db.Platform(filter.Platform)
		if err != nil {
			return qc, err
		}
		platformNames := append(platform.Aliases, platform.Name)
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$expr", Value: bson.D{{Key: "$in", Value: bson.A{"$platform", platformNames}}}},
			}},
		})
	}

	qc.PaginationMetadata = cleve.PaginationMetadata{
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	metaPipeline := append(
		pipeline,
		bson.D{{Key: "$count", Value: "total_count"}},
	)
	cursor, err := db.RunQCCollection().Aggregate(context.TODO(), metaPipeline)
	if err != nil {
		return qc, err
	}
	defer cursor.Close(context.TODO())

	if !cursor.Next(context.TODO()) {
		// No documents
		qc.PaginationMetadata.TotalCount = 0
	}
	if err := cursor.Decode(&qc.PaginationMetadata); qc.PaginationMetadata.TotalCount > 0 && err != nil {
		return qc, err
	}
	if qc.PageSize > 0 {
		qc.TotalPages = qc.TotalCount/qc.PageSize + 1
	} else {
		qc.TotalPages = 1
	}

	// Sort by date, descending
	pipeline = append(pipeline, bson.D{
		{Key: "$sort", Value: bson.D{
			{Key: "date", Value: -1},
			{Key: "run_id", Value: -1},
		}},
	})

	// Pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		pipeline = append(
			pipeline,
			bson.D{{Key: "$skip", Value: (filter.Page - 1) * filter.PageSize}},
		)
	}

	if filter.PageSize > 0 {
		pipeline = append(
			pipeline,
			bson.D{{Key: "$limit", Value: filter.PageSize}},
		)
	}

	cursor, err = db.RunQCCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return qc, err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var auxQc struct {
			Version int                    `bson:"schema_version"`
			Qc      interop.InteropSummary `bson:",inline"`
		}

		if err := cursor.Decode(&auxQc); err != nil {
			return qc, err
		}

		if auxQc.Version < 2 {
			auxQc.Qc.Date = time.Time{}
		}

		qc.PaginationMetadata.Count++
		qc.InteropSummary = append(qc.InteropSummary, auxQc.Qc)
	}

	if qc.TotalCount == 0 {
		qc.TotalPages = 1
		qc.InteropSummary = make([]interop.InteropSummary, 0)
	}
	if qc.Page > qc.TotalPages {
		return qc, PageOutOfBoundsError{
			page:       qc.Page,
			totalPages: qc.TotalPages,
		}
	}

	return qc, nil
}

func (db DB) RunQC(runId string) (interop.InteropSummary, error) {
	var is interop.InteropSummary
	filter := cleve.QcFilter{
		RunId: runId,
	}
	qc, err := db.RunQCs(filter)
	if err != nil {
		return is, err
	}
	if qc.Count == 0 {
		return is, mongo.ErrNoDocuments
	}
	if qc.Count != 1 {
		return is, fmt.Errorf("found more than one matching document")
	}
	is = qc.InteropSummary[0]
	return is, nil
}

func (db DB) RunQCIndex() ([]map[string]string, error) {
	cursor, err := db.RunQCCollection().Indexes().List(context.TODO())
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

func (db DB) SetRunQCIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	_, err := db.RunQCCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	name, err := db.RunQCCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

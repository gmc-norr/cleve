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

	qcFacet := mongo.Pipeline{}

	// Sort by date, descending
	qcFacet = append(qcFacet, bson.D{
		{Key: "$sort", Value: bson.D{
			{Key: "date", Value: -1},
		}},
	})

	// Skip
	if filter.Page > 0 && filter.PageSize > 0 {
		qcFacet = append(qcFacet, bson.D{
			{Key: "$skip", Value: filter.PageSize * (filter.Page - 1)},
		})
	}

	// Limit
	if filter.PageSize > 0 {
		qcFacet = append(qcFacet, bson.D{
			{Key: "$limit", Value: filter.PageSize},
		})
	}

	// Facet to get both metadata and qc
	pipeline = append(pipeline, bson.D{
		{Key: "$facet", Value: bson.M{
			"metadata": bson.A{
				bson.D{{Key: "$count", Value: "total_count"}},
			},
			"interop": qcFacet,
		}},
	})

	pipeline = append(pipeline, bson.D{
		{
			Key: "$project",
			Value: bson.M{
				"metadata": bson.M{
					"$arrayElemAt": bson.A{"$metadata", 0},
				},
				"interop": 1,
			},
		},
	})

	pipeline = append(pipeline, bson.D{
		{
			Key: "$set",
			Value: bson.M{
				"metadata.count": bson.M{
					"$size": "$interop",
				},
				"metadata.page":      filter.Page,
				"metadata.page_size": filter.PageSize,
				"metadata.total_pages": bson.M{
					"$cond": bson.M{
						"if": bson.M{
							"$gt": bson.A{
								filter.PageSize,
								0,
							},
						},
						"then": bson.M{
							"$ceil": bson.M{
								"$divide": bson.A{
									"$metadata.total_count",
									filter.PageSize,
								},
							},
						},
						"else": 1,
					},
				},
			},
		},
	})

	cursor, err := db.RunQCCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return qc, err
	}
	defer cursor.Close(context.TODO())

	cursor.Next(context.TODO())

	var raw bson.Raw
	if err := cursor.Decode(&raw); err != nil {
		return qc, err
	}

	var pagination struct {
		cleve.PaginationMetadata `bson:"metadata"`
	}
	if err := bson.Unmarshal(raw, &pagination); err != nil {
		return qc, err
	}
	qc.PaginationMetadata = pagination.PaginationMetadata

	rawQc := raw.Lookup("interop")
	if rawQc.Type != bson.TypeArray {
		return qc, fmt.Errorf("expected interop to be an array")
	}

	rawQcArray := rawQc.Array()
	rawQcElems, _ := rawQcArray.Elements()
	qc.InteropSummary = make([]interop.InteropSummary, qc.Count)

	for i, elem := range rawQcElems {
		var schemaVersion struct {
			Version int `bson:"schema_version"`
		}

		if err := bson.Unmarshal(elem.Value().Document(), &schemaVersion); err != nil {
			return qc, err
		}

		var interop interop.InteropSummary
		if err := bson.Unmarshal(elem.Value().Document(), &interop); err != nil {
			return qc, err
		}

		if schemaVersion.Version < 2 {
			interop.Date = time.Time{}
		}

		qc.InteropSummary[i] = interop
	}

	if qc.TotalCount == 0 {
		qc.TotalPages = 1
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

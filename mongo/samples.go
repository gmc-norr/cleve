package mongo

import (
	"context"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Retrieves a single sample from the database.
func (db DB) Sample(sampleId string) (*cleve.Sample, error) {
	var sample *cleve.Sample
	if err := db.SampleCollection().FindOne(context.TODO(), bson.M{"id": sampleId}).Decode(sample); err != nil {
		return sample, err
	}
	return sample, nil
}

// Retrieves samples from the database.
func (db DB) Samples(filter *cleve.SampleFilter) (*cleve.SampleResult, error) {
	var sampleResult cleve.SampleResult

	var pipeline mongo.Pipeline

	// Sample name filtering
	if filter.Name != "" {
		pipeline = append(pipeline, bson.D{
			{
				Key: "$match",
				Value: bson.M{
					"$expr": bson.M{
						"$regexMatch": bson.M{
							"input":   "$name",
							"regex":   filter.Name,
							"options": "i",
						},
					},
				},
			},
		})
	}

	// Facetting pipeline
	facetPipeline := mongo.Pipeline{}

	if filter.Page > 0 {
		facetPipeline = append(facetPipeline, bson.D{{
			Key:   "$skip",
			Value: filter.PageSize * (filter.Page - 1),
		}})
	}

	if filter.PageSize > 0 {
		facetPipeline = append(facetPipeline, bson.D{{
			Key:   "$limit",
			Value: filter.PageSize,
		}})
	}

	// Facetting
	pipeline = append(pipeline, bson.D{
		{
			Key: "$facet",
			Value: bson.M{
				"metadata": bson.A{
					bson.M{
						"$count": "total_count",
					},
				},
				"samples": facetPipeline,
			},
		},
	})

	// Projection
	pipeline = append(pipeline, bson.D{
		{
			Key: "$project",
			Value: bson.M{
				"samples": 1,
				"metadata": bson.M{
					"$arrayElemAt": bson.A{"$metadata", 0},
				},
			},
		},
	})

	// Add more pagination metadata
	pipeline = append(pipeline, bson.D{
		{
			Key: "$set",
			Value: bson.M{
				"metadata.count": bson.M{
					"$size": "$samples",
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

	cursor, err := db.SampleCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return &sampleResult, err
	}
	defer closeCursor(cursor, context.TODO())
	if ok := cursor.Next(context.TODO()); ok {
		err := cursor.Decode(&sampleResult)
		if err != nil {
			return &sampleResult, err
		}
		if sampleResult.TotalCount == 0 {
			// No results found. Represent this as a single page
			// with an empty slice of samples.
			sampleResult.TotalPages = 1
		}
		if sampleResult.Page > sampleResult.TotalPages {
			return &sampleResult, PageOutOfBoundsError{
				page:       sampleResult.Page,
				totalPages: sampleResult.TotalPages,
			}
		}
	}
	return &sampleResult, cursor.Err()
}

// CreateSample stores a sample in the database.
func (db DB) CreateSample(sample *cleve.Sample) error {
	_, err := db.SampleCollection().InsertOne(context.TODO(), sample)
	return err
}

// CreateSamples populates the database with a slice of samples.
func (db DB) CreateSamples(samples []*cleve.Sample) error {
	_, err := db.SampleCollection().InsertMany(context.TODO(), []any{samples})
	return err
}

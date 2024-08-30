package mongo

import (
	"context"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
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
func (db DB) Samples() ([]*cleve.Sample, error) {
	var samples []*cleve.Sample
	cursor, err := db.SampleCollection().Find(context.TODO(), bson.D{})
	if err != nil {
		return samples, err
	}
	err = cursor.All(context.TODO(), &samples)
	if err == nil && samples == nil {
		samples = make([]*cleve.Sample, 0)
	}
	return samples, err
}

// Stores a sample in the database.
func (db DB) CreateSample(sample *cleve.Sample) error {
	_, err := db.SampleCollection().InsertOne(context.TODO(), sample)
	return err
}

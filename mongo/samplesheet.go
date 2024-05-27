package mongo

import (
	"context"
	"fmt"
	"log"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SampleSheetService struct {
	coll *mongo.Collection
}

// Add a samplesheet to the database. If a samplesheet for the run already
// exists, it will be updated, but only if the modification time is newer than
// the existing samplesheet.
func (s *SampleSheetService) Create(runID string, sampleSheet cleve.SampleSheet) (*mongo.UpdateResult, error) {
	sampleSheet.RunID = runID

	updateCond := bson.E{Key: "$gt", Value: bson.A{
		sampleSheet.ModificationTime,
		"$modification_time",
	}}

	return s.coll.UpdateOne(context.TODO(),
		bson.D{{Key: "run_id", Value: runID}},
		bson.A{
			bson.D{{Key: "$set", Value: bson.D{
				{Key: "run_id", Value: bson.D{
					{Key: "$cond", Value: bson.A{
						updateCond,
						sampleSheet.RunID,
						"$run_id",
					}},
				}}}}},
			bson.D{{Key: "$set", Value: bson.D{
				{Key: "modification_time", Value: bson.D{
					{Key: "$cond", Value: bson.A{
						updateCond,
						sampleSheet.ModificationTime,
						"$modification_time",
					}},
				}}}}},
			bson.D{{Key: "$set", Value: bson.D{
				{Key: "sections", Value: bson.D{
					{Key: "$cond", Value: bson.A{
						updateCond,
						sampleSheet.Sections,
						"$sections",
					}},
				}}}}},
		},
		options.Update().SetUpsert(true),
	)
}

func (s *SampleSheetService) All() ([]cleve.SampleSheet, error) {
	var sampleSheets []cleve.SampleSheet
	cursor, err := s.coll.Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &sampleSheets); err != nil {
		return nil, err
	}
	return sampleSheets, nil
}

func (s *SampleSheetService) Get(runID string) (cleve.SampleSheet, error) {
	var sampleSheet cleve.SampleSheet
	err := s.coll.FindOne(context.TODO(), bson.D{{Key: "run_id", Value: runID}}).Decode(&sampleSheet)
	return sampleSheet, err
}

func (s *SampleSheetService) GetIndex() ([]map[string]string, error) {
	cursor, err := s.coll.Indexes().List(context.TODO())
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

func (s *SampleSheetService) SetIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	res, err := s.coll.Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := s.coll.Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

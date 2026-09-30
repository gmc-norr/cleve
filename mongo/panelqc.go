package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)


func (db DB) PanelQCs(filter cleve.PanelQcFilter) (cleve.PanelQcResult, error) {
	var (
		pipeline             mongo.Pipeline
		panelqc_result       cleve.PanelQcResult
	)

	panelqc_result.PanelQCs = make([]*cleve.PanelQc, 0)

	panelqc_result.PaginationMetadata = cleve.PaginationMetadata{
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	
	if filter.RunId != "" {
		pipeline = append(pipeline, bson.D{
			{
				Key: "$match",
				Value: bson.M{
					"run_id": filter.RunId,
				},
			},
		})
	}

	if filter.GenePanelId != "" {
		pipeline = append(pipeline, bson.D{
			{
				Key: "$match",
				Value: bson.D{ 
					{Key: "genepanel.genepanel_id", Value: filter.GenePanelId},
				},
			},
		})
	}

	metaPipeline := append(pipeline,bson.D{
		{Key: "$count", Value: "total_count"},
	})

	cursor, err := db.PanelQcCollection().Aggregate(context.TODO(), metaPipeline)
	if err != nil {
		return panelqc_result, err
	} 
	if !cursor.Next(context.TODO()) {
		panelqc_result.TotalCount = 0
	}
	if err := cursor.Decode(&panelqc_result.PaginationMetadata); panelqc_result.TotalCount > 0 && err != nil {
		return panelqc_result, err
	}
	if panelqc_result.PageSize > 0 {
		panelqc_result.TotalPages = panelqc_result.TotalCount / panelqc_result.PageSize
		if panelqc_result.TotalCount%panelqc_result.PageSize > 0 {
			panelqc_result.TotalPages += 1
		}
	} else {
		panelqc_result.TotalPages = 1
	}

	if filter.Page > 0 {
		pipeline = append(pipeline, bson.D{
			{Key: "$skip", Value: filter.PageSize * (filter.Page - 1)},
		})
	}

	if filter.PageSize > 0 {
		pipeline = append(pipeline, bson.D{
			{Key: "$limit", Value: filter.PageSize},
		})
	}
	
	cursor, err = db.PanelQcCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return panelqc_result, err
	}

	defer closeCursor(cursor, context.TODO())

	
	for cursor.Next(context.TODO()) {
		var run cleve.PanelQc
		err = cursor.Decode(&run)
		if err != nil {
			return panelqc_result, err
		}
		panelqc_result.Count++
		panelqc_result.PanelQCs = append(panelqc_result.PanelQCs, &run)
	}
	

	if panelqc_result.TotalCount == 0 {

		panelqc_result.TotalPages = 1
		panelqc_result.PanelQCs = make([]*cleve.PanelQc, 0)
	}
	
	if panelqc_result.Page > panelqc_result.TotalPages {
		return panelqc_result, PageOutOfBoundsError{
			page:       panelqc_result.Page,
			totalPages: panelqc_result.TotalPages,
		}
	}

	return panelqc_result, nil

}

func (db DB) PanelQc(runId string) (*cleve.PanelQc, error) {
	
	filter := cleve.PanelQcFilter {
		RunId: runId,
	}

	panelqc, err := db.PanelQCs(filter)
	if err != nil {
		return nil, err
	}

	if panelqc.Count == 0 {
		return nil, mongo.ErrNoDocuments
	}
	if panelqc.Count > 1 {
		return nil, fmt.Errorf("found more than one matching run")
	}
	
	return panelqc.PanelQCs[0], nil

}

func (db DB) GenesQCs(filter cleve.PanelQcFilter) (cleve.GeneQcResult, error) {
	var (
		pipeline             mongo.Pipeline
		panelqc_result       cleve.GeneQcResult
	)

	panelqc_result.GeneQcs  = make([]*cleve.GeneQCs, 0)

	
	pipeline = append(pipeline, 
		bson.D{
			{Key: "$unwind", Value: "$genepanel"},
		},
		bson.D{
			{Key: "$unwind", Value: "$genepanel.genes"},
		},
		bson.D{
			{"$group", bson.D{
				{"_id", "$genepanel.genes.hgnc"},
				{"genes", bson.D{
					{"$push", bson.D{
						{"run_id", "$run_id"},
						{"hgnc", "$genepanel.genes.hgnc"},
						{"mean_coverage", "$genepanel.genes.mean_coverage"},
						{"mean_completness", "$genepanel.genes.mean_completness"},
						{"mean_mapping_quality", "$genepanel.genes.mean_mapping_quality"},
					}},
				}},
			}},
		},
		
		
	)


	if filter.HGNC != "" {
		pipeline = append(pipeline,
		bson.D{
			{
				Key: "$match",
				Value: bson.D{ 
					{Key: "genes.hgnc", Value: filter.HGNC},
				},
			},
		},
	
	)}

	cursor, err := db.PanelQcCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return panelqc_result, err
	}

	defer closeCursor(cursor, context.TODO())

	
	for cursor.Next(context.TODO()) {
		var run cleve.GeneQCs
		err = cursor.Decode(&run)
		if err != nil {
			return panelqc_result, err
		} 
		panelqc_result.GeneQcs = append(panelqc_result.GeneQcs, &run)
	}

	if len(panelqc_result.GeneQcs) == 0 {
		return panelqc_result, mongo.ErrNoDocuments
	}


	return panelqc_result, nil

}

func (db DB) GeneQc(hgnc string) (*cleve.GeneQCs, error) {
	
	filter := cleve.PanelQcFilter {
		HGNC: hgnc,
	}

	geneqc, err := db.GenesQCs(filter)
	if err != nil {
		return nil, err
	}

	return geneqc.GeneQcs[0], nil

}


func (db DB) ExonsQCs(filter cleve.PanelQcFilter) (cleve.ExonQcResult, error) {
	var (
		pipeline             mongo.Pipeline
		panelqc_result       cleve.ExonQcResult
	)

	panelqc_result.ExonQcs  = make([]*cleve.ExonQCs, 0)

	
	pipeline = append(pipeline, 
		bson.D{
			{Key: "$unwind", Value: "$genepanel"},
		},
		bson.D{
			{Key: "$unwind", Value: "$genepanel.genes"},
		},
		bson.D{
			{Key: "$unwind", Value: "$genepanel.genes.incomplete_exons"},
		},
		bson.D{
			{"$group", bson.D{
				{"_id", bson.D{
					{"hgnc", "$genepanel.genes.incomplete_exons.hgnc"},
					{"exon_number", "$genepanel.genes.incomplete_exons.exon_number"},
				}},
				{"exons", bson.D{
					{"$push", bson.D{
						{"run_id", "$run_id"},
						{"hgnc", "$genepanel.genes.incomplete_exons.hgnc"},
						{"exon_number", "$genepanel.genes.incomplete_exons.exon_number"},
						{"mean_coverage_exon", "$genepanel.genes.incomplete_exons.mean_coverage_exon"},
						{"mean_completness_exon", "$genepanel.genes.incomplete_exons.mean_completness_exon"},
						{"mean_mapping_quality_exon", "$genepanel.genes.incomplete_exons.mean_mapping_quality_exon"},
					}},
				}},
			}},
		},
	)

	pipeline = append(pipeline, bson.D{
		{Key: "$sort", Value: bson.D{
			{Key: "exons.exon_number", Value: 1},
		}},
	})

	if filter.Exon_number != 0 {
		pipeline = append(pipeline,
		bson.D{
			{
				Key: "$match",
				Value: bson.D{ 
					{Key: "exons.exon_number", Value: filter.Exon_number},
				},
			},
		},
	)}

	cursor, err := db.PanelQcCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return panelqc_result, err
	}

	defer closeCursor(cursor, context.TODO())

	
	for cursor.Next(context.TODO()) {
		var run cleve.ExonQCs
		err = cursor.Decode(&run)
		if err != nil {
			return panelqc_result, err
		} 

		panelqc_result.ExonQcs = append(panelqc_result.ExonQcs, &run)
	}

	if len(panelqc_result.ExonQcs) == 0 {
		return panelqc_result, mongo.ErrNoDocuments
	}

	return panelqc_result, nil

}

func (db DB) ExonQc(exon_number float64) (*cleve.ExonQCs, error) {
	
	filter := cleve.PanelQcFilter {
		Exon_number: exon_number,
	}

	exonqc, err := db.ExonsQCs(filter)
	if err != nil {
		return nil, err
	}

	return exonqc.ExonQcs[0], nil

}


func (db DB) CreatePanelQc(p cleve.PanelQc) error {

	p.Created = time.Now()

	if _, err := db.PanelQcCollection().InsertOne(context.TODO(), p); err != nil {
		return err
	}
	return nil
}

func (db DB) DeletePanelQc(runId string) error {

	res, err := db.PanelQcCollection().DeleteOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}})
	if err == nil && res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (db DB) PanelQcIndex() ([]map[string]string, error) {
	cursor, err := db.PanelQcCollection().Indexes().List(context.TODO())
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


func (db DB) SetPanelQcIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	res, err := db.PanelQcCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := db.PanelQcCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
	
}
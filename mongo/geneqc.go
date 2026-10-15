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
func (db DB) AllGeneQCs(filter cleve.PanelQcFilter) (cleve.AllGenesQcResult, error) {
	var (
		pipeline             mongo.Pipeline
		panelqc_result       cleve.AllGenesQcResult
	)

	panelqc_result.AllGenes = make([]*cleve.GeneQc, 0)

	pipeline = append(pipeline,
		bson.D{
			{"$unwind", "$genes"},
		},
		bson.D{
			{"$unwind", bson.D{
				{"path", "$genes.incomplete_exons"},
				{"preserveNullAndEmptyArrays", true},
			}},
		},

		// Calculate averages per gene and exon
		bson.D{
			{"$group", bson.D{
				{"_id", bson.D{
					{"symbol", "$genes.symbol"},
					{"exon_number", "$genes.incomplete_exons.exon_number"},
				}},

				{"hgnc", bson.D{{"$first", "$genes.hgnc"}}},
				{"symbol", bson.D{{"$first", "$genes.symbol"}}},
				{"gene_type", bson.D{{"$first", "$genes.gene_type"}}},
				{"segment_duplication", bson.D{{"$first", "$genes.segment_duplication"}}},
				{"pseudogene", bson.D{{"$first", "$genes.pseudogene"}}},
				{"pathogenic_variants", bson.D{{"$first", "$genes.pathogenic_variants"}}},

				{"mean_coverage", bson.D{{"$avg", "$genes.mean_coverage"}}},
				{"mean_completness", bson.D{{"$avg", "$genes.mean_completness"}}},
				{"mean_mapping_quality", bson.D{{"$avg", "$genes.mean_mapping_quality"}}},

				{"genomic_position_exon", bson.D{{"$first", "$genes.incomplete_exons.genomic_position_exon"}}},
				{"transcript", bson.D{{"$first", "$genes.incomplete_exons.transcript"}}},
				{"segment_duplication_exon", bson.D{{"$first", "$genes.incomplete_exons.segment_duplication_exon"}}},
				{"suboptional_coverage_exon", bson.D{{"$first", "$genes.incomplete_exons.suboptional_coverage_exon"}}},

				{"mean_mapping_quality_exon", bson.D{{"$avg", "$genes.incomplete_exons.mean_mapping_quality_exon"}}},
				{"mean_completness_exon", bson.D{{"$avg", "$genes.incomplete_exons.mean_completness_exon"}}},
				{"mean_coverage_exon", bson.D{{"$avg", "$genes.incomplete_exons.mean_coverage_exon"}}},
			}}},
		
		bson.D{
			{"$sort", bson.D{{"_id.exon_number", 1}},},
		},

		// Group back by gene
		bson.D{
			{"$group", bson.D{
				{"_id", "$_id.symbol"},

				{"hgnc", bson.D{{"$first", "$hgnc"}}},
				{"symbol", bson.D{{"$first", "$symbol"}}},
				{"gene_type", bson.D{{"$first", "$gene_type"}}},
				{"segment_duplication", bson.D{{"$first", "$segment_duplication"}}},
				{"pseudogene", bson.D{{"$first", "$pseudogene"}}},
				{"pathogenic_variants", bson.D{{"$first", "$pathogenic_variants"}}},

				{"mean_coverage", bson.D{{"$first", "$mean_coverage"}}},
				{"mean_completness", bson.D{{"$first", "$mean_completness"}}},
				{"mean_mapping_quality", bson.D{{"$first", "$mean_mapping_quality"}}},
				
				{"exons", bson.D{
					{"$push", bson.D{
						{"exon_number", "$_id.exon_number"},
						{"genomic_position_exon", "$genomic_position_exon"},
						{"transcript", "$transcript"},
						{"segment_duplication_exon", "$segment_duplication_exon"},
						{"mean_mapping_quality_exon", bson.D{
							{"$round", bson.A{"$mean_mapping_quality_exon", 2}},
						}},
						{"mean_completness_exon", bson.D{
							{"$round", bson.A{"$mean_completness_exon", 2}},
						}},
						{"mean_coverage_exon", bson.D{
							{"$round", bson.A{"$mean_coverage_exon", 2}},
						}},
						{"suboptional_coverage_exon", bson.D{
							{"$not", bson.A{
								bson.D{
									{"$gte", bson.A{"$mean_completness_exon", 20}},
								},
						}}}},
					}},
				}},
			}},
		},

		bson.D{
			{"$sort", bson.D{{"symbol", 1}},},
		},
		// Remove the empty structs of incomplete_exons
		bson.D{
			{"$addFields", bson.D{
				{"incomplete_exons", bson.D{
					{"$filter", bson.D{
						{"input", "$exons"},
						{"as", "e"},
						{"cond", bson.D{
							{"$ne", bson.A{"$$e.transcript", nil}},
						}},
					}},
				}},
			}},
			},

		bson.D{
			{"$set", bson.D{
				{"mean_coverage", bson.D{
					{"$round", bson.A{"$mean_coverage", 2}},
				}},
				{"mean_completness", bson.D{
					{"$round", bson.A{"$mean_completness", 2}},
				}},
				{"mean_mapping_quality", bson.D{
					{"$round", bson.A{"$mean_mapping_quality", 2}},
				}},

			}},
		},
		// Set suboptimal coverage based on the mean value of mean_completness
		bson.D{
			{"$set", bson.D{
				{"suboptional_coverage", bson.D{
					{"$not", bson.A{
						bson.D{
							{"$gte", bson.A{"$mean_completness", 20}},
						},
					}},
				}},
			}},
		},


	)


	

	if filter.Symbol != "" {
		pipeline = append(pipeline,
		bson.D{
			{
				Key: "$match",
				Value: bson.D{ 
					{Key: "symbol", Value: filter.Symbol},
				},
			},
		},
	
	)}


	cursor, err := db.GeneQcCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return panelqc_result, err
	}

	defer closeCursor(cursor, context.TODO())

	
	for cursor.Next(context.TODO()) {
		var run cleve.GeneQc
		err = cursor.Decode(&run)
		if err != nil {
			return panelqc_result, err
		}
		panelqc_result.AllGenes = append(panelqc_result.AllGenes, &run)
	}

	if len(panelqc_result.AllGenes) == 0 {
		return panelqc_result, mongo.ErrNoDocuments
	}
	

	return panelqc_result, nil

}

func (db DB) AllGeneQc(symbol string) (*cleve.GeneQc , error) {
	
	filter := cleve.PanelQcFilter {
		Symbol: symbol,
	}

	panelqc, err := db.AllGeneQCs(filter)
	if err != nil {
		return nil, err
	}
	
	return panelqc.AllGenes[0], nil

}

func (db DB) GenesQCs(filter cleve.PanelQcFilter) (cleve.GenesQcResult, error) {
	var (
		pipeline             mongo.Pipeline
		panelqc_result       cleve.GenesQcResult
	)

	panelqc_result.GenesQcs = make([]*cleve.GeneQCs, 0)


	pipeline = append(pipeline, 
		bson.D{
			{Key: "$unwind", Value: "$genes"},
		},
		bson.D{
			{"$group", bson.D{
				{"_id", "$genes.symbol"},
				{"genes", bson.D{
					{"$push", bson.D{
						{"run_id", "$run_id"},
						{"symbol", "$genes.symbol"},
						{"mean_coverage", "$genes.mean_coverage"},
						{"mean_completness", "$genes.mean_completness"},
						{"mean_mapping_quality", "$genes.mean_mapping_quality"},
					}},
				}},
			}},
		},	
	)

	if filter.Symbol != "" {
		pipeline = append(pipeline,
		bson.D{
			{
				Key: "$match",
				Value: bson.D{ 
					{Key: "genes.symbol", Value: filter.Symbol},
				},
			},
		},
	
	)}

	

	cursor, err := db.GeneQcCollection().Aggregate(context.TODO(), pipeline)
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
		panelqc_result.GenesQcs = append(panelqc_result.GenesQcs, &run)
	}
	
	if len(panelqc_result.GenesQcs) == 0 {
		return panelqc_result, mongo.ErrNoDocuments
	}


	return panelqc_result, nil

}

func (db DB) GeneQc(symbol string) (*cleve.GeneQCs, error) {
	
	filter := cleve.PanelQcFilter {
		Symbol: symbol,
	}

	geneqc, err := db.GenesQCs(filter)
	if err != nil {
		return nil, err
	}

	return geneqc.GenesQcs[0], nil

}

func (db DB) ExonsQCs(filter cleve.PanelQcFilter) (cleve.ExonQcResult, error) {
	var (
		pipeline             mongo.Pipeline
		panelqc_result       cleve.ExonQcResult
	)

	panelqc_result.ExonQcs  = make([]*cleve.ExonQCs, 0)

	
	pipeline = append(pipeline, 
		bson.D{
			{Key: "$unwind", Value: "$genes"},
		},
		bson.D{
			{Key: "$unwind", Value: "$genes.incomplete_exons"},
		},
		bson.D{
			{"$group", bson.D{
				{"_id", bson.D{
					{"hgnc", "$genes.hgnc"},
					{"exon_number", "$genes.incomplete_exons.exon_number"},
				}},
				{"exons", bson.D{
					{"$push", bson.D{
						{"run_id", "$run_id"},
						{"hgnc", "$genes.hgnc"},
						{"symbol", "$genes.symbol"},
						{"exon_number", "$genes.incomplete_exons.exon_number"},
						{"mean_coverage_exon", "$genes.incomplete_exons.mean_coverage_exon"},
						{"mean_completness_exon", "$genes.incomplete_exons.mean_completness_exon"},
						{"mean_mapping_quality_exon", "$genes.incomplete_exons.mean_mapping_quality_exon"},
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

	if ((filter.Exon_number != 0) && (filter.Symbol != "")) {
		pipeline = append(pipeline,
		bson.D{
			{
				Key: "$match",
				Value: bson.D{ 
					{Key: "exons.exon_number", Value: filter.Exon_number},
					{Key: "exons.symbol", Value: filter.Symbol},
				},
			},
		},
	)}

	cursor, err := db.GeneQcCollection().Aggregate(context.TODO(), pipeline)
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

func (db DB) ExonQc(symbol string, exon_number float64) (*cleve.ExonQCs, error) {
	
	filter := cleve.PanelQcFilter {
		Symbol: symbol,
		Exon_number: exon_number,
		
	}

	exonqc, err := db.ExonsQCs(filter)
	if err != nil {
		return nil, err
	}

	return exonqc.ExonQcs[0], nil

}



func (db DB) CreateGeneQc(p cleve.GenesQc) error {

	p.RunDate = time.Now()

	if _, err := db.GeneQcCollection().InsertOne(context.TODO(), p); err != nil {
		return err
	}
	return nil
}

func (db DB) DeleteGeneQc(runId string) error {

	res, err := db.GeneQcCollection().DeleteOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}})
	if err == nil && res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (db DB) GeneQcIndex() ([]map[string]string, error) {
	cursor, err := db.GeneQcCollection().Indexes().List(context.TODO())
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


func (db DB) SetGeneQcIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	res, err := db.GeneQcCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := db.GeneQcCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
	
}
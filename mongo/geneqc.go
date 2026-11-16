package mongo

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db DB) AllGeneQCs(filter cleve.PanelQcFilter) (cleve.AllGenesQcResult, error) {
	var (
		pipeline      mongo.Pipeline
		geneqc_result cleve.AllGenesQcResult
	)

	geneqc_result.AllGenes = make([]*cleve.GeneQc, 0)

	geneqc_result.PaginationMetadata = cleve.PaginationMetadata{
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}

	pipeline = append(pipeline,
		// Important to receive the last retrieved value first later down
		bson.D{
			{
				Key: "$sort",
				Value: bson.D{
					{Key: "date", Value: -1},
				},
			},
		},
		bson.D{
			{
				Key:   "$unwind",
				Value: "$genes",
			},
		},
		bson.D{
			{
				Key: "$unwind",
				Value: bson.D{
					{Key: "path", Value: "$genes.incomplete_exons"},
					{Key: "preserveNullAndEmptyArrays", Value: true},
				},
			},
		},

		// Calculate averages per gene and exon
		bson.D{
			{
				Key: "$group",
				Value: bson.D{
					{
						Key: "_id",
						Value: bson.D{
							{Key: "symbol", Value: "$genes.symbol"},
							{Key: "exon_number", Value: "$genes.incomplete_exons.exon_number"},
							{Key: "transcript", Value: "$genes.incomplete_exons.transcript"},
						},
					},

					{Key: "hgnc", Value: bson.D{{Key: "$first", Value: "$genes.hgnc"}}},
					{Key: "symbol", Value: bson.D{{Key: "$first", Value: "$genes.symbol"}}},
					{Key: "gene_type", Value: bson.D{{Key: "$first", Value: "$genes.gene_type"}}},
					{Key: "segment_duplication", Value: bson.D{{Key: "$first", Value: "$genes.segment_duplication"}}},
					{Key: "pseudogene", Value: bson.D{{Key: "$first", Value: "$genes.pseudogene"}}},
					{Key: "pathogenic_variants", Value: bson.D{{Key: "$first", Value: "$genes.pathogenic_variants"}}},

					{Key: "mean_coverage", Value: bson.D{{Key: "$avg", Value: "$genes.mean_coverage"}}},
					{Key: "mean_completness", Value: bson.D{{Key: "$avg", Value: "$genes.mean_completness"}}},
					{Key: "mean_mapping_quality", Value: bson.D{{Key: "$avg", Value: "$genes.mean_mapping_quality"}}},

					{Key: "genomic_position_exon", Value: bson.D{{Key: "$first", Value: "$genes.incomplete_exons.genomic_position_exon"}}},
					{Key: "transcript", Value: bson.D{{Key: "$first", Value: "$genes.incomplete_exons.transcript"}}},
					{Key: "select_type", Value: bson.D{{Key: "$first", Value: "$genes.incomplete_exons.select_type"}}},
					{Key: "segment_duplication_exon", Value: bson.D{{Key: "$first", Value: "$genes.incomplete_exons.segment_duplication_exon"}}},
					{Key: "suboptional_coverage_exon", Value: bson.D{{Key: "$first", Value: "$genes.incomplete_exons.suboptional_coverage_exon"}}},

					{Key: "mean_mapping_quality_exon", Value: bson.D{{Key: "$avg", Value: "$genes.incomplete_exons.mean_mapping_quality_exon"}}},
					{Key: "mean_completness_exon", Value: bson.D{{Key: "$avg", Value: "$genes.incomplete_exons.mean_completness_exon"}}},
					{Key: "mean_coverage_exon", Value: bson.D{{Key: "$avg", Value: "$genes.incomplete_exons.mean_coverage_exon"}}},
				},
			},
		},

		bson.D{
			{
				Key: "$sort",
				Value: bson.D{
					{Key: "_id.exon_number", Value: 1},
				},
			},
		},

		// Group back by gene
		bson.D{
			{
				Key: "$group",
				Value: bson.D{
					{Key: "_id", Value: "$_id.symbol"},

					{Key: "hgnc", Value: bson.D{{Key: "$first", Value: "$hgnc"}}},
					{Key: "symbol", Value: bson.D{{Key: "$first", Value: "$symbol"}}},
					{Key: "gene_type", Value: bson.D{{Key: "$first", Value: "$gene_type"}}},
					{Key: "segment_duplication", Value: bson.D{{Key: "$first", Value: "$segment_duplication"}}},
					{Key: "pseudogene", Value: bson.D{{Key: "$first", Value: "$pseudogene"}}},
					{Key: "pathogenic_variants", Value: bson.D{{Key: "$first", Value: "$pathogenic_variants"}}},

					{Key: "mean_coverage", Value: bson.D{{Key: "$first", Value: "$mean_coverage"}}},
					{Key: "mean_completness", Value: bson.D{{Key: "$first", Value: "$mean_completness"}}},
					{Key: "mean_mapping_quality", Value: bson.D{{Key: "$first", Value: "$mean_mapping_quality"}}},

					{
						Key: "exons",
						Value: bson.D{
							{
								Key: "$push",
								Value: bson.D{
									{Key: "exon_number", Value: "$_id.exon_number"},
									{Key: "genomic_position_exon", Value: "$genomic_position_exon"},
									{Key: "transcript", Value: "$transcript"},
									{Key: "select_type", Value: "$select_type"},
									{Key: "segment_duplication_exon", Value: "$segment_duplication_exon"},

									{
										Key: "mean_mapping_quality_exon",
										Value: bson.D{
											{
												Key: "$round",
												Value: bson.A{
													"$mean_mapping_quality_exon",
													2,
												},
											},
										},
									},

									{
										Key: "mean_completness_exon",
										Value: bson.D{
											{
												Key: "$round",
												Value: bson.A{
													"$mean_completness_exon",
													2,
												},
											},
										},
									},

									{
										Key: "mean_coverage_exon",
										Value: bson.D{
											{
												Key: "$round",
												Value: bson.A{
													"$mean_coverage_exon",
													2,
												},
											},
										},
									},

									{
										Key: "suboptional_coverage_exon",
										Value: bson.D{
											{
												Key: "$not",
												Value: bson.A{
													bson.D{
														{
															Key: "$gte",
															Value: bson.A{
																"$mean_completness_exon",
																10,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},

		bson.D{
			{
				Key: "$sort",
				Value: bson.D{
					{Key: "symbol", Value: 1},
				},
			},
		},

		// Remove the empty structs of incomplete_exons
		bson.D{
			{
				Key: "$addFields",
				Value: bson.D{
					{
						Key: "incomplete_exons",
						Value: bson.D{
							{
								Key: "$filter",
								Value: bson.D{
									{Key: "input", Value: "$exons"},
									{Key: "as", Value: "e"},
									{
										Key: "cond",
										Value: bson.D{
											{
												Key: "$ne",
												Value: bson.A{
													"$$e.transcript",
													nil,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},

		bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{
						Key: "mean_coverage",
						Value: bson.D{
							{
								Key: "$round",
								Value: bson.A{
									"$mean_coverage",
									2,
								},
							},
						},
					},
					{
						Key: "mean_completness",
						Value: bson.D{
							{
								Key: "$round",
								Value: bson.A{
									"$mean_completness",
									2,
								},
							},
						},
					},
					{
						Key: "mean_mapping_quality",
						Value: bson.D{
							{
								Key: "$round",
								Value: bson.A{
									"$mean_mapping_quality",
									2,
								},
							},
						},
					},
				},
			},
		},

		// Set suboptimal coverage based on mean value of mean_completness
		bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{
						Key: "suboptional_coverage",
						Value: bson.D{
							{
								Key: "$not",
								Value: bson.A{
									bson.D{
										{
											Key: "$gte",
											Value: bson.A{
												"$mean_completness",
												10,
											},
										},
									},
								},
							},
						},
					},
				},
			},
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
		)
	}

	metaPipeline := append(pipeline, bson.D{
		{Key: "$count", Value: "total_count"},
	})

	cursor, err := db.GeneQcCollection().Aggregate(context.TODO(), metaPipeline)
	if err != nil {
		return geneqc_result, err
	}
	if !cursor.Next(context.TODO()) {
		geneqc_result.TotalCount = 0
	}
	if err := cursor.Decode(&geneqc_result.PaginationMetadata); geneqc_result.TotalCount > 0 && err != nil {
		return geneqc_result, err
	}
	if geneqc_result.PageSize > 0 {
		geneqc_result.TotalPages = geneqc_result.TotalCount / geneqc_result.PageSize
		if geneqc_result.TotalCount%geneqc_result.PageSize > 0 {
			geneqc_result.TotalPages += 1
		}
	} else {
		geneqc_result.TotalPages = 1
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

	cursor, err = db.GeneQcCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return geneqc_result, err
	}

	defer closeCursor(cursor, context.TODO())

	for cursor.Next(context.TODO()) {
		var run cleve.GeneQc
		err = cursor.Decode(&run)
		if err != nil {
			return geneqc_result, err
		}
		geneqc_result.Count++
		geneqc_result.AllGenes = append(geneqc_result.AllGenes, &run)
	}

	if geneqc_result.TotalCount == 0 {

		geneqc_result.TotalPages = 1
		geneqc_result.AllGenes = make([]*cleve.GeneQc, 0)
	}

	if geneqc_result.Page > geneqc_result.TotalPages {
		return geneqc_result, PageOutOfBoundsError{
			page:       geneqc_result.Page,
			totalPages: geneqc_result.TotalPages,
		}
	}

	return geneqc_result, nil
}

func (db DB) AllGeneQc(symbol string) (*cleve.GeneQc, error) {
	filter := cleve.PanelQcFilter{
		Symbol: symbol,
	}

	geneqc, err := db.AllGeneQCs(filter)
	if err != nil {
		return nil, err
	}
	if geneqc.Count == 0 {
		return nil, mongo.ErrNoDocuments
	}
	if geneqc.Count > 1 {
		// We don't expect more than one matching symbol.
		return nil, fmt.Errorf("found more than one matching symbol")
	}

	return geneqc.AllGenes[0], nil
}

func (db DB) GenesQCs(filter cleve.PanelQcFilter) (cleve.GenesQcResult, error) {
	var (
		pipeline      mongo.Pipeline
		geneqc_result cleve.GenesQcResult
	)

	geneqc_result.GenesQcs = make([]*cleve.GeneQCs, 0)

	pipeline = append(pipeline,
		bson.D{
			{Key: "$sort", Value: bson.D{
				{Key: "date", Value: 1},
			}},
		},
		bson.D{
			{Key: "$unwind", Value: "$genes"},
		},
		bson.D{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$genes.symbol"},
				{Key: "genes", Value: bson.D{
					{Key: "$push", Value: bson.D{
						{Key: "run_id", Value: "$run_id"},
						{Key: "symbol", Value: "$genes.symbol"},
						{Key: "mean_coverage", Value: "$genes.mean_coverage"},
						{Key: "mean_completness", Value: "$genes.mean_completness"},
						{Key: "mean_mapping_quality", Value: "$genes.mean_mapping_quality"},
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
		)
	}

	if filter.RunId != "" {
		pipeline = append(pipeline,
			bson.D{
				{
					Key: "$match",
					Value: bson.D{
						{Key: "genes.run_id", Value: filter.RunId},
					},
				},
			},
		)
	}

	cursor, err := db.GeneQcCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return geneqc_result, err
	}

	defer closeCursor(cursor, context.TODO())

	for cursor.Next(context.TODO()) {
		var run cleve.GeneQCs
		err = cursor.Decode(&run)
		if err != nil {
			return geneqc_result, err
		}
		geneqc_result.GenesQcs = append(geneqc_result.GenesQcs, &run)
	}

	if len(geneqc_result.GenesQcs) == 0 {
		return geneqc_result, mongo.ErrNoDocuments
	}

	return geneqc_result, nil
}

func (db DB) GeneQc(symbol string) (*cleve.GeneQCs, error) {
	filter := cleve.PanelQcFilter{
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
		pipeline      mongo.Pipeline
		geneqc_result cleve.ExonQcResult
	)

	geneqc_result.ExonQcs = make([]*cleve.ExonQCs, 0)

	pipeline = append(pipeline,
		bson.D{
			{Key: "$sort", Value: bson.D{
				{Key: "date", Value: 1},
			}},
		},
		bson.D{
			{Key: "$unwind", Value: "$genes"},
		},
		bson.D{
			{Key: "$unwind", Value: "$genes.incomplete_exons"},
		},
		bson.D{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: bson.D{
					{Key: "hgnc", Value: "$genes.hgnc"},
					{Key: "exon_number", Value: "$genes.incomplete_exons.exon_number"},
					{Key: "transcript", Value: "$genes.incomplete_exons.transcript"},
				}},
				{Key: "exons", Value: bson.D{
					{Key: "$push", Value: bson.D{
						{Key: "run_id", Value: "$run_id"},
						{Key: "hgnc", Value: "$genes.hgnc"},
						{Key: "symbol", Value: "$genes.symbol"},
						{Key: "exon_number", Value: "$genes.incomplete_exons.exon_number"},
						{Key: "transcript", Value: "$genes.incomplete_exons.transcript"},
						{Key: "mean_coverage_exon", Value: "$genes.incomplete_exons.mean_coverage_exon"},
						{Key: "mean_completness_exon", Value: "$genes.incomplete_exons.mean_completness_exon"},
						{Key: "mean_mapping_quality_exon", Value: "$genes.incomplete_exons.mean_mapping_quality_exon"},
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

	if (filter.Exon_number != 0) && (filter.Symbol != "") && (filter.Transcript != "") {
		pipeline = append(pipeline,
			bson.D{
				{
					Key: "$match",
					Value: bson.D{
						{Key: "exons.exon_number", Value: filter.Exon_number},
						{Key: "exons.transcript", Value: filter.Transcript},
						{Key: "exons.symbol", Value: filter.Symbol},
					},
				},
			},
		)
	}

	cursor, err := db.GeneQcCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return geneqc_result, err
	}

	defer closeCursor(cursor, context.TODO())

	for cursor.Next(context.TODO()) {
		var run cleve.ExonQCs
		err = cursor.Decode(&run)
		if err != nil {
			return geneqc_result, err
		}

		geneqc_result.ExonQcs = append(geneqc_result.ExonQcs, &run)
	}

	if len(geneqc_result.ExonQcs) == 0 {
		return geneqc_result, mongo.ErrNoDocuments
	}

	return geneqc_result, nil
}

func (db DB) ExonQc(symbol string, exon_number float64, transcript string) (*cleve.ExonQCs, error) {
	filter := cleve.PanelQcFilter{
		Symbol:      symbol,
		Exon_number: exon_number,
		Transcript:  transcript,
	}

	exonqc, err := db.ExonsQCs(filter)
	if err != nil {
		return nil, err
	}

	return exonqc.ExonQcs[0], nil
}

func (db DB) CreateGeneQc(p cleve.GenesQc) error {
	// Retrieve date from RunID if it exists, if not -> take today's date
	date := strings.SplitN(p.RunId, "_", 2)[0]

	if dateInt, err := strconv.Atoi(date); err == nil {
		p.Date = dateInt
	} else {
		todayInt, err := strconv.Atoi(time.Now().Format("20060102"))
		if err != nil {
			return err
		}
		p.Date = todayInt
	}

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

func (db DB) UpdateGeneQc(runId string, geneinfo cleve.GeneQc) error {
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "genes", Value: geneinfo}}}}
	result, err := db.GeneQcCollection().UpdateOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}, update)
	if err == nil && result.MatchedCount == 0 {
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

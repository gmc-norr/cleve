package cleve

type QcResultItem struct {
	InteropQC `bson:",inline" json:",inline"`
	Run       Run `bson:"run" json:"run"`
}

type QcResult struct {
	PaginationMetadata `bson:"metadata" json:"metadata"`
	Qc                 []QcResultItem `bson:"qc" json:"qc"`
}

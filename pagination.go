package cleve

type PaginationMetadata struct {
	TotalCount int `bson:"total_count" json:"total_count"`
	Count      int `bson:"count" json:"count"`
	Page       int `bson:"page" json:"page"`
	PageSize   int `bson:"page_size" json:"page_size"`
	TotalPages int `bson:"total_pages" json:"total_pages"`
}

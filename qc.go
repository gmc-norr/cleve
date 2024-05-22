package cleve

import (
	"time"

	"github.com/gmc-norr/cleve/interop"
)

type QcFilter struct {
	RunID     string
	Platform  string
	StartDate time.Time
	EndDate   time.Time
	Page      int
	PageSize  int
}

type QcResultItem struct {
	interop.InteropSummary `bson:",inline" json:",inline"`
	Run Run `bson:"run" json:"run"`
}

type QcResult struct {
	RunMetadata `bson:"metadata" json:"metadata"`
	Qc          []QcResultItem `bson:"qc" json:"qc"`
}

type RunQcService interface {
	Create(string, *interop.InteropSummary) error
	All(QcFilter) (QcResult, error)
	Get(string) (*interop.InteropSummary, error)
	GetTotalQ30(string) (float64, error)
	GetTotalErrorRate(string) (float64, error)
	GetIndex() ([]map[string]string, error)
	SetIndex() (string, error)
}

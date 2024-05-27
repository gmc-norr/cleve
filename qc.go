package cleve

import (
	"fmt"
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

func (f QcFilter) UrlParams() string {
	s := "?"
	sep := ""

	if f.RunID != "" {
		s = fmt.Sprintf("%s%srun_id=%s", s, sep, f.RunID)
		sep = "&"
	}

	if f.Platform != "" {
		s = fmt.Sprintf("%s%splatform=%s", s, sep, f.Platform)
		sep = "&"
	}

	if f.Page != 0 {
		s = fmt.Sprintf("%s%spage=%d", s, sep, f.Page)
		sep = "&"
	}

	if f.PageSize != 0 {
		s = fmt.Sprintf("%s%spage_size=%d", s, sep, f.PageSize)
		sep = "&"
	}

	return s
}

type QcResultItem struct {
	interop.InteropSummary `bson:",inline" json:",inline"`
	Run                    Run `bson:"run" json:"run"`
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

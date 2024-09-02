package cleve

import (
	"fmt"
	"time"
)

// Pagination filtering.
type PaginationFilter struct {
	Page     int
	PageSize int
}

// Run filtering.
type RunFilter struct {
	RunID    string
	Brief    bool
	Platform string
	State    string
	From     time.Time
	To       time.Time
	PaginationFilter
}

// Convert a run filter to URL query parameters.
func (f RunFilter) UrlParams() string {
	p := "?"
	sep := ""
	if f.RunID != "" {
		p += fmt.Sprintf("%srun_id=%s", sep, f.RunID)
		sep = "&"
	}
	if f.Platform != "" {
		p += fmt.Sprintf("%splatform=%s", sep, f.Platform)
		sep = "&"
	}
	if f.State != "" {
		p += fmt.Sprintf("%sstate=%s", sep, f.State)
		sep = "&"
	}
	if f.Page != 0 {
		p += fmt.Sprintf("%spage=%d", sep, f.Page)
		sep = "&"
	}
	return p
}

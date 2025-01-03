package cleve

import (
	"fmt"
	"time"
)

// Pagination filtering.
type PaginationFilter struct {
	Page     int `form:"page,default=1"`
	PageSize int `form:"page_size"`
}

func (f PaginationFilter) Validate() error {
	if f.Page < 1 {
		return fmt.Errorf("illegal page number %d", f.Page)
	}
	if f.PageSize < 0 {
		return fmt.Errorf("illegal page size %d", f.PageSize)
	}
	return nil
}

// Run filtering.
type RunFilter struct {
	RunID            string    `form:"run_id"`
	RunIdQuery       string    `form:"run_id_query"`
	Brief            bool      `form:"brief"`
	Platform         string    `form:"platform"`
	State            string    `form:"state"`
	From             time.Time `form:"from"`
	To               time.Time `form:"to"`
	PaginationFilter `form:",inline"`
}

func NewRunFilter() RunFilter {
	return RunFilter{
		PaginationFilter: PaginationFilter{
			PageSize: 10,
		},
	}
}

// Convert a run filter to URL query parameters.
func (f RunFilter) UrlParams() string {
	p := "?"
	sep := ""
	if f.RunID != "" {
		p += fmt.Sprintf("%srun_id=%s", sep, f.RunID)
		sep = "&"
	}
	if f.RunIdQuery != "" {
		p += fmt.Sprintf("%srun_id_query=%s", sep, f.RunIdQuery)
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
	}
	return p
}

// QC filtering.
type QcFilter struct {
	RunIdQuery string    `form:"run_id_query"`
	Platform   string    `form:"platform"`
	StartDate  time.Time `form:"start_time"`
	EndDate    time.Time `form:"end_time"`
	PaginationFilter
}

func NewQcFilter() QcFilter {
	return QcFilter{
		PaginationFilter: PaginationFilter{
			PageSize: 5,
		},
	}
}

func (f QcFilter) UrlParams() string {
	s := "?"
	sep := ""

	if f.RunIdQuery != "" {
		s = fmt.Sprintf("%s%srun_id_query=%s", s, sep, f.RunIdQuery)
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
	}

	return s
}

// Sample filtering.
type SampleFilter struct {
	Name             string `form:"sample_name"`
	Id               string `form:"sample_id"`
	RunId            string `form:"run_id"`
	Analysis         string `form:"analysis"`
	PaginationFilter `form:",inline"`
}

func NewSampleFilter() SampleFilter {
	return SampleFilter{
		PaginationFilter: PaginationFilter{
			PageSize: 10,
		},
	}
}

// Convert a sample filter to URL query parameters.
func (f SampleFilter) UrlParams() string {
	p := "?"
	sep := ""
	if f.Name != "" {
		p += fmt.Sprintf("%sname=%s", sep, f.Name)
		sep = "&"
	}
	if f.Id != "" {
		p += fmt.Sprintf("%ssample_id=%s", sep, f.Id)
		sep = "&"
	}
	if f.RunId != "" {
		p += fmt.Sprintf("%srun_id=%s", sep, f.RunId)
		sep = "&"
	}
	if f.Analysis != "" {
		p += fmt.Sprintf("%sanalysis=%s", sep, f.Analysis)
	}
	return p
}

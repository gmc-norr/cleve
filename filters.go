package cleve

import (
	"fmt"
	"time"
)

// Pagination filtering.
type PaginationFilter struct {
	Page     int `form:"page,default=1"`
	PageSize int `form:"page_size,default=10"`
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
	Brief            bool      `form:"brief"`
	Platform         string    `form:"platform"`
	State            string    `form:"state"`
	From             time.Time `form:"from"`
	To               time.Time `form:"to"`
	PaginationFilter `form:",inline"`
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

// QC filtering.
type QcFilter struct {
	RunID     string
	Platform  string
	StartDate time.Time
	EndDate   time.Time
	PaginationFilter
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

// Sample filtering.
type SampleFilter struct {
	Name             string `form:"sample_name"`
	Id               string `form:"sample_id"`
	RunId            string `form:"run_id"`
	Analysis         string `form:"analysis"`
	PaginationFilter `form:",inline"`
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
		sep = "&"
	}
	return p
}

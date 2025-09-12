package cleve

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"time"
)

// Pagination filtering.
type PaginationFilter struct {
	Page     int `form:"page,default=1"`
	PageSize int `form:"page_size"`
}

func NewPaginationFilter() PaginationFilter {
	return PaginationFilter{
		Page:     1,
		PageSize: 10,
	}
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
	Platform         string    `form:"platform"`
	State            string    `form:"state"`
	From             time.Time `form:"from"`
	To               time.Time `form:"to"`
	PaginationFilter `form:",inline"`
}

func NewRunFilter() RunFilter {
	return RunFilter{
		PaginationFilter: NewPaginationFilter(),
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
	RunId      string    `form:"run_id"`
	RunIdQuery string    `form:"run_id_query"`
	Platform   string    `form:"platform"`
	StartDate  time.Time `form:"start_time"`
	EndDate    time.Time `form:"end_time"`
	PaginationFilter
}

func NewQcFilter() QcFilter {
	return QcFilter{
		PaginationFilter: NewPaginationFilter(),
	}
}

func (f QcFilter) UrlParams() string {
	s := "?"
	sep := ""

	if f.RunId != "" {
		s = fmt.Sprintf("%s%srun_id=%s", s, sep, f.RunId)
		sep = "&"
	}

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

// Analysis filtering
type AnalysisFilter struct {
	AnalysisId       string `form:"analysis_id"`
	RunId            string `form:"run_id"`
	Software         string `form:"software"`
	SoftwarePattern  string `form:"software_pattern"`
	State            State  `form:"state"`
	PaginationFilter `form:",inline"`
}

func NewAnalysisFilter() AnalysisFilter {
	return AnalysisFilter{
		PaginationFilter: NewPaginationFilter(),
	}
}

func (f *AnalysisFilter) Validate() error {
	errs := []error{f.PaginationFilter.Validate()}
	if f.Software != "" && f.SoftwarePattern != "" {
		errs = append(errs, fmt.Errorf("only one of software and software pattern can be used"))
	}
	return errors.Join(errs...)
}

// Analysis file filtering
type AnalysisFileFilter struct {
	AnalysisId string           `form:"analysis_id" bson:"analysis_id,omitempty" json:"analysis_id,omitzero"`
	RunId      string           `form:"run_id" bson:"run_id,omitempty" json:"run_id,omitzero"`
	FileType   AnalysisFileType `form:"type" bson:"type,omitempty" json:"type,omitzero"`
	Level      AnalysisLevel    `form:"level" bson:"level,omitempty" json:"level,omitzero"`
	ParentId   string           `form:"parent_id" bson:"parent_id,omitempty" json:"parent_id,omitzero"`
	Name       string           `form:"name" bson:"name,omitempty" json:"name,omitzero"`
	Pattern    *regexp.Regexp   `form:"-" bson:"-" json:"-"`
}

func NewAnalysisFileFilter() AnalysisFileFilter {
	return AnalysisFileFilter{}
}

func (f *AnalysisFileFilter) Validate() error {
	var errs []error
	if !f.FileType.IsValid() && f.ParentId == "" && f.Name == "" && f.Pattern == nil {
		errs = append(errs, fmt.Errorf("one of FileType, ParentId, Name or Pattern must be defined"))
	}
	if f.Pattern != nil && f.Name != "" {
		errs = append(errs, fmt.Errorf("cannot use both name and pattern"))
	}
	return errors.Join(errs...)
}

func (f *AnalysisFileFilter) Apply(file AnalysisFile) bool {
	pass := true
	if f.FileType.IsValid() && f.FileType != file.FileType {
		return false
	}
	if f.Level.IsValid() && f.Level != file.Level {
		return false
	}
	if f.ParentId != "" && f.ParentId != file.ParentId {
		return false
	}
	if f.Name != "" && f.Name != filepath.Base(file.Path) {
		return false
	}
	if f.Pattern != nil {
		pass = f.Pattern.Match([]byte(file.Path))
	}
	return pass
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
		PaginationFilter: NewPaginationFilter(),
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
	if f.Page != 0 {
		p = fmt.Sprintf("%s%spage=%d", p, sep, f.Page)
		sep = "&"
	}
	if f.PageSize != 0 {
		p = fmt.Sprintf("%s%spage_size=%d", p, sep, f.PageSize)
	}

	return p
}

type PanelFilter struct {
	Category  string `form:"category"`
	Name      string `form:"name"`
	NameQuery string `form:"name_query"`
	Gene      string `form:"gene"`
	GeneQuery string `form:"gene_query"`
	HGNC      string `form:"hgnc"`
	Version   string `form:"version"`
	Archived  bool   `form:"archived"`
}

func NewPanelFilter() PanelFilter {
	return PanelFilter{}
}

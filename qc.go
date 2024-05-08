package cleve

import "github.com/gmc-norr/cleve/interop"

type RunQcService interface {
	Create(string, *interop.InteropSummary) error
	Get(string) (*interop.InteropSummary, error)
	GetIndex() ([]map[string]string, error)
	SetIndex() (string, error)
}

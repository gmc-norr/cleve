package runparameters

import (
	"encoding/xml"
	"fmt"
	"log"
	"time"
)

type CustomTime struct {
	time.Time
}

func (c *CustomTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	const customLayout = "2006-01-02T15:04:05"
	var v string
	d.DecodeElement(&v, &start)
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		t, err = time.Parse(customLayout, v)
	}
	if err != nil {
		return err
	}
	c.Time = t
	return nil
}

type RunParameters interface {
	IsValid() bool
	GetExperimentName() string
	GetRunID() string
	Platform() string
}

func ParseRunParameters(paramsData []byte) (RunParameters, error) {
	var runParams RunParameters

	// Try the different platforms in order
	runParams = ParseNovaSeqRunParameters(paramsData)
	if runParams.IsValid() {
		log.Printf("Identified run as %s\n", runParams.Platform())
		return runParams, nil
	}

	log.Println("Invalid NovaSeq run parameters, trying NextSeq")
	runParams = ParseNextSeqRunParameters(paramsData)

	if !runParams.IsValid() {
		// Return an error if platform cannot reliably be determined
		return runParams, fmt.Errorf("Invalid run parameters")
	}

	log.Printf("Identified run as %s\n", runParams.Platform())
	return runParams, nil
}

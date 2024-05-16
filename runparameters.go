package cleve

import (
	"encoding/xml"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

type CustomTime time.Time

func (c *CustomTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	const customLayout1 = "2006-01-02T15:04:05"
	const customLayout2 = "2006-01-02T15:04:05-07:00"
	const customLayout3 = "2006-01-02"
	const customLayout4 = "060102"
	var v string
	d.DecodeElement(&v, &start)
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		t, err = time.Parse(customLayout1, v)
	}
	if err != nil {
		t, err = time.Parse(customLayout2, v)
	}
	if err != nil {
		t, err = time.Parse(customLayout3, v)
	}
	if err != nil {
		t, err = time.Parse(customLayout4, v)
	}
	if err != nil {
		return err
	}
	*c = CustomTime(t)
	return nil
}

func (c CustomTime) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(time.Time(c))
}

func (c *CustomTime) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	raw := bson.RawValue{
		Type:  t,
		Value: b,
	}

	var res time.Time
	if err := raw.Unmarshal(&res); err != nil {
		return err
	}

	*c = CustomTime(res)
	return nil
}

func (c CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *CustomTime) String() string {
	return time.Time(*c).Format(time.RFC3339)
}

func (c *CustomTime) Format(layout string) string {
	return time.Time(*c).Format(layout)
}

func (c CustomTime) Local() time.Time {
	return time.Time(c).Local()
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

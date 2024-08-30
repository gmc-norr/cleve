package cleve

// Represents a sequenced sample.
type Sample struct {
	// Sample name. If missing it should be set to the sample ID.
	Name string `bson:"name" json:"name"`
	// Sample ID as listed in the samplesheet.
	Id string `bson:"id" json:"id"`
	// Paths to fastq files related to the sample.
	Fastq []string `bson:"fastq" json:"fastq"`
	// Analyses associated with the sample.
	Analyses []*SampleAnalysis `bson:"analyses" json:"analyses"`
}

// Pipeline represents an analysis pipeline.
type Pipeline struct {
	Name    string `bson:"name" json:"name"`
	Version string `bson:"version" json:"version"`
	URL     string `bson:"url" json:"url"`
}

// SampleAnalysis represents a collection of analysis results from an analysis pipeline.
type SampleAnalysis struct {
	Pipeline `bson:"pipeline" json:"pipeline"`
	Results  []SampleAnalysisResult `bson:"path" json:"path"`
}

// SampleAnalysisResult is a specific result from an analysis pipeline.
type SampleAnalysisResult struct {
	Type        string   `bson:"type" json:"type"`
	Description string   `bson:"description" json:"description"`
	Path        []string `bson:"path" json:"path"`
}

package cleve

type Platform struct {
	Name         string `bson:"name" json:"name" binding:"required"`
	SerialTag    string `bson:"serial_tag" json:"serial_tag" binding:"required"`
	SerialPrefix string `bson:"serial_prefix" json:"serial_prefix" binding:"required"`
	ReadyMarker  string `bson:"ready_marker" json:"ready_marker"`
}

func NewPlatform(name, serialTag, serialPrefix, readyMarker string) *Platform {
	return &Platform{
		name,
		serialTag,
		serialPrefix,
		readyMarker,
	}
}

package cleve

type Platform struct {
	Name        string `bson:"name" json:"name" binding:"required"`
	ReadyMarker string `bson:"ready_marker" json:"ready_marker"`
}

func NewPlatform(name, readyMarker string) *Platform {
	return &Platform{
		name,
		readyMarker,
	}
}

package domain

import "time"

type ContentType string

const (
	ContentTypeVideo ContentType = "video"
	ContentTypeText  ContentType = "text"
)

type Metrics struct {
	Views       int64
	Likes       int64
	ReadingTime float64
	Reactions   int64
}

type Content struct {
	ExternalID  string
	Provider    string
	Title       string
	Type        ContentType
	Metrics     Metrics
	PublishedAt time.Time
	Tags        []string
	RawMetrics  []byte
	FinalScore  float64
}

func (c Content) UniqueKey() string {
	return c.Provider + ":" + c.ExternalID
}

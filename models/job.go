package models

type JobMessage struct {
	UUID      string `json:"uuid"`
	MediaPath string `json:"media"`
	AudioPath string `json:"audio"`
	Bucket    string `json:"bucket"`
}

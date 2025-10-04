package models

type JobMessage struct {
	UUID      string `json:"uuid"`
	ImagePath string `json:"image"`
	AudioPath string `json:"audio"`
	Bucket    string `json:"bucket"`
}

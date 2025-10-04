package services

import "fmt"

type ErrorType string

const (
	ErrTypeMinIO    ErrorType = "minio"
	ErrTypeRabbitMQ ErrorType = "rabbitmq"
	ErrTypeFFmpeg   ErrorType = "ffmpeg"
	ErrTypeImage    ErrorType = "image"
	ErrTypeSystem   ErrorType = "system"
)

type JobError struct {
	Type    ErrorType
	JobUUID string
	Op      string
	Err     error
}

func (e *JobError) Error() string {
	if e.JobUUID != "" {
		return fmt.Sprintf("[%s] job=%s op=%s: %v", e.Type, e.JobUUID, e.Op, e.Err)
	}
	return fmt.Sprintf("[%s] op=%s: %v", e.Type, e.Op, e.Err)
}

func (e *JobError) Unwrap() error {
	return e.Err
}

func newJobError(errType ErrorType, jobUUID, op string, err error) error {
	return &JobError{
		Type:    errType,
		JobUUID: jobUUID,
		Op:      op,
		Err:     err,
	}
}

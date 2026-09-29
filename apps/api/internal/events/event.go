package events

import "time"

type Event struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Timestamp     time.Time `json:"timestamp"`
	Source        string    `json:"source"`
	CorrelationID string    `json:"correlationId"`
	Payload       any       `json:"payload"`
}

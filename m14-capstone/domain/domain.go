// Package domain holds the capstone's core types shared across the slice.
package domain

import "time"

type Customer struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// CustomerEvent is published to JetStream when a customer is created.
type CustomerEvent struct {
	ID         string    `json:"id"` // event/customer id; doubles as dedup MsgID
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	OccurredAt time.Time `json:"occurred_at"`
}

const (
	StreamName             = "CXM_EVENTS"
	SubjectCustomerCreated = "cxm.customer.created"
	SubjectCustomerAll     = "cxm.customer.>"
	SubjectDLQ             = "cxm.customer.dlq"
)

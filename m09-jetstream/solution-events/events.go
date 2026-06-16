package events

import "time"

const (
	StreamName             = "CXM_EVENTS"
	SubjectCustomerCreated = "cxm.customer.created"
	SubjectCustomerAll     = "cxm.customer.>"
	// DLQ lives under cxm.customer.> so the same stream captures it; the main
	// consumer filters on .created, so it never re-consumes dead letters.
	SubjectDLQ = "cxm.customer.dlq"
)

type CustomerEvent struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Package events defines Pulse's event taxonomy and payload types.
package events

import "time"

// Subject naming: cxm.<entity>.<action>
const (
	SubjectCustomerCreated = "cxm.customer.created"
	SubjectCustomerUpdated = "cxm.customer.updated"
	SubjectCustomerAll     = "cxm.customer.*"
	SubjectProfileLookup   = "cxm.profile.lookup" // request-reply
)

// CustomerEvent is the payload published when a customer changes.
type CustomerEvent struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Package events is YOUR implementation target for M08 (NATS Core).
//
// Goal: define Pulse's event taxonomy and payload types so that the bus
// package and its tests compile and pass. The bus test (../bus/bus_test.go)
// references the symbols below — they define the exact API you must provide.
// Reference answer key: ../solution-events/ (and §2.2 / §3 details in
// ../M08-nats-core.md). Try it yourself before peeking.
//
// Build, in THIS file (events.go):
//
//   - Subject-name string constants following the cxm.<entity>.<action>
//     taxonomy from the lesson:
//     SubjectCustomerCreated = "cxm.customer.created"
//     SubjectCustomerUpdated = "cxm.customer.updated"
//     SubjectCustomerAll     = "cxm.customer.*"   // wildcard subscription
//     SubjectProfileLookup   = "cxm.profile.lookup" // request-reply subject
//
//   - `CustomerEvent` — the JSON payload published when a customer changes.
//     Fields (all with snake_case JSON tags so encoding is stable across
//     services):
//     ID         string    `json:"id"`
//     Email      string    `json:"email"`
//     Name       string    `json:"name"`
//     OccurredAt time.Time `json:"occurred_at"`
//
// The bus test uses SubjectCustomerCreated, SubjectProfileLookup, and builds
// CustomerEvent values with ID/Email/Name — so those must exist exactly.
//
// Delete this comment block as you implement. The package will not compile
// until the constants and CustomerEvent type the tests reference exist.
package events

// TODO(M08): implement SubjectCustomerCreated, SubjectCustomerUpdated,
// SubjectCustomerAll, SubjectProfileLookup, and the CustomerEvent struct.

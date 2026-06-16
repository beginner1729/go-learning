// Package profile is YOUR implementation target for M05 Exercises 3.1 & 3.2.
//
// Goal: make `go test ./profile/...` pass and `go run ./cmd/server` work. The
// tests in profile_test.go drive a real (in-memory, bufconn) gRPC server, so
// they pin down the exact API you must provide. Reference answer key:
// ../solution-profile/ (and §2.3–2.5 + §3 of ../M05-grpc.md). Try it yourself
// before peeking.
//
// The generated protobuf code in ../proto/profile/v1/ is fixed — import it as
//
//	profilev1 "cxm/m05/proto/profile/v1"
//
// and code to its generated interface (ProfileServiceServer). Use the nil-safe
// GetX() accessors on proto messages, never raw field access.
//
// Build, in THIS file (profile.go):
//
//   - ErrNotFound — a sentinel error (var, errors.New). The domain Client
//     re-domains gRPC codes.NotFound back into this for its callers.
//
//   - Profile struct: CustomerID string, Traits map[string]string,
//     UpdatedAt time.Time.
//
//   - Event struct: ID, CustomerID, Type string, OccurredAt time.Time.
//
//   - Repo interface (storage contract, swapped for Mongo in M7):
//     Get(ctx, customerID string) (Profile, error)
//     Upsert(ctx, p Profile) (Profile, error)
//     Events(ctx, customerID string) ([]Event, error)
//
//   - Server — adapts a Repo to the generated gRPC server interface. EMBED
//     profilev1.UnimplementedProfileServiceServer (forward compatibility) plus
//     a Repo field. Provide NewServer(repo Repo) *Server. Implement:
//     GetProfile(ctx, *GetProfileRequest) (*Profile, error)
//     — empty customer_id -> codes.InvalidArgument; repo ErrNotFound ->
//     codes.NotFound; other err -> codes.Internal; else map to proto.
//     UpsertProfile(ctx, *UpsertProfileRequest) (*Profile, error)
//     — empty customer_id -> codes.InvalidArgument; stamp UpdatedAt =
//     time.Now().UTC(); repo err -> codes.Internal; else return stored.
//     ListEvents(*ListEventsRequest, ProfileService_ListEventsServer) error
//     — SERVER STREAMING: empty customer_id -> codes.InvalidArgument; fetch
//     events; stream.Send each (respect stream.Context().Done()).
//     (Map domain <-> proto with small toProto/toProtoEvent helpers.)
//
//   - InMemoryRepo — concurrency-safe Repo backed by maps + sync.RWMutex.
//     Provide NewInMemoryRepo() *InMemoryRepo. Get returns ErrNotFound when
//     absent; Upsert stores by CustomerID. Also expose, for the tests:
//     AddEvent(id string, e Event)  — append an event (used to seed streaming)
//     Events returns a copy of the slice.
//
//   - Client (Exercise 3.2) — wraps the generated ProfileServiceClient with a
//     per-call deadline and domain error translation. Provide
//     NewClient(raw profilev1.ProfileServiceClient, timeout time.Duration)
//     *Client and GetProfile(ctx, customerID string) (Profile, error): apply
//     context.WithTimeout, call the raw stub, translate codes.NotFound into
//     ErrNotFound, wrap other errors with %w, else map proto -> Profile.
//
// The profile tests reference: NewInMemoryRepo, InMemoryRepo.AddEvent,
// NewServer, NewClient, Client.GetProfile, Event, ErrNotFound.
//
// Delete this comment block as you implement. The package will not compile
// until these types and functions exist.
package profile

// TODO(3.1): implement ErrNotFound, Profile, Event, Repo, Server (NewServer,
// GetProfile, UpsertProfile, ListEvents), InMemoryRepo (NewInMemoryRepo, Get,
// Upsert, AddEvent, Events).
// TODO(3.2): implement Client (NewClient, GetProfile) with deadline + error
// translation.

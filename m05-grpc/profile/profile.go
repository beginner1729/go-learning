// Package profile implements the ProfileService gRPC server over an in-memory
// repo (swapped for MongoDB in M7). It maps domain errors to gRPC status codes.
package profile

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	profilev1 "cxm/m05/proto/profile/v1"
)

var ErrNotFound = errors.New("profile: not found")

type Profile struct {
	CustomerID string
	Traits     map[string]string
	UpdatedAt  time.Time
}

type Event struct {
	ID         string
	CustomerID string
	Type       string
	OccurredAt time.Time
}

// Repo is the storage contract (in-memory now, Mongo later).
type Repo interface {
	Get(ctx context.Context, customerID string) (Profile, error)
	Upsert(ctx context.Context, p Profile) (Profile, error)
	Events(ctx context.Context, customerID string) ([]Event, error)
}

// Server adapts a Repo to the generated gRPC server interface.
type Server struct {
	profilev1.UnimplementedProfileServiceServer // forward-compatible embedding
	repo Repo
}

func NewServer(repo Repo) *Server { return &Server{repo: repo} }

func (s *Server) GetProfile(ctx context.Context, req *profilev1.GetProfileRequest) (*profilev1.Profile, error) {
	if req.GetCustomerId() == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}
	p, err := s.repo.Get(ctx, req.GetCustomerId())
	if errors.Is(err, ErrNotFound) {
		return nil, status.Errorf(codes.NotFound, "profile %q not found", req.GetCustomerId())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	return toProto(p), nil
}

func (s *Server) UpsertProfile(ctx context.Context, req *profilev1.UpsertProfileRequest) (*profilev1.Profile, error) {
	if req.GetCustomerId() == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}
	saved, err := s.repo.Upsert(ctx, Profile{
		CustomerID: req.GetCustomerId(),
		Traits:     req.GetTraits(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	return toProto(saved), nil
}

func (s *Server) ListEvents(req *profilev1.ListEventsRequest, stream profilev1.ProfileService_ListEventsServer) error {
	if req.GetCustomerId() == "" {
		return status.Error(codes.InvalidArgument, "customer_id is required")
	}
	events, err := s.repo.Events(stream.Context(), req.GetCustomerId())
	if err != nil {
		return status.Error(codes.Internal, "internal error")
	}
	for _, e := range events {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		if err := stream.Send(toProtoEvent(e)); err != nil {
			return err
		}
	}
	return nil
}

func toProto(p Profile) *profilev1.Profile {
	return &profilev1.Profile{
		CustomerId:    p.CustomerID,
		Traits:        p.Traits,
		UpdatedAtUnix: p.UpdatedAt.Unix(),
	}
}

func toProtoEvent(e Event) *profilev1.Event {
	return &profilev1.Event{
		Id:             e.ID,
		CustomerId:     e.CustomerID,
		Type:           e.Type,
		OccurredAtUnix: e.OccurredAt.Unix(),
	}
}

// InMemoryRepo is a concurrency-safe Repo for M5.
type InMemoryRepo struct {
	mu       sync.RWMutex
	profiles map[string]Profile
	events   map[string][]Event
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{profiles: map[string]Profile{}, events: map[string][]Event{}}
}

func (r *InMemoryRepo) Get(_ context.Context, id string) (Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.profiles[id]
	if !ok {
		return Profile{}, ErrNotFound
	}
	return p, nil
}

func (r *InMemoryRepo) Upsert(_ context.Context, p Profile) (Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[p.CustomerID] = p
	return p, nil
}

func (r *InMemoryRepo) AddEvent(id string, e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[id] = append(r.events[id], e)
}

func (r *InMemoryRepo) Events(_ context.Context, id string) ([]Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Event(nil), r.events[id]...), nil
}

// Client wraps the generated stub with deadlines + domain error translation.
type Client struct {
	raw     profilev1.ProfileServiceClient
	timeout time.Duration
}

func NewClient(raw profilev1.ProfileServiceClient, timeout time.Duration) *Client {
	return &Client{raw: raw, timeout: timeout}
}

func (c *Client) GetProfile(ctx context.Context, customerID string) (Profile, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	p, err := c.raw.GetProfile(ctx, &profilev1.GetProfileRequest{CustomerId: customerID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return Profile{}, ErrNotFound
		}
		return Profile{}, fmt.Errorf("get profile: %w", err)
	}
	return Profile{CustomerID: p.GetCustomerId(), Traits: p.GetTraits(),
		UpdatedAt: time.Unix(p.GetUpdatedAtUnix(), 0).UTC()}, nil
}

package profile_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"cxm/m05/interceptor"
	"cxm/m05/profile"
	profilev1 "cxm/m05/proto/profile/v1"
)

const token = "test-token"

// startBufconn spins up an in-memory gRPC server+client (no real sockets).
func startBufconn(t *testing.T) (profilev1.ProfileServiceClient, *profile.InMemoryRepo) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	repo := profile.NewInMemoryRepo()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(
		interceptor.RecoverUnary(log),
		interceptor.LogUnary(log),
		interceptor.AuthUnary(token),
	))
	profilev1.RegisterProfileServiceServer(srv, profile.NewServer(repo))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return profilev1.NewProfileServiceClient(conn), repo
}

func authCtx() context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
}

func TestUpsertAndGet(t *testing.T) {
	cli, _ := startBufconn(t)
	ctx := authCtx()

	_, err := cli.UpsertProfile(ctx, &profilev1.UpsertProfileRequest{
		CustomerId: "cus_1", Traits: map[string]string{"tier": "gold"},
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := cli.GetProfile(ctx, &profilev1.GetProfileRequest{CustomerId: "cus_1"})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.GetTraits()["tier"] != "gold" {
		t.Fatalf("traits = %v", got.GetTraits())
	}
}

func TestGet_NotFound(t *testing.T) {
	cli, _ := startBufconn(t)
	_, err := cli.GetProfile(authCtx(), &profilev1.GetProfileRequest{CustomerId: "ghost"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("want NotFound, got %v", err)
	}
}

func TestAuth_Rejected(t *testing.T) {
	cli, _ := startBufconn(t)
	// No metadata -> Unauthenticated.
	_, err := cli.GetProfile(context.Background(), &profilev1.GetProfileRequest{CustomerId: "cus_1"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestListEvents_Streaming(t *testing.T) {
	cli, repo := startBufconn(t)
	now := time.Now()
	for i := 0; i < 3; i++ {
		repo.AddEvent("cus_1", profile.Event{ID: string(rune('a' + i)), CustomerID: "cus_1", Type: "page_view", OccurredAt: now})
	}
	stream, err := cli.ListEvents(authCtx(), &profilev1.ListEventsRequest{CustomerId: "cus_1"})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("recv: %v", err)
		}
		count++
	}
	if count != 3 {
		t.Fatalf("streamed %d events, want 3", count)
	}
}

func TestDomainClient_NotFoundTranslation(t *testing.T) {
	raw, _ := startBufconn(t)
	dc := profile.NewClient(raw, 2*time.Second)
	_, err := dc.GetProfile(authCtx(), "ghost")
	if err != profile.ErrNotFound {
		t.Fatalf("want domain ErrNotFound, got %v", err)
	}
}

# Module 5 — gRPC for Internal Services

> **Capstone contribution:** the internal **`profile-svc`** gRPC API and the
> client `customer-api` uses to call it. Covers protobuf, codegen, unary +
> server-streaming RPCs, interceptors (auth, logging, recovery), and error mapping.

---

## 0. Setup & run

```bash
cd go-cxm-course/m05-grpc
go mod tidy
go vet ./...
go test ./...        # uses bufconn — in-memory gRPC, no ports needed
go run ./cmd/server  # listens on :9090
# regen protos only if you change the .proto (generated code is committed):
#   protoc --go_out=. --go_opt=paths=source_relative \
#          --go-grpc_out=. --go-grpc_opt=paths=source_relative \
#          proto/profile/v1/profile.proto
```

Layout:

```
m05-grpc/
  go.mod                       # module cxm/m05
  proto/profile/v1/
    profile.proto              # service + message definitions (the contract)
    profile.pb.go              # generated messages   (committed)
    profile_grpc.pb.go         # generated client+server stubs (committed)
  profile/                     # the service implementation (in-memory)
  interceptor/                 # auth + logging + recovery interceptors
  cmd/server/                  # runnable gRPC server
```

---

## 1. Learning objectives

By the end you will be able to:

- Define services and messages in **protobuf** and generate Go with `protoc`.
- Implement and call **unary** and **server-streaming** RPCs.
- Write **interceptors** (gRPC middleware) for auth, logging, and recovery.
- Map domain errors to gRPC **status codes** and back.
- Test gRPC in-memory with **`bufconn`** — fast, hermetic, no real sockets.
- Explain when to choose gRPC over REST.

---

## 2. Concepts

### 2.1 Why gRPC for internal calls

REST/JSON is great for *external* clients (browsers, partners, curl). For
*service-to-service* traffic, gRPC gives you:

- **A typed contract** (`.proto`) shared by both sides — no drift, no hand-written
  client. The compiler catches breaking changes.
- **Efficiency**: binary protobuf over HTTP/2, multiplexed streams.
- **Streaming**: client/server/bidi streaming as first-class.
- **Codegen** in every language — polyglot orgs love it.

Trade-off: not browser-native (needs grpc-web/proxy), binary payloads are harder
to eyeball. So Pulse uses **REST at the edge, gRPC between services** — the
common enterprise split.

### 2.2 The proto contract

```proto
syntax = "proto3";
package profile.v1;
option go_package = "cxm/m05/proto/profile/v1;profilev1";

service ProfileService {
  rpc GetProfile(GetProfileRequest) returns (Profile);
  rpc UpsertProfile(UpsertProfileRequest) returns (Profile);
  rpc ListEvents(ListEventsRequest) returns (stream Event); // server streaming
}

message Profile {
  string customer_id = 1;
  map<string, string> traits = 2;   // flexible profile attributes
  int64 updated_at_unix = 3;
}
```

Field numbers are the wire identity — **never reuse or renumber** them; that's how
protobuf stays backward/forward compatible. Add fields with new numbers; reserve
removed ones.

> **`buf` vs raw `protoc`:** in real projects use **`buf`** — it manages
> dependencies, lints protos, and detects breaking changes in CI. We invoke
> `protoc` directly here to show the moving parts; the capstone uses `buf`.

### 2.3 Codegen → server interface

`protoc` generates a `ProfileServiceServer` interface. You implement it and
register it. The `mustEmbedUnimplemented...` pattern means **adding a new RPC to
the proto won't break your build** — your server keeps compiling, returning
`Unimplemented` until you implement it.

```go
type Server struct {
    profilev1.UnimplementedProfileServiceServer // forward-compat embedding
    repo ProfileRepo
}

func (s *Server) GetProfile(ctx context.Context, req *profilev1.GetProfileRequest) (*profilev1.Profile, error) {
    p, err := s.repo.Get(ctx, req.GetCustomerId())
    if errors.Is(err, ErrNotFound) {
        return nil, status.Errorf(codes.NotFound, "profile %q not found", req.GetCustomerId())
    }
    if err != nil {
        return nil, status.Error(codes.Internal, "internal error") // don't leak details
    }
    return toProto(p), nil
}
```

### 2.4 Errors: `status` + `codes`

gRPC errors are a `(code, message)` pair via `google.golang.org/grpc/status`. Map
your domain errors to codes at the boundary:

| Domain | gRPC code |
|---|---|
| not found | `codes.NotFound` |
| validation | `codes.InvalidArgument` |
| conflict | `codes.AlreadyExists` |
| unauthenticated | `codes.Unauthenticated` |
| permission | `codes.PermissionDenied` |
| anything else | `codes.Internal` (generic message — no internal leakage) |

The client inspects with `status.FromError(err)`.

### 2.5 Server streaming

`ListEvents` returns a `stream Event`. The server sends many; the client ranges
until EOF:

```go
func (s *Server) ListEvents(req *profilev1.ListEventsRequest, stream profilev1.ProfileService_ListEventsServer) error {
    events, _ := s.repo.Events(stream.Context(), req.GetCustomerId())
    for _, e := range events {
        if err := stream.Send(toProtoEvent(e)); err != nil {
            return err // client went away / context cancelled
        }
    }
    return nil // closing the stream
}
```

Always check `stream.Context()` for cancellation in long streams.

### 2.6 Interceptors = gRPC middleware

Two kinds: **unary** and **stream** interceptors. Same decorator idea as HTTP
middleware. Auth example:

```go
func AuthUnary(token string) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler) (any, error) {
        md, _ := metadata.FromIncomingContext(ctx)
        if vals := md.Get("authorization"); len(vals) == 0 || vals[0] != "Bearer "+token {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        return handler(ctx, req) // proceed
    }
}
```

Chain them with `grpc.ChainUnaryInterceptor(recovery, logging, auth)`.

#### Pitfalls

- **Forgetting `paths=source_relative`** → generated files land in the wrong dir.
- **Renumbering proto fields** → silent wire corruption across versions.
- **Returning raw domain errors** from RPCs → clients get `codes.Unknown` and leak
  internals. Always wrap in `status`.
- **Not setting deadlines on the client** → calls can hang forever. The client
  must pass a `context` with a timeout.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — Implement `UpsertProfile`

**Task:** Implement the unary `UpsertProfile` RPC: validate that `customer_id` is
non-empty (`InvalidArgument` otherwise), upsert traits, return the stored profile.

<details>
<summary>Reference solution</summary>

```go
func (s *Server) UpsertProfile(ctx context.Context, req *profilev1.UpsertProfileRequest) (*profilev1.Profile, error) {
    if req.GetCustomerId() == "" {
        return nil, status.Error(codes.InvalidArgument, "customer_id is required")
    }
    p := Profile{
        CustomerID: req.GetCustomerId(),
        Traits:     req.GetTraits(),
        UpdatedAt:  time.Now().UTC(),
    }
    saved, err := s.repo.Upsert(ctx, p)
    if err != nil {
        return nil, status.Error(codes.Internal, "internal error")
    }
    return toProto(saved), nil
}
```

**Reasoning:** Validation maps to `InvalidArgument` (the gRPC equivalent of 400).
Internal failures get a generic `Internal` message. `GetX()` accessors are
nil-safe — always prefer them over field access on proto messages.

</details>

### Exercise 3.2 — A client wrapper with a deadline

**Task:** Write a `Client` wrapping the generated stub that always applies a
per-call timeout and translates `codes.NotFound` back into a domain `ErrNotFound`.

<details>
<summary>Reference solution</summary>

```go
type Client struct {
    raw     profilev1.ProfileServiceClient
    timeout time.Duration
}

func (c *Client) GetProfile(ctx context.Context, customerID string) (Profile, error) {
    ctx, cancel := context.WithTimeout(ctx, c.timeout)
    defer cancel()
    p, err := c.raw.GetProfile(ctx, &profilev1.GetProfileRequest{CustomerId: customerID})
    if err != nil {
        if status.Code(err) == codes.NotFound {
            return Profile{}, ErrNotFound // re-domain the error for callers
        }
        return Profile{}, fmt.Errorf("get profile: %w", err)
    }
    return fromProto(p), nil
}
```

**Reasoning:** Callers of `Client` deal in domain types and `ErrNotFound`, not gRPC
codes — the gRPC-ness stops at this boundary. The deadline is enforced here so no
caller can forget it.

</details>

---

## 4. Fill-in-the-blank

Complete the server bootstrap with chained interceptors.

```go
func NewGRPCServer(token string, log *slog.Logger, impl profilev1.ProfileServiceServer) *grpc.Server {
    s := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            interceptor.RecoverUnary(log),
            interceptor.LogUnary(log),
            interceptor./* ___1: auth ___ */(token),
        ),
    )
    profilev1./* ___2: register the impl ___ */(s, impl)
    return s
}
```

<details>
<summary>Answers</summary>

```go
grpc.ChainUnaryInterceptor(
    interceptor.RecoverUnary(log),
    interceptor.LogUnary(log),
    interceptor.AuthUnary(token), // 1
),
...
profilev1.RegisterProfileServiceServer(s, impl) // 2
```

Order: recovery outermost (catches panics from everything), then logging, then
auth — same reasoning as HTTP middleware in M4.

</details>

---

## 5. Implement it yourself

**Problem:** Add a **client-streaming** RPC `RecordEvents(stream Event) returns
(RecordSummary)` to `profile-svc`: the client streams many events, the server
counts/persists them and returns a summary. Then:
- Add a **stream interceptor** version of your auth check.
- Wire `customer-api` (M4) to call `profile-svc` over gRPC when a customer is
  created (fetch/seed their profile) — this is the first cross-service call in Pulse.

**Curated resources:**
- gRPC Go quickstart — https://grpc.io/docs/languages/go/quickstart/
- gRPC Go basics tutorial (streaming) — https://grpc.io/docs/languages/go/basics/
- `buf` — https://buf.build/docs/  ·  Protobuf style guide — https://protobuf.dev/programming-guides/style/
- `status`/`codes` — https://pkg.go.dev/google.golang.org/grpc/status , https://pkg.go.dev/google.golang.org/grpc/codes
- `bufconn` testing — https://pkg.go.dev/google.golang.org/grpc/test/bufconn
- Error model — https://grpc.io/docs/guides/error/

**Hints:** client streaming server signature is `Record(stream X_RecordServer) error`;
loop `stream.Recv()` until `io.EOF`, then `stream.SendAndClose(&Summary{...})`.

---

## 6. Capstone contribution

Pulse now has `profile-svc` exposing gRPC (`GetProfile`, `UpsertProfile`,
`ListEvents`) with interceptors and a domain-friendly client. In M7 the in-memory
profile repo becomes MongoDB-backed; the gRPC surface stays identical — the
benefit of coding to the generated interface.

---

## 7. Self-check — you should now be able to…

- [ ] Write a `.proto` and generate Go server/client stubs.
- [ ] Implement unary and server-streaming RPCs with proper `status`/`codes`.
- [ ] Write and chain unary interceptors (auth/log/recover).
- [ ] Test gRPC with `bufconn` (no real network).
- [ ] Explain the REST-at-edge / gRPC-between-services split.

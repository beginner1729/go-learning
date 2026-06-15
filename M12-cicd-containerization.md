# Module 12 — CI/CD & Containerization

> **Capstone contribution:** Pulse ships — **multi-stage Docker images** (tiny,
> secure), a **GitHub Actions** pipeline (lint → test → build → image → release),
> a **docker-compose** for local full-stack runs, and **Kubernetes** manifests for
> deployment.

---

## 0. Setup & run

```bash
cd go-cxm-course/m12-cicd
go build ./...
docker build -t pulse/customer-api:dev -f Dockerfile .   # multi-stage build
docker run --rm -p 8080:8080 pulse/customer-api:dev &
curl -s localhost:8080/healthz
```

Layout:

```
m12-cicd/
  go.mod  main.go            # minimal service to containerize
  Dockerfile                 # multi-stage, distroless, non-root
  .dockerignore
  docker-compose.yaml        # app + postgres + mongo + nats
  .github/workflows/ci.yml   # the pipeline
  deploy/k8s/                # deployment + service + configmap
```

---

## 1. Learning objectives

By the end you will be able to:

- Write a **multi-stage Dockerfile** producing a tiny, non-root, static image.
- Build a **CI pipeline** (GitHub Actions): lint, test (with race + coverage),
  build, and publish an image.
- Use **docker-compose** to run the whole stack locally.
- Apply **versioning/release** conventions (semver tags, build metadata).
- Deploy to **Kubernetes** with Deployment/Service/ConfigMap and health probes.

---

## 2. Concepts

### 2.1 Multi-stage Docker builds

Compile in a full Go image, then copy *only the binary* into a minimal runtime
image. Result: ~10–20MB images with no compiler, shell, or OS cruft to attack.

```dockerfile
# ---- build stage ----
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download                      # cached layer: deps change rarely
COPY . .
# CGO off => static binary; trim symbols; embed version via ldflags
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/app ./

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot                      # never run as root
EXPOSE 8080
ENTRYPOINT ["/app"]
```

Key choices:
- **`CGO_ENABLED=0`** → fully static binary; runs on `scratch`/`distroless`.
- **`distroless` / `scratch`** → no shell, tiny attack surface. `distroless:nonroot`
  gives a non-root user and CA certs.
- **Layer caching:** copy `go.mod`/`go.sum` and `go mod download` *before* the
  source so dependency layers cache across code changes.
- **`-ldflags "-s -w"`** strips debug info; **`-X`** injects the version string.

### 2.2 `.dockerignore`

Keep the build context small and secrets out:

```
.git
*.md
**/*_test.go
deploy
.github
```

### 2.3 The CI pipeline (GitHub Actions)

Stages, each a required gate:

```yaml
name: ci
on:
  push: { branches: [main] }
  pull_request:
jobs:
  quality:
    runs-on: ubuntu-latest
    services:                      # real deps for integration tests
      postgres:
        image: postgres:16-alpine
        env: { POSTGRES_PASSWORD: pulse, POSTGRES_USER: pulse, POSTGRES_DB: pulse }
        ports: ["5432:5432"]
        options: >-
          --health-cmd pg_isready --health-interval 10s --health-timeout 5s --health-retries 5
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24', cache: true }
      - name: Lint
        uses: golangci/golangci-lint-action@v6
      - name: Test
        run: go test -race -coverprofile=cover.out ./...
        env:
          PULSE_PG_DSN: postgres://pulse:pulse@localhost:5432/pulse?sslmode=disable
      - name: Coverage gate
        run: |
          pct=$(go tool cover -func=cover.out | tail -1 | awk '{print $3}' | tr -d '%')
          echo "coverage: $pct%"
          awk "BEGIN{exit !($pct >= 70)}"   # fail under 70%
```

The release/image job builds and pushes on tags:

```yaml
  image:
    needs: quality
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    permissions: { contents: read, packages: write }
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with: { registry: ghcr.io, username: ${{ github.actor }}, password: ${{ secrets.GITHUB_TOKEN }} }
      - uses: docker/build-push-action@v6
        with:
          push: true
          tags: ghcr.io/acme/pulse-customer-api:${{ github.ref_name }}
          build-args: VERSION=${{ github.ref_name }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

> **Why GitHub Actions?** Ubiquitous, free for public repos, first-class Go +
> Docker support, service containers for integration tests. The *concepts*
> (lint→test→build→release gates, ephemeral service deps) transfer to GitLab CI,
> CircleCI, etc.

### 2.4 Versioning & releases

- **Semantic versioning** tags: `v1.4.2`. Tag → pipeline builds + pushes the image.
- Inject the version + commit into the binary via `-ldflags -X` and expose it on a
  `/version` endpoint and in logs — invaluable for debugging "what's deployed?".
- Keep a `CHANGELOG.md`; consider `goreleaser` for multi-platform binaries + GitHub
  Releases automation.

### 2.5 docker-compose for local dev

One command brings up the app + Postgres + Mongo + NATS so you can run the whole
CXM flow locally (used heavily in the M14 capstone). Compose is for **local/dev**;
Kubernetes is for real deployment.

### 2.6 Kubernetes basics

The minimum to run a stateless Go service:
- **Deployment** — replicas, the image, env, resource requests/limits, and
  **liveness/readiness probes** (hit `/healthz`).
- **Service** — stable in-cluster DNS + load balancing across pods.
- **ConfigMap/Secret** — config and credentials injected as env/files.

```yaml
livenessProbe:  { httpGet: { path: /healthz, port: 8080 }, initialDelaySeconds: 3 }
readinessProbe: { httpGet: { path: /healthz, port: 8080 }, periodSeconds: 5 }
resources:
  requests: { cpu: "50m", memory: "64Mi" }
  limits:   { cpu: "500m", memory: "256Mi" }
```

Readiness gates traffic until the pod is ready; liveness restarts a wedged pod.
Pair with the **graceful shutdown** from M4/M10 so in-flight requests drain on
rolling deploys (k8s sends SIGTERM, waits `terminationGracePeriodSeconds`).

#### Pitfalls

- **Running as root in the container** → security risk; use `nonroot`.
- **Fat images (full `golang` as runtime)** → slow pulls, big attack surface.
- **No health probes** → k8s can't tell if your app is alive/ready.
- **Ignoring SIGTERM** → dropped requests on every deploy. Wire graceful shutdown.
- **Baking secrets into images/env in plain YAML** → use Secrets / a secrets
  manager (M13).
- **No layer caching in CI** → slow builds; cache `go mod download` and use
  buildx GHA cache.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — Multi-stage Dockerfile with version injection

**Task:** Containerize the minimal service so the image is <25MB, runs as non-root,
and `/version` reports the `VERSION` build-arg.

<details>
<summary>Reference solution</summary>

See `m12-cicd/Dockerfile` (built & verified in this module). The `main.go` exposes
`/version` reading a `var version = "dev"` overridden via
`-ldflags "-X main.version=$VERSION"`. Build:

```bash
docker build --build-arg VERSION=v1.2.3 -t pulse/customer-api:v1.2.3 .
docker run --rm -p 8080:8080 pulse/customer-api:v1.2.3
curl localhost:8080/version   # -> {"version":"v1.2.3"}
```

**Reasoning:** the `-X` linker flag rewrites a package variable at build time — no
codegen, no config file. Distroless + `CGO_ENABLED=0` yields a tiny static image.

</details>

### Exercise 3.2 — A coverage gate

**Task:** Add a CI step that fails the build if total coverage drops below 70%.

<details>
<summary>Reference solution</summary>

```bash
go test -race -coverprofile=cover.out ./...
pct=$(go tool cover -func=cover.out | tail -1 | awk '{print $3}' | tr -d '%')
awk "BEGIN{exit !($pct >= 70)}"   # nonzero exit (fail) if below threshold
```

**Reasoning:** `go tool cover -func` prints a `total:` line; we parse the percent
and let `awk` set the exit code. A gate that's a *floor* (not a target) prevents
silent erosion without encouraging test-padding.

</details>

---

## 4. Fill-in-the-blank

Complete the Dockerfile's caching + static-build essentials.

```dockerfile
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod /* ___1: pre-fetch deps for caching ___ */
COPY . .
RUN /* ___2: disable cgo for a static binary ___ */ GOOS=linux go build -o /out/app ./

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER /* ___3 ___ */
ENTRYPOINT ["/app"]
```

<details>
<summary>Answers</summary>

```dockerfile
RUN go mod download                                  # 1
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./ # 2
USER nonroot:nonroot                                 # 3
```

</details>

---

## 5. Implement it yourself

**Problem:** Productionize the Pulse repo from M10:
- A Dockerfile **per service** (`customer-api`, `profile-svc`, `notification-svc`)
  — or one parametrized by build target.
- A `docker-compose.yaml` bringing up all three services + Postgres + Mongo + NATS,
  with healthchecks and depends_on ordering.
- A GitHub Actions pipeline: lint → unit tests → integration tests (service
  containers) → build images → push on tags, with caching.
- K8s manifests for one service incl. probes, resource limits, a ConfigMap, and a
  Secret; verify graceful shutdown drains on `kubectl rollout restart`.

**Curated resources:**
- Docker multi-stage builds — https://docs.docker.com/build/building/multi-stage/
- Distroless images — https://github.com/GoogleContainerTools/distroless
- GitHub Actions for Go — https://docs.github.com/en/actions/automating-builds-and-tests/building-and-testing-go
- `docker/build-push-action` — https://github.com/docker/build-push-action
- Compose file reference — https://docs.docker.com/compose/compose-file/
- Kubernetes Deployment — https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
- Configure probes — https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
- `goreleaser` — https://goreleaser.com/

**Hints:** parametrize one Dockerfile with `ARG SERVICE` and `go build ./cmd/$SERVICE`.
In compose, gate the app on DB healthchecks via `depends_on: { condition: service_healthy }`.

---

## 6. Capstone contribution

Pulse is now buildable, testable, and deployable through automation: tiny secure
images, a gated CI pipeline, a one-command local stack (compose), and k8s manifests
with health probes wired to the graceful-shutdown server. M13 adds the
observability needed to operate it.

---

## 7. Self-check — you should now be able to…

- [ ] Write a multi-stage, non-root, distroless Dockerfile with version injection.
- [ ] Build a CI pipeline gating on lint, race tests, and a coverage floor.
- [ ] Run the full stack locally with docker-compose.
- [ ] Apply semver tagging and inject build metadata via ldflags.
- [ ] Deploy a Go service to k8s with probes, limits, config/secrets, graceful drain.

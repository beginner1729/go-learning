# ---- build stage ----
FROM golang:1.24 AS build
WORKDIR /src

# Cache dependencies separately from source for fast rebuilds.
COPY go.mod ./
RUN go mod download

COPY . .
ARG VERSION=dev
# CGO off -> static binary; -s -w strip debug; -X injects the version.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/app ./

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]

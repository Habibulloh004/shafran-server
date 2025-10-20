FROM golang:1.24 AS builder

WORKDIR /src

# Leverage Go modules cache
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/shafran cmd/server/main.go

FROM gcr.io/distroless/base-debian12 AS runtime

WORKDIR /app

COPY --from=builder /out/shafran /usr/local/bin/shafran

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/shafran"]

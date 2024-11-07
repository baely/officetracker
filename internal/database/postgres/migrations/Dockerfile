FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

RUN go mod download

COPY . .

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /migrate -tags=integrated ./internal/database/postgres/migrations

FROM alpine

WORKDIR /app

COPY --from=builder /migrate /migrate

ENTRYPOINT ["/migrate"]

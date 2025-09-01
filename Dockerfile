FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

RUN go mod download

COPY . .

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /tracker -tags=integrated

FROM alpine

WORKDIR /app

COPY --from=builder /tracker /tracker
COPY ./config ./config

RUN apk --no-cache add tzdata

ENTRYPOINT ["/tracker"]

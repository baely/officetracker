FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

COPY ./index.html ./index.html
COPY ./main.go ./main.go
COPY ./db.go ./db.go
COPY ./notify.go ./notify.go

RUN go build -o /tracker .

FROM alpine

WORKDIR /app

COPY --from=builder /tracker /tracker
COPY ./index.html ./index.html

ENTRYPOINT ["/tracker"]

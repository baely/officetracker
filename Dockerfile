FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

COPY . .

RUN go build -o /tracker .

FROM alpine

WORKDIR /app

COPY --from=builder /tracker /tracker
COPY ./app/index.html ./index.html

RUN apk --no-cache add tzdata

ENTRYPOINT ["/tracker"]

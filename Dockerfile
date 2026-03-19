FROM golang:1.25.6-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE_NAME=staff-simulator
ARG VERSION=latest

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /out/staff-simulator \
    ./internal

FROM alpine:3.22

RUN adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/staff-simulator /app/staff-simulator

ARG SERVICE_NAME=staff-simulator
ARG VERSION=latest

ENV SERVICE_NAME=${SERVICE_NAME}
ENV VERSION=${VERSION}

USER appuser

ENTRYPOINT ["/app/staff-simulator"]

FROM golang:1.18-alpine AS builder

WORKDIR /build
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
COPY . .
RUN rm -rf dist && mkdir dist && go build -o dist ./...

FROM alpine/curl

COPY --from=builder /build/dist .
COPY --from=builder /build/migrations /migrations
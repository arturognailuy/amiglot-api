# syntax=docker/dockerfile:1

FROM golang:1.24.0 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags "-s -w" -o /out/amiglot-api ./cmd/server

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/amiglot-api /app/amiglot-api

ENV PORT=6174
EXPOSE 6174

USER nonroot:nonroot
ENTRYPOINT ["/app/amiglot-api"]

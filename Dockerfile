# syntax=docker/dockerfile:1

FROM golang:1.24.0 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
ARG GIT_SHA=dev
ARG GIT_BRANCH=dev
ARG BUILD_TIME_UTC=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -buildvcs=false -ldflags "-s -w -X github.com/gnailuy/amiglot-api/internal/buildinfo.GitSHA=${GIT_SHA} -X github.com/gnailuy/amiglot-api/internal/buildinfo.GitBranch=${GIT_BRANCH} -X github.com/gnailuy/amiglot-api/internal/buildinfo.BuildTimeUTC=${BUILD_TIME_UTC}" -o /out/amiglot-api ./cmd/server

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/amiglot-api /app/amiglot-api

ENV PORT=6176
EXPOSE 6176

USER nonroot:nonroot
ENTRYPOINT ["/app/amiglot-api"]

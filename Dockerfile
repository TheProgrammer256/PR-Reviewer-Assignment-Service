# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
COPY gen ./gen
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pr-reviewer ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /src/pr-reviewer /app/pr-reviewer
ENV HTTP_PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/pr-reviewer"]


# Build stage
FROM golang:1.25 AS builder
WORKDIR /src

# Prime the module cache (creates a minimal go.sum)
COPY go.mod ./
RUN go mod download

# Copy source and resolve the full graph (writes complete go.sum)
COPY . .
RUN go mod tidy

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/server ./cmd/server

# Runtime
FROM gcr.io/distroless/base-debian12
ENV GIN_MODE=release
COPY --from=builder /bin/server /server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]

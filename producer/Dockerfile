# producer/Dockerfile
# Build stage
FROM golang:1.25 AS builder
WORKDIR /src

# Copy module files first (cache-friendly)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Sanity check so we fail with a friendly message if layout is wrong
RUN test -f ./cmd/producer/main.go || (echo "ERROR: ./cmd/producer/main.go not found. Did you place files under producer/cmd/producer/ ?" && exit 1)

# Build the producer binary from the cmd/producer package
ENV CGO_ENABLED=0
RUN GOOS=linux GOARCH=amd64 go build -o /bin/producer ./cmd/producer

# Runtime stage
FROM gcr.io/distroless/base-debian12
COPY --from=builder /bin/producer /producer
USER nonroot:nonroot
ENTRYPOINT ["/producer"]

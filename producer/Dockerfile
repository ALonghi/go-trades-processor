# producer/Dockerfile (top-level build)
FROM golang:1.25.0 AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/producer .

FROM gcr.io/distroless/base-debian12
COPY --from=builder /bin/producer /producer
USER nonroot:nonroot
ENTRYPOINT ["/producer"]

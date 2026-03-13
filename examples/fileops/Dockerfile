FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy the flicker module (build context is flicker root).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the example.
WORKDIR /build/examples/fileops
RUN go mod download
RUN CGO_ENABLED=0 go build -o /fileops .

# ---

FROM alpine:latest

RUN apk add --no-cache ca-certificates file coreutils

COPY --from=builder /fileops /app/fileops

WORKDIR /app
RUN mkdir -p /data/work /data/archive

EXPOSE 9001

CMD ["./fileops"]

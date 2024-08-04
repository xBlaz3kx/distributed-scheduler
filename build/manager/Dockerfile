# Use an official GoLang runtime as the base image
FROM golang:1.20-alpine AS builder

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-build
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux

# Set the working directory inside the container
WORKDIR /app

# Copy the Go mod and sum files to the working directory
COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/root/.cache/go-build go mod download

# Verify dependencies
RUN go mod verify

# Copy the source code from the current directory to the working directory inside the container
COPY . .

# install go swaggo, used to generate swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12
# build docs
RUN swag init -g handlers/doc.go

# Build the Go application
RUN --mount=type=cache,target="/root/.cache/go-build" go build -o bin/manager cmd/manager/main.go

# Build the migration tool
RUN --mount=type=cache,target="/root/.cache/go-build" go build -o bin/tooling cmd/tooling/main.go

# Use an official Alpine Linux runtime as a base image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage to the current stage
COPY --from=builder /app/bin/manager /app/manager
COPY --from=builder /app/bin/tooling /app/tooling

HEALTHCHECK --interval=5s --timeout=3s --retries=3  CMD curl -f http://localhost:8000/health || exit 1

# set command to run when starting the container
CMD ["/app/manager"]
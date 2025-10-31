# Use the official Golang image as the base image
FROM golang:1.24.9-alpine3.22 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download dependencies using go mod
ENV GOPROXY=https://proxy.golang.org,direct
RUN go mod download

# Copy the source code into the container
COPY . .
COPY .env .env

# Build the Go application
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build" go build -o scanner .

# Use a lightweight Alpine image to run the application
FROM alpine:latest

# Install dependencies: libc6-compat, make, curl, and timezone data
RUN apk add --no-cache libc6-compat make curl tzdata

# Set the working directory inside the container
WORKDIR /app

# Copy the Go binary and Node.js files from the respective builder stages
COPY --from=builder /app/scanner .
COPY --from=builder /app/.env .env

RUN mkdir -p files \
    && mkdir -p vectors \
    && chmod -R 777 files/ \
    && chmod -R 777 vectors/ 

# Command to run the Go application
CMD ["./scanner"]

# Use the official Golang image to build the application
FROM golang:1.23.3 AS build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum are not changed
RUN go mod tidy

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o main .

# Start a new stage from a smaller image
FROM ubuntu:22.04

# Install CA certificates, need this to make good https calls.
RUN apt-get update && apt-get install -y ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy .env file to /root
COPY .env /root/.env

# Copy the Pre-built binary file from the previous stage
COPY --from=build /app/main .

# Expose the port the app runs on
EXPOSE 3000 

# Command to run the executable
CMD ["./main"]

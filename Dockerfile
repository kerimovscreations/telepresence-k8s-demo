# Use the official Go image as a builder stage
FROM golang:1.24-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod file
COPY go.mod ./
COPY go.sum ./

# Copy the source code
COPY main.go ./

# Build the application with static linking
RUN go build -o app .

# Use a minimal alpine image for the final stage
FROM alpine:3.21

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/app .

# Expose port 8080
EXPOSE 8080

# Expose port 8081
EXPOSE 8081

# Expose port 8082
EXPOSE 8082

# Command to run the application
CMD ["./app"]
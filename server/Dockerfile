# Use the official Go image as base
FROM golang:1.21-alpine

# Set the working directory inside the container
WORKDIR /app

# Install git (needed for go mod download with private repos)
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o main .

# Expose port 8080 (adjust as needed)
EXPOSE 8080

# Command to run the application
CMD ["./main"]

FROM golang:1.20-alpine

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server main.go

# Make sure the binary is executable
RUN chmod +x /app/server

# Set environment variables
ENV REDIS_HOST="etcd-redis"
ENV REDIS_PORT="6379"
ENV SERVER_PORT="8080"

# Expose the server port
EXPOSE 8080

# Run the server
ENTRYPOINT ["go", "run", ".", "-mode", "server"]
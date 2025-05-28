FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o node-agent ./cmd/node-agent

FROM docker:dind

# Install necessary packages
RUN apk add --no-cache curl procps

WORKDIR /app
COPY --from=builder /app/node-agent .

# Add logging to container startup
ENV NODE_NAME=""
ENV NODE_IP=""
ENV API_HOST=""
ENV API_PORT=""

# Start both dockerd and node-agent
ENTRYPOINT ["sh", "-c", "dockerd & sleep 3 && echo '🐳 Docker daemon started' && echo '🚀 Starting node agent...' && ./node-agent"]
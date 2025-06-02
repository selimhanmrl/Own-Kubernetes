#!/bin/sh

# Start Docker daemon in background
dockerd &

# Wait for Docker daemon to be ready
echo "🐳 Waiting for Docker daemon to start..."
while ! docker info > /dev/null 2>&1; do
    sleep 1
done
echo "✅ Docker daemon is ready!"

# Run your node-agent binary
echo "🚀 Starting node agent..."
exec ./node-agent
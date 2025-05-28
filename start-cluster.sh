# Build the node image
docker build -t own-k8s-node -f docker/node.Dockerfile .

# Create a Docker network for the cluster
docker network create own-k8s-net

# Start Redis if not already running
docker run -d --name redis --network own-k8s-net -p 6379:6379 redis

# Start the API server
docker run -d --name own-k8s-master \
    --network own-k8s-net \
    -e REDIS_HOST=redis \
    -p 8080:8080 \
    own-k8s-node -mode server

# Start multiple worker nodes
for i in {1..3}
do
    docker run -d \
        --name own-k8s-node$i \
        --network own-k8s-net \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -e NODE_NAME=worker$i \
        -e NODE_IP=node$i \
        -e API_HOST=own-k8s-master \
        -e API_PORT=8080 \
        own-k8s-node -mode node
done

package main

import (
    "github.com/selimhanmrl/Own-Kubernetes/redis"
    "github.com/selimhanmrl/Own-Kubernetes/server"
)

func main() {
    // Initialize Redis
    own_redis.InitRedis()

    // Create and start API server
    apiServer := server.NewAPIServer()
    apiServer.Start()
}
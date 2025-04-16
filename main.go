package main

import (
	"github.com/selimhanmrl/Own-Kubernetes/cmd"
	"github.com/selimhanmrl/Own-Kubernetes/redis" // Alias the package as 'store'
)

func main() {
	own_redis.InitRedis() // Use the alias 'store' here
	cmd.Execute()
}

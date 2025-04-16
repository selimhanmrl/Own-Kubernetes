package main

import (
	"github.com/selimhanmrl/Own-Kubernetes/cmd"

	"github.com/selimhanmrl/Own-Kubernetes/store"
)

func main() {
	store.InitRedis() // Initialize Redis client
	cmd.Execute()

}

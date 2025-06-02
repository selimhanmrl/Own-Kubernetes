# Handmade Kubernetes

Build Own Kubernetes From Scratch with Go. Here is commands which you will use:

For applying pod
         
    go run main.go apply -f pod.yaml -n <Optional>
  
For applying service
  
    go run main.go apply-service -f service.yaml -n <Optional>
  
For Showing pods
   
    go run . get pods -n <Optional>

For Delete Pods
  
    go run . delete pod <Pod-Name> -n <Optional>

Calling Scheduler
    
    go run . scheduler

For Api-Server (Run in Docker)

    docker build -t api-server -f .\docker\server.Dockerfile . 
    docker run -d --name api-server -p 8080:8080 api-server

For ETCD

    docker run -d --name etcd-redis -p 6379:6379 redis

For Each Nodes (Kubelet)
    
    go run . node-server <Node-Name> --api-host <Api-Server IP> --api-port <Api-Server Port> --node-ip <Node Port>
    
Kube-Proxy (LoadBalancer for NodePort)
    
    go run main.go proxy

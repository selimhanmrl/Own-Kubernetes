package own_redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	RedisClient *redis.Client
	Ctx         = context.Background()
)

func InitRedis() {
	// Get Redis configuration from environment variables
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")

	// Use default values if environment variables are not set
	if redisHost == "" {
		redisHost = "localhost"
	}
	if redisPort == "" {
		redisPort = "6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "", // No password by default
		DB:       0,  // Default DB
	})

	// Try to connect with retries
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		_, err := RedisClient.Ping(Ctx).Result()
		if err == nil {
			log.Println("✅ Connected to Redis")
			return
		}
		log.Printf("⚠️ Attempt %d: Failed to connect to Redis: %v", i+1, err)
		if i < maxRetries-1 {
			log.Printf("Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}
	log.Fatalf("❌ Failed to connect to Redis after %d attempts", maxRetries)
}

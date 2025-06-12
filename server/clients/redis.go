package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/vitwit/resolute/server/config"
)

// RedisClient is the Redis client instance
var RedisClient *redis.Client
var ctx = context.Background()

// InitializeRedis initializes the Redis client
func InitializeRedis(addr string, password string, db int) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
}

// SetValue sets a value in Redis
func SetValue(key string, value string) error {
	err := RedisClient.Set(ctx, key, value, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

// GetValue gets a value from Redis
func GetValue(key string) (string, error) {
	val, err := RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return val, nil
}

func GetChain(chainId string) *config.ChainConfig {
	log.Printf("Looking up chain configuration for chainId: %s", chainId)
	
	data, err := GetValue("chains")
	if err != nil {
		log.Printf("Error fetching chains from Redis for chainId %s: %v", chainId, err)
		return nil
	}

	if data == "" {
		log.Printf("No chains found in Redis for chainId %s", chainId)
		return nil
	}

	var chains []config.ChainConfig

	e := json.Unmarshal([]byte(data), &chains)
	if e != nil {
		log.Printf("Error while unmarshaling chain info for chainId %s: %v", chainId, e)
		return nil
	}

	log.Printf("Found %d chains in Redis, searching for chainId: %s", len(chains), chainId)

	for _, c := range chains {
		if c.ChainId == chainId {
			log.Printf("Found chain configuration for %s: RestURI=%s, SourceEnd=%s", chainId, c.RestURI, c.SourceEnd)
			return &c
		}
	}

	// Log available chains for debugging
	availableChains := make([]string, len(chains))
	for i, c := range chains {
		availableChains[i] = c.ChainId
	}
	log.Printf("Chain %s not found. Available chains: %v", chainId, availableChains)

	return nil
}

func GetChains() []*config.ChainConfig {

	data, err := GetValue("chains")
	if err != nil {
		fmt.Println("Error fetching chains from Redis:", err)
		return nil
	}

	if data == "" {
		fmt.Println("No chains found in Redis")
		return nil
	}

	var chains []config.ChainConfig
	e := json.Unmarshal([]byte(data), &chains)
	if e != nil {
		fmt.Println("Error while unmarshal chain info ", e.Error())
		return nil
	}

	chainPointers := make([]*config.ChainConfig, len(chains))
	for i := range chains {
		chainPointers[i] = &chains[i]
	}

	return chainPointers
}

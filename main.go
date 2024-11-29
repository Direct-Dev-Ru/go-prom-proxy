package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// curl http://localhost:8080/cpu_usage/30/ollama-ollama_3-1

// Define the secret file path
const secretFilePath = "api_key.secret"

func getEnvironment(name, defaultValue string) string {
	if envVal := os.Getenv(name); envVal == "" {
		return defaultValue
	} else {
		return envVal
	}
}

func main() {
	// Prometheus server URL
	promURL := getEnvironment("PROM_PROXY_SERVER_URL", "http://192.168.87.108:9090")

	// Initialize Gin router
	router := gin.Default()

	// Apply API key middleware to the /avg-cpu-load endpoint
	router.Use(apiKeyMiddleware)

	router.GET("/cpu_usage/:seconds/:container", func(c *gin.Context) {
		seconds := c.Param("seconds")
		containerName := c.Param("container")
		averageCPUUsage, err := getAverageCPUUsage(c, promURL, seconds, containerName) // Pass the container name here
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"average_cpu_usage": averageCPUUsage})
	})

	apiPort := getEnvironment("PROM_PROXY_SERVER_PORT", "48080")

	// Run the server on specified port
	router.Run(fmt.Sprintf(":%s", apiPort))
}

// Middleware to check Bearer token
func apiKeyMiddleware(c *gin.Context) {
	apiKey := ""
	if getEnvironment("PROM_PROXY_SECURE_API_WITH_KEY", "False") == "True" {
		apiKey = getOrGenerateAPIKey()
	}

	if apiKey != "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			c.Abort()
			return
		}

		// Check if the header is in the format "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			c.Abort()
			return
		}

		// Check if the token matches the API key
		if parts[1] != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
	}
	c.Next()
}

// Function to get or generate API key
func getOrGenerateAPIKey() string {
	// Check if the secret key setting up through env variable
	apiKey := getEnvironment("PROM_PROXY_SECURE_API_KEY", "")
	if len(apiKey) > 0 {
		return apiKey
	}

	// Check if the secret file exists
	if _, err := os.Stat(secretFilePath); os.IsNotExist(err) {
		// Generate a new API key
		apiKey := generateAPIKey()
		// Store the API key in the secret file
		if err := os.WriteFile(secretFilePath, []byte(apiKey), 0600); err != nil {
			log.Fatalf("Failed to write API key to file: %v", err)
		}
		os.Setenv("PROM_PROXY_SECURE_API_KEY", apiKey)
		return apiKey
	}

	// Read the API key from the secret file
	apiKeyFile, err := os.ReadFile(secretFilePath)
	if err != nil {
		log.Fatalf("Failed to read API key from file: %v", err)
	}
	os.Setenv("PROM_PROXY_SECURE_API_KEY", string(apiKeyFile))
	return string(apiKeyFile)
}

// Function to generate a new API key
func generateAPIKey() string {
	bytes := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate API key: %v", err)
	}
	return hex.EncodeToString(bytes)
}

func getAverageCPULoad(promURL, containerNamePrefix string, seconds int) (map[string]float64, error) {
	// Create Prometheus API client
	client, err := api.NewClient(api.Config{
		Address: promURL,
	})
	if err != nil {
		return nil, err
	}
	apiV1 := v1.NewAPI(client)

	// Build the query
	query := fmt.Sprintf(
		"rate(container_cpu_usage_seconds_total{image!=\"\",name=~\"%s.*\"}[%ds]) * 100",
		containerNamePrefix,
		seconds,
	)
	// Query Prometheus
	result, _, err := apiV1.Query(
		context.Background(),
		query,
		time.Now().Add(-time.Duration(seconds)*time.Second),
	)
	if err != nil {
		return nil, err
	}

	// Parse the query result
	metricVec, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	// Collect average CPU load per container
	avgCPU := make(map[string]float64)
	for _, m := range metricVec {
		nameLabel := m.Metric["name"]
		// varName, ok := nameLabel.(model.LabelValue)
		// if !ok {
		// 	continue
		// }
		name := string(nameLabel)
		value := m.Value
		avgCPU[name] = float64(value)
	}

	return avgCPU, nil
}

func getAverageCPUUsage(c *gin.Context, promURL, seconds string, containerName string) (float64, error) {
	// Create a new Prometheus API client
	client, err := api.NewClient(api.Config{
		Address: promURL,
	})
	if err != nil {
		return 0, err
	}

	v1api := v1.NewAPI(client)

	// Define the query to get CPU usage for the specified duration and container name
	query := fmt.Sprintf(`avg(rate(container_cpu_usage_seconds_total{name="%s"}[%ss])) * 100`, containerName, seconds)

	// Execute the query
	result, warnings, err := v1api.Query(c, query, time.Now())
	if err != nil {
		return 0, err
	}
	if len(warnings) > 0 {
		fmt.Println("Warnings:", warnings)
	}

	// Parse the result
	if result.Type() == model.ValVector {
		vector := result.(model.Vector)
		if len(vector) > 0 {
			// Return the average CPU usage of the specified container
			return float64(vector[0].Value), nil
		}
	}

	return 0, fmt.Errorf("no data found for container: %s", containerName)
}

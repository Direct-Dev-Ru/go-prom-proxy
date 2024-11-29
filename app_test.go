package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	// v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	// Clean up
	os.Remove(secretFilePath)
	os.Exit(code)
}

func TestAPIKeyMiddleware(t *testing.T) {
	apiKey := getOrGenerateAPIKey()

	// Test valid token
	c, _ := gin.CreateTestContext(nil)
	c.Request, _ = http.NewRequest("GET", "/avg-cpu-load", nil)
	c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	apiKeyMiddleware(c)
	assert.Equal(t, http.StatusOK, c.Writer.Status())

	// Test missing header
	c, _ = gin.CreateTestContext(nil)
	c.Request, _ = http.NewRequest("GET", "/avg-cpu-load", nil)
	apiKeyMiddleware(c)
	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())

	// Test invalid token
	c, _ = gin.CreateTestContext(nil)
	c.Request, _ = http.NewRequest("GET", "/avg-cpu-load", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid_token")
	apiKeyMiddleware(c)
	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())

	// Test invalid header format
	c, _ = gin.CreateTestContext(nil)
	c.Request, _ = http.NewRequest("GET", "/avg-cpu-load", nil)
	c.Request.Header.Set("Authorization", "InvalidHeaderFormat")
	apiKeyMiddleware(c)
	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())
}

func TestGetOrGenerateAPIKey(t *testing.T) {
	// Ensure the file does not exist
	os.Remove(secretFilePath)

	// Generate and store the API key
	apiKey := getOrGenerateAPIKey()
	assert.NotEmpty(t, apiKey)

	// Read the API key from the file
	storedAPIKey, err := ioutil.ReadFile(secretFilePath)
	assert.NoError(t, err)
	assert.Equal(t, apiKey, string(storedAPIKey))

	// Ensure the API key is the same on subsequent calls
	apiKey2 := getOrGenerateAPIKey()
	assert.Equal(t, apiKey, apiKey2)
}

type MockPrometheusClient struct {
	mock.Mock
}

func (m *MockPrometheusClient) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	args := m.Called(ctx, query, ts)
	return args.Get(0).(model.Value), args.Error(1)
}

// func TestGetAverageCPULoad(t *testing.T) {
// 	mockClient := new(MockPrometheusClient)
// 	apiV1 := v1.API(mockClient)

// 	// Mock Prometheus response
// 	mockResult := model.Vector{
// 		&model.Sample{
// 			Metric: model.Metric{
// 				"name": "container_name1",
// 			},
// 			Value: 0.5,
// 		},
// 		&model.Sample{
// 			Metric: model.Metric{
// 				"name": "container_name2",
// 			},
// 			Value: 0.3,
// 		},
// 	}

// 	mockClient.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(mockResult, nil)

// 	avgCPU, err := getAverageCPULoad(apiV1, "container_name_prefix", 60)
// 	assert.NoError(t, err)
// 	assert.Equal(t, map[string]float64{
// 		"container_name1": 0.5,
// 		"container_name2": 0.3,
// 	}, avgCPU)
// }


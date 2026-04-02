package weather

import (
	"time"

	"piko.sh/piko"
)

type ForecastInput struct{}

type ForecastOutput struct {
	Timestamp   int64  `json:"timestamp"`
	Temperature int    `json:"temperature"`
	Condition   string `json:"condition"`
}

// ForecastAction returns weather data with response caching.
// See https://piko.sh/docs/guide/advanced-actions#caching
type ForecastAction struct {
	piko.ActionMetadata
}

func (a *ForecastAction) Call(input ForecastInput) (ForecastOutput, error) {
	return ForecastOutput{
		Timestamp:   time.Now().UnixMilli(),
		Temperature: 22,
		Condition:   "Partly Cloudy",
	}, nil
}

// CacheConfig enables response caching. Piko stores the serialised response
// keyed by action name + input, and returns it within the TTL window.
func (a *ForecastAction) CacheConfig() *piko.CacheConfig {
	return &piko.CacheConfig{
		TTL: 10 * time.Second,
	}
}

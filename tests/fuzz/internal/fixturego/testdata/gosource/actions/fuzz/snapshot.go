package fuzz

import (
	"sync/atomic"
	"time"

	"piko.sh/piko"
)

type SnapshotInput struct {
	Key string `json:"key"`
}
type SnapshotResult struct {
	Key       string `json:"key"`
	Language  string `json:"language"`
	Execution int64  `json:"execution"`
}

var snapshotExecutions atomic.Int64

type SnapshotAction struct{ piko.ActionMetadata }

func (a *SnapshotAction) CacheConfig() *piko.CacheConfig {
	ttl := time.Hour
	if a.Request().Headers.Get("Accept-Language") == "de" {
		ttl = 100 * time.Millisecond
	}
	return &piko.CacheConfig{TTL: ttl, VaryHeaders: []string{"Accept-Language"}}
}

func (a *SnapshotAction) Call(input SnapshotInput) (SnapshotResult, error) {
	return SnapshotResult{Key: input.Key, Language: a.Request().Headers.Get("Accept-Language"), Execution: snapshotExecutions.Add(1)}, nil
}

// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package provider_grpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

var _ tui_domain.FDsProvider = (*FDsProvider)(nil)

// FDsProvider provides file descriptor data via gRPC.
// It implements tui_domain.FDsProvider.
type FDsProvider struct {
	// conn holds the gRPC connection with health and metrics clients.
	conn *Connection

	// data holds the current file descriptor data; nil until first refresh.
	data *tui_domain.FDsData

	// mu guards access to data during reads and writes.
	mu sync.RWMutex

	// interval is the refresh interval between updates.
	interval time.Duration
}

// NewFDsProvider creates a new FDsProvider.
//
// Takes conn (*Connection) which is the shared gRPC connection.
// Takes interval (time.Duration) which is the refresh interval.
//
// Returns *FDsProvider which is the configured provider.
func NewFDsProvider(conn *Connection, interval time.Duration) *FDsProvider {
	return &FDsProvider{
		conn:     conn,
		data:     nil,
		mu:       sync.RWMutex{},
		interval: interval,
	}
}

// Name returns the provider name.
//
// Returns string which is the identifier "grpc-fds" for this provider.
func (*FDsProvider) Name() string {
	return "grpc-fds"
}

// Health checks if the gRPC connection is healthy.
//
// Returns error when the health check fails.
func (p *FDsProvider) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking file descriptors provider health via gRPC: %w", err)
	}
	return nil
}

// Close releases resources.
//
// Returns error when resources cannot be released; currently always nil.
func (*FDsProvider) Close() error {
	return nil
}

// RefreshInterval returns the refresh interval.
//
// Returns time.Duration which is the interval between data refreshes.
func (p *FDsProvider) RefreshInterval() time.Duration {
	return p.interval
}

// Refresh fetches the latest file descriptor data via gRPC.
//
// Returns error when the gRPC call fails.
//
// Safe for concurrent use.
func (p *FDsProvider) Refresh(ctx context.Context) error {
	return refreshProvider(ctx,
		func(ctx context.Context) (*tui_domain.FDsData, error) {
			response, err := p.conn.metricsClient.GetFileDescriptors(ctx, &pb.GetFileDescriptorsRequest{})
			if err != nil {
				return nil, err
			}
			return convertFDsData(response), nil
		},
		func(data *tui_domain.FDsData) {
			p.mu.Lock()
			p.data = data
			p.mu.Unlock()
		},
		"file descriptors",
	)
}

// GetFDs returns the current file descriptor information.
//
// Returns *tui_domain.FDsData which contains the file descriptor data.
// Returns error when no file descriptor data is available.
//
// Safe for concurrent use.
func (p *FDsProvider) GetFDs(_ context.Context) (*tui_domain.FDsData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.data == nil {
		return nil, errors.New("no file descriptor data available")
	}

	return p.data, nil
}

// convertFDsData converts protobuf FD data to domain FDsData.
//
// Takes response (*pb.GetFileDescriptorsResponse) which contains the protobuf
// response with file descriptor information.
//
// Returns *tui_domain.FDsData which contains the converted domain model.
func convertFDsData(response *pb.GetFileDescriptorsResponse) *tui_domain.FDsData {
	categories := make([]tui_domain.FDCategory, 0, len(response.GetCategories()))

	for _, cat := range response.GetCategories() {
		fds := make([]tui_domain.FDInfo, 0, len(cat.GetFds()))
		for _, fd := range cat.GetFds() {
			fds = append(fds, tui_domain.FDInfo{
				FD:        int(fd.GetFd()),
				Category:  fd.GetCategory(),
				Target:    fd.GetTarget(),
				FirstSeen: fd.GetFirstSeenMs(),
				AgeMs:     fd.GetAgeMs(),
			})
		}

		categories = append(categories, tui_domain.FDCategory{
			Category: cat.GetCategory(),
			Count:    int(cat.GetCount()),
			FDs:      fds,
		})
	}

	return &tui_domain.FDsData{
		Categories: categories,
		Total:      int(response.GetTotal()),
		Timestamp:  response.GetTimestampMs(),
	}
}

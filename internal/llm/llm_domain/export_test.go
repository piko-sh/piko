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

package llm_domain

import (
	"piko.sh/piko/wdk/clock"
)

type TestService = service

type TestEmbeddingService = embeddingService

func NewTestEmbeddingService(c clock.Clock) *TestEmbeddingService {
	if c == nil {
		return newEmbeddingService()
	}
	return newEmbeddingService(withEmbeddingServiceClock(c))
}

type TestEmbeddingServiceOption = embeddingServiceOption

func WithTestEmbeddingServiceClock(c clock.Clock) TestEmbeddingServiceOption {
	return withEmbeddingServiceClock(c)
}

func GetServiceClock(llmService Service) clock.Clock {
	if s, ok := llmService.(*service); ok {
		return s.clock
	}
	return nil
}

func GetCostCalculatorClock(cc *CostCalculator) clock.Clock {
	return cc.clock
}

func GetServiceVectorStore(llmService Service) VectorStorePort {
	if s, ok := llmService.(*service); ok {
		return s.vectorStore
	}
	return nil
}

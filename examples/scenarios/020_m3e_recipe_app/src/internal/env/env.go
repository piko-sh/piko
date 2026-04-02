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
//
// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package env

import (
	"context"
	"fmt"
	"sync"

	"github.com/sethvargo/go-envconfig"
)

type Env struct {
	// NormanKitchenEmail is the recipient email for contact form submissions.
	NormanKitchenEmail string `env:"NORMAN_KITCHEN_EMAIL,default=hello@thenormankitchen.com"`
}

var envInstance *Env
var onceEnv sync.Once

func Get() *Env {
	onceEnv.Do(func() {
		var envs Env
		if err := envconfig.Process(context.Background(), &envs); err != nil {
			fmt.Printf("Error processing environments: %v\n", err)
			panic(0)
		}
		envInstance = &envs
	})

	return envInstance
}

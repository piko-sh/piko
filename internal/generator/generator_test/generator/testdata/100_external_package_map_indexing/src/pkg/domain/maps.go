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

package domain

var ParishMap = map[string]string{
	"st-helier":   "St Helier",
	"st-peter":    "St Peter",
	"st-clement":  "St Clement",
	"st-brelade":  "St Brelade",
	"st-ouen":     "St Ouen",
	"grouville":   "Grouville",
	"trinity":     "Trinity",
	"st-mary":     "St Mary",
	"st-lawrence": "St Lawrence",
	"st-martin":   "St Martin",
	"st-saviour":  "St Saviour",
	"st-john":     "St John",
}

var StatusCodes = []int{200, 201, 204, 400, 404, 500}

var AppName = "Piko Test App"

const MaxRetries = 3

const DefaultTimeout = "30s"

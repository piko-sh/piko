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

import "time"

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

const MaxRetries int = 3

const AppVersion string = "1.0.0"

var AppName = "Piko Test"

var DefaultTimeout = 30 * time.Second

type Config struct {
	Name    string
	Timeout time.Duration
	Enabled bool
}

var DefaultConfig = Config{
	Name:    "default",
	Timeout: 60 * time.Second,
	Enabled: true,
}

type Result[T any] struct {
	Value T
	Error error
}

var StringResult = Result[string]{Value: "success", Error: nil}

type Address struct {
	Street  string
	City    string
	Country Country
}

type Country struct {
	Name    string
	Code    string
	Regions []Region
}

type Region struct {
	Name       string
	Population int
	Capital    City
}

type City struct {
	Name     string
	PostCode string
}

type DeepConfigType struct {
	Primary   Address
	Secondary Address
	Lookup    map[string]Address
}

var DefaultUser = &User{
	ID:   "user-123",
	Name: "Default User",
}

type User struct {
	ID   string
	Name string
}

type BaseEntity struct {
	CreatedAt string
	UpdatedAt string
}

type Article struct {
	BaseEntity
	Title   string
	Content string
}

var DefaultArticle = Article{
	BaseEntity: BaseEntity{
		CreatedAt: "2024-01-01",
		UpdatedAt: "2024-01-15",
	},
	Title:   "Test Article",
	Content: "This is test content.",
}

type ID = string

type UserID string

var CurrentUserID ID = "user-abc-123"

var OwnerID UserID = "owner-xyz-789"

var DeepConfig = DeepConfigType{
	Primary: Address{
		Street: "123 Main St",
		City:   "London",
		Country: Country{
			Name: "United Kingdom",
			Code: "UK",
			Regions: []Region{
				{
					Name:       "England",
					Population: 56000000,
					Capital: City{
						Name:     "London",
						PostCode: "EC1A",
					},
				},
			},
		},
	},
	Secondary: Address{
		Street: "456 High St",
		City:   "Manchester",
		Country: Country{
			Name:    "United Kingdom",
			Code:    "UK",
			Regions: nil,
		},
	},
	Lookup: map[string]Address{
		"home": {
			Street: "789 Oak Ave",
			City:   "Bristol",
			Country: Country{
				Name: "United Kingdom",
				Code: "UK",
				Regions: []Region{
					{
						Name:       "South West",
						Population: 5600000,
						Capital: City{
							Name:     "Bristol",
							PostCode: "BS1",
						},
					},
				},
			},
		},
	},
}

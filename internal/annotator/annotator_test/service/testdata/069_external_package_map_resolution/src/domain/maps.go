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

// ParishMap maps parish codes to their display names.
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

// StatusCodes is a slice of common HTTP status codes.
var StatusCodes = []int{200, 201, 204, 400, 404, 500}

// MaxRetries is a typed int constant for configuration.
const MaxRetries int = 3

// AppVersion is a typed string constant.
const AppVersion string = "1.0.0"

// AppName is a simple string variable.
var AppName = "Piko Test"

// DefaultTimeout is a non-primitive time.Duration variable.
var DefaultTimeout = 30 * time.Second

// Config is a struct type variable.
type Config struct {
	Name    string
	Timeout time.Duration
	Enabled bool
}

// DefaultConfig is a struct variable.
var DefaultConfig = Config{
	Name:    "default",
	Timeout: 60 * time.Second,
	Enabled: true,
}

// Generic type example - a generic Result type.
type Result[T any] struct {
	Value T
	Error error
}

// StringResult is a concrete instance of the generic Result type.
var StringResult = Result[string]{Value: "success", Error: nil}

// Address represents a physical address.
type Address struct {
	Street  string
	City    string
	Country Country
}

// Country represents a country with regions.
type Country struct {
	Name    string
	Code    string
	Regions []Region
}

// Region represents a region within a country.
type Region struct {
	Name       string
	Population int
	Capital    City
}

// City represents a city.
type City struct {
	Name     string
	PostCode string
}

// DeepConfigType is a named type for deep configuration.
type DeepConfigType struct {
	Primary   Address
	Secondary Address
	Lookup    map[string]Address
}

// DefaultUser is a pointer to a User struct.
var DefaultUser = &User{
	ID:   "user-123",
	Name: "Default User",
}

// User represents a user.
type User struct {
	ID   string
	Name string
}

// BaseEntity contains common fields.
type BaseEntity struct {
	CreatedAt string
	UpdatedAt string
}

// Article embeds BaseEntity.
type Article struct {
	BaseEntity // embedded
	Title      string
	Content    string
}

// DefaultArticle is an article with embedded fields.
var DefaultArticle = Article{
	BaseEntity: BaseEntity{
		CreatedAt: "2024-01-01",
		UpdatedAt: "2024-01-15",
	},
	Title:   "Test Article",
	Content: "This is test content.",
}

// ID is a type alias for string.
type ID = string

// UserID is a named type (not alias) wrapping string.
type UserID string

// CurrentUserID is a variable of the alias type.
var CurrentUserID ID = "user-abc-123"

// OwnerID is a variable of the named type.
var OwnerID UserID = "owner-xyz-789"

// DeepConfig demonstrates deeply nested struct access.
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
			Name: "United Kingdom",
			Code: "UK",
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

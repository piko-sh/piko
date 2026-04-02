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

package cache_test

import "time"

type User struct {
	CreatedAt time.Time
	ID        string
	Name      string
	Email     string
	IsActive  bool
}

type Product struct {
	LastUpdated time.Time
	Name        string
	Category    string
	ID          int
	Price       float64
	InStock     bool
}

type Order struct {
	Metadata map[string]any
	ID       string
	UserID   string
	Status   string
	Products []Product
	Total    float64
}

type SimpleStruct struct {
	Value string
}

func CreateSampleUser(id string) User {
	return User{
		ID:        id,
		Name:      "Test User " + id,
		Email:     "user" + id + "@example.com",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		IsActive:  true,
	}
}

func CreateSampleProduct(id int) Product {
	return Product{
		ID:          id,
		Name:        "Product",
		Price:       99.99,
		Category:    "Electronics",
		InStock:     true,
		LastUpdated: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func CreateSampleOrder(id string) Order {
	return Order{
		ID:     id,
		UserID: "user-123",
		Products: []Product{
			CreateSampleProduct(1),
			CreateSampleProduct(2),
		},
		Total:  199.98,
		Status: "pending",
		Metadata: map[string]any{
			"source":   "web",
			"priority": 1,
		},
	}
}

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

package encoding_test

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"testing"
	"time"

	"piko.sh/piko/wdk/encoding"

	"github.com/google/uuid"
)

func TestNewV7At(t *testing.T) {
	t.Run("Generate UUID at a given time", func(t *testing.T) {
		testTime := time.Date(2023, time.January, 14, 10, 30, 0, 0, time.UTC)
		u, err := encoding.NewV7At(testTime)
		if err != nil {
			t.Fatalf("Unexpected error generating UUID: %v", err)
		}

		if v := u.Version(); v != 7 {
			t.Errorf("Expected UUID version 7, got %d", v)
		}
	})

	t.Run("Ascending order for out-of-order times", func(t *testing.T) {
		later := time.Date(2023, time.January, 14, 11, 30, 0, 0, time.UTC)
		earlier := time.Date(2022, time.December, 31, 23, 59, 59, 0, time.UTC)

		u1, err := encoding.NewV7At(later)
		if err != nil {
			t.Fatalf("First UUID generation error: %v", err)
		}
		u2, err := encoding.NewV7At(earlier)
		if err != nil {
			t.Fatalf("Second UUID generation error: %v", err)
		}

		if bytes.Compare(u2[:], u1[:]) <= 0 {
			t.Errorf("Expected u2 to be greater than u1 (ascending), but it was not.\nu1: %v\nu2: %v",
				u1.String(), u2.String())
		}
	})

	t.Run("Identical times produce ascending values", func(t *testing.T) {
		testTime := time.Date(2023, time.January, 14, 12, 0, 0, 0, time.UTC)
		u1, err := encoding.NewV7At(testTime)
		if err != nil {
			t.Fatalf("First UUID generation error: %v", err)
		}
		u2, err := encoding.NewV7At(testTime)
		if err != nil {
			t.Fatalf("Second UUID generation error: %v", err)
		}

		if bytes.Compare(u2[:], u1[:]) <= 0 {
			t.Errorf("Expected second UUID to be greater than the first for identical times, but it was not.\nu1: %v\nu2: %v",
				u1.String(), u2.String())
		}
	})
}

func TestNewV7AtFromReader(t *testing.T) {
	t.Run("Generate UUID at a given time using custom reader", func(t *testing.T) {
		predictable := []byte{0xBA, 0xDB, 0xEE, 0xF0, 0x12, 0x34, 0x56, 0x78, 0x90, 0xAB, 0xCD, 0xEF,
			0xAA, 0xBB, 0xCC, 0xDD, 0x42, 0x42, 0x42, 0x42}
		r := bytes.NewReader(predictable)

		testTime := time.Date(2025, time.April, 1, 15, 45, 30, 0, time.UTC)

		u, err := encoding.NewV7AtFromReader(r, testTime)
		if err != nil {
			t.Fatalf("Unexpected error generating UUID from custom reader: %v", err)
		}

		if v := u.Version(); v != 7 {
			t.Errorf("Expected UUID version 7, got %d", v)
		}

		gotBytes := u[:]
		randomSegment := gotBytes[8:]
		if len(randomSegment) < 8 {
			t.Errorf("Unexpected random segment length: %v", len(randomSegment))
		}
	})

	t.Run("Multiple identical times using custom reader", func(t *testing.T) {
		r := bytes.NewReader(make([]byte, 100))
		testTime := time.Date(2025, time.April, 1, 16, 0, 0, 0, time.UTC)

		u1, err := encoding.NewV7AtFromReader(r, testTime)
		if err != nil {
			t.Fatalf("Unexpected error generating UUID: %v", err)
		}
		u2, err := encoding.NewV7AtFromReader(r, testTime)
		if err != nil {
			t.Fatalf("Unexpected error generating UUID: %v", err)
		}

		if bytes.Compare(u2[:], u1[:]) <= 0 {
			t.Errorf("Expected ascending order, got \nu1: %v\nu2: %v", u1.String(), u2.String())
		}
	})
}

func TestNewV7MinAt(t *testing.T) {
	testTime := time.Date(2025, time.June, 15, 12, 0, 0, 0, time.UTC)
	u := encoding.NewV7MinAt(testTime)

	if v := u.Version(); v != 7 {
		t.Errorf("expected version 7, got %d", v)
	}

	if u[8] != 0x80 {
		t.Errorf("expected variant byte 0x80, got 0x%02X", u[8])
	}

	for i := 9; i < 16; i++ {
		if u[i] != 0x00 {
			t.Errorf("expected random byte %d to be 0x00, got 0x%02X", i, u[i])
		}
	}

	if u[6]&0x0F != 0 || u[7] != 0 {
		t.Errorf("expected sub-ms sequence to be zero, got byte6=0x%02X byte7=0x%02X", u[6], u[7])
	}
}

func TestNewV7MaxAt(t *testing.T) {
	testTime := time.Date(2025, time.June, 15, 12, 0, 0, 0, time.UTC)
	u := encoding.NewV7MaxAt(testTime)

	if v := u.Version(); v != 7 {
		t.Errorf("expected version 7, got %d", v)
	}

	if u[8] != 0xBF {
		t.Errorf("expected variant byte 0xBF, got 0x%02X", u[8])
	}

	for i := 9; i < 16; i++ {
		if u[i] != 0xFF {
			t.Errorf("expected random byte %d to be 0xFF, got 0x%02X", i, u[i])
		}
	}

	if u[6]&0x0F != 0x0F || u[7] != 0xFF {
		t.Errorf("expected sub-ms sequence 0xFFF, got byte6=0x%02X byte7=0x%02X", u[6], u[7])
	}
}

func TestV7BoundaryOrdering(t *testing.T) {
	testTime := time.Date(2025, time.June, 15, 12, 0, 0, 0, time.UTC)
	minVal := encoding.NewV7MinAt(testTime)
	maxVal := encoding.NewV7MaxAt(testTime)

	if bytes.Compare(minVal[:], maxVal[:]) >= 0 {
		t.Errorf("expected min < max.\nMin: %s\nMax: %s", minVal, maxVal)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("simulated reader error")
}

func TestNewV7AtFromReader_Error(t *testing.T) {
	testTime := time.Date(2025, time.June, 15, 12, 0, 0, 0, time.UTC)
	_, err := encoding.NewV7AtFromReader(errReader{}, testTime)
	if err == nil {
		t.Error("expected error from failing reader, got nil")
	}
}

var (
	benchUUID uuid.UUID
	benchErr  error
)

func BenchmarkNewV7At(b *testing.B) {
	testTime := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchUUID, benchErr = encoding.NewV7At(testTime)
	}
}

func BenchmarkNewV7MinAt(b *testing.B) {
	testTime := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchUUID = encoding.NewV7MinAt(testTime)
	}
}

func BenchmarkNewV7MaxAt(b *testing.B) {
	testTime := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchUUID = encoding.NewV7MaxAt(testTime)
	}
}

func BenchmarkNewV7AtFromReader(b *testing.B) {
	customSeed := uint64(12345)
	detRand := rand.New(rand.NewPCG(customSeed, 0))
	rd := newReaderFromMathRand(detRand)
	uuid.SetRand(rd)
	defer uuid.SetRand(nil)

	testTime := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchUUID, benchErr = encoding.NewV7At(testTime)
	}
}

func TestThreadSafety(t *testing.T) {
	const goroutineCount = 20
	testTime := time.Now()
	results := make(chan uuid.UUID, goroutineCount)

	for range goroutineCount {
		go func() {
			u, err := encoding.NewV7At(testTime)
			if err != nil {
				t.Errorf("Unexpected error in goroutine: %v", err)
				results <- uuid.Nil
				return
			}
			results <- u
		}()
	}

	seen := make(map[uuid.UUID]bool, goroutineCount)
	for range goroutineCount {
		u := <-results
		if u == uuid.Nil {
			continue
		}
		if seen[u] {
			t.Errorf("Duplicate UUID generated under concurrent calls: %v", u)
		}
		seen[u] = true
	}
}

func TestPredictableWithCustomRand(t *testing.T) {
	customSeed := uint64(42)
	detRand := rand.New(rand.NewPCG(customSeed, 0))
	rd := newReaderFromMathRand(detRand)
	uuid.SetRand(rd)

	encoding.ResetLastV7timeAt()

	testTime := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	u1, err := encoding.NewV7At(testTime)
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}

	detRand2 := rand.New(rand.NewPCG(customSeed, 0))
	rd2 := newReaderFromMathRand(detRand2)
	uuid.SetRand(rd2)

	encoding.ResetLastV7timeAt()

	u2, err := encoding.NewV7At(testTime)
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}

	if !bytes.Equal(u1[:], u2[:]) {
		t.Errorf("Expected identical UUIDs for same time & random source, but got:\n  u1=%v\n  u2=%v",
			u1, u2)
	}

	uuid.SetRand(nil)
}

func TestPredictableAscending(t *testing.T) {
	customSeed := uint64(99)
	detRand := rand.New(rand.NewPCG(customSeed, 0))
	rd := newReaderFromMathRand(detRand)
	uuid.SetRand(rd)

	encoding.ResetLastV7timeAt()

	testTime := time.Date(2025, 3, 2, 12, 0, 0, 0, time.UTC)

	u1, err := encoding.NewV7At(testTime)
	if err != nil {
		t.Fatalf("failed to generate first: %v", err)
	}
	u2, err := encoding.NewV7At(testTime)
	if err != nil {
		t.Fatalf("failed to generate second: %v", err)
	}

	if bytes.Compare(u2[:], u1[:]) <= 0 {
		t.Errorf("Expected second UUID to be greater than the first:\n  u1=%v\n  u2=%v",
			u1, u2)
	}

	uuid.SetRand(nil)
}

func newReaderFromMathRand(r *rand.Rand) *mathRandReader {
	return &mathRandReader{r: r}
}

type mathRandReader struct {
	r *rand.Rand
}

func (m *mathRandReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(m.r.IntN(256))
	}
	return len(p), nil
}

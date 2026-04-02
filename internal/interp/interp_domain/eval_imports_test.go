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

package interp_domain

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalImports(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"fmt": {
			"Sprintf": reflect.ValueOf(fmt.Sprintf),
		},
		"strings": {
			"ToUpper":  reflect.ValueOf(strings.ToUpper),
			"Contains": reflect.ValueOf(strings.Contains),
			"Join":     reflect.ValueOf(strings.Join),
		},
	})

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "import fmt Sprintf",
			code: `import "fmt"
result := fmt.Sprintf("hello %s", "world")
result`,
			expect: "hello world",
		},
		{
			name: "import strings ToUpper",
			code: `import "strings"
result := strings.ToUpper("hello")
result`,
			expect: "HELLO",
		},
		{
			name: "import strings Contains",
			code: `import "strings"
result := strings.Contains("hello world", "world")
result`,
			expect: true,
		},
		{
			name: "import multiple packages",
			code: `import "fmt"
import "strings"
name := strings.ToUpper("world")
result := fmt.Sprintf("hello %s", name)
result`,
			expect: "hello WORLD",
		},
		{
			name: "import variadic Sprintf with int",
			code: `import "fmt"
result := fmt.Sprintf("value: %d", 42)
result`,
			expect: "value: 42",
		},
		{
			name: "grouped import",
			code: `import (
	"fmt"
	"strings"
)
name := strings.ToUpper("piko")
result := fmt.Sprintf("hello %s", name)
result`,
			expect: "hello PIKO",
		},
		{
			name: "Sprintf no varargs",
			code: `import "fmt"
fmt.Sprintf("hello")`,
			expect: "hello",
		},
		{
			name: "Sprintf three varargs",
			code: `import "fmt"
fmt.Sprintf("%s=%d (%.1f)", "x", 10, 3.14)`,
			expect: "x=10 (3.1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			service.UseSymbols(symbols)
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

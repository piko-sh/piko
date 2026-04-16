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

package annotator_domain

// Handles loop variable analysis for p-for directives by extracting, validating, and registering loop variables in the symbol table.
// Manages index and value variables, maintains proper scoping, and validates loop expressions for semantic correctness.

import (
	"fmt"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// maxLoopVarSearchIterations is the maximum number of attempts to find a
// unique loop variable name.
const maxLoopVarSearchIterations = 100

// loopVariableManagerPool reuses LoopVariableManager instances
// to reduce allocation pressure.
var loopVariableManagerPool = sync.Pool{
	New: func() any {
		return &LoopVariableManager{}
	},
}

// LoopVariableManager handles loop variable names during code generation.
// It checks that user-defined variables do not shadow built-in symbols and
// creates unique names for index variables in nested loops.
type LoopVariableManager struct {
	// ctx holds the analysis context for symbol lookup and diagnostics.
	ctx *AnalysisContext
}

// ValidateLoopVariable checks a single loop variable for shadowing reserved
// names. It emits warnings if the variable shadows built-in Piko system
// symbols or global functions.
//
// Takes variable (*ast_domain.Identifier) which is the loop variable to check.
// Takes forDirective (*ast_domain.Directive) which provides location context
// for diagnostics.
func (lvm *LoopVariableManager) ValidateLoopVariable(variable *ast_domain.Identifier, forDirective *ast_domain.Directive) {
	if variable == nil {
		return
	}

	if _, isReserved := reservedSystemSymbols[variable.Name]; isReserved {
		message := fmt.Sprintf("Loop variable '%s' shadows a built-in Piko system symbol. This may lead to unexpected behaviour.", variable.Name)
		finalLocation := forDirective.Location.Add(variable.RelativeLocation)
		lvm.ctx.addDiagnostic(ast_domain.Warning, message, variable.Name, finalLocation, forDirective.GoAnnotations, annotator_dto.CodeLoopVariableShadow)
		return
	}

	if _, isBuiltIn := builtInFunctions[variable.Name]; isBuiltIn {
		message := fmt.Sprintf("Loop variable '%s' shadows a global built-in function. This may lead to unexpected behaviour.", variable.Name)
		finalLocation := forDirective.Location.Add(variable.RelativeLocation)
		lvm.ctx.addDiagnostic(ast_domain.Warning, message, variable.Name, finalLocation, forDirective.GoAnnotations, annotator_dto.CodeLoopVariableShadow)
	}
}

// GenerateUniqueLoopVarName generates a unique loop variable name to avoid
// shadowing in nested loops.
//
// It checks the current context for existing __pikoLoopIdx variables and
// returns __pikoLoopIdx, __pikoLoopIdx2, __pikoLoopIdx3, etc. This avoids
// name collisions so each nested loop level has its own distinct index variable.
//
// Takes depth (int) which specifies the current loop nesting level.
//
// Returns string which is the unique loop variable name.
func (lvm *LoopVariableManager) GenerateUniqueLoopVarName(depth int) string {
	baseName := "__pikoLoopIdx"

	if _, found := lvm.ctx.Symbols.Find(baseName); !found {
		return baseName
	}

	for i := 2; i < maxLoopVarSearchIterations; i++ {
		candidateName := fmt.Sprintf("%s%d", baseName, i)
		if _, found := lvm.ctx.Symbols.Find(candidateName); !found {
			return candidateName
		}
	}

	return fmt.Sprintf("__pikoLoopIdx%d", depth)
}

// getLoopVariableManager gets a LoopVariableManager from the pool and sets it
// up for use.
//
// Takes ctx (*AnalysisContext) which provides the analysis context for the
// manager.
//
// Returns *LoopVariableManager which is ready to use.
func getLoopVariableManager(ctx *AnalysisContext) *LoopVariableManager {
	lvm, ok := loopVariableManagerPool.Get().(*LoopVariableManager)
	if !ok {
		lvm = &LoopVariableManager{}
	}
	lvm.ctx = ctx
	return lvm
}

// putLoopVariableManager resets the LoopVariableManager and returns it to
// the pool.
//
// Takes lvm (*LoopVariableManager) which is the manager to reset and return.
func putLoopVariableManager(lvm *LoopVariableManager) {
	lvm.ctx = nil
	loopVariableManagerPool.Put(lvm)
}

// newLoopVariableManager creates a new LoopVariableManager for the given
// analysis context.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
//
// Returns *LoopVariableManager which is the initialised manager.
func newLoopVariableManager(ctx *AnalysisContext) *LoopVariableManager {
	return &LoopVariableManager{ctx: ctx}
}

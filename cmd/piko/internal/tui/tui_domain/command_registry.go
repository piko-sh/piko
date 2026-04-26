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

package tui_domain

import (
	"slices"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
)

// CommandHandler runs a registered command.
//
// The handler receives the arguments parsed from the input line and the
// model so it can mutate state. Returning a non-nil tea.Cmd defers an
// action to the bubbletea dispatch loop.
type CommandHandler func(args []string, model *Model) tea.Cmd

// Command describes a single entry in the command palette.
type Command struct {
	// Run executes the command logic.
	Run CommandHandler

	// Name is the canonical name; matched against the typed input.
	Name string

	// Description is the short help line shown next to the command.
	Description string

	// Aliases is an optional list of alternate names.
	Aliases []string
}

// CommandRegistry stores commands keyed by name and alias for quick
// lookup. It is safe for concurrent reads after initialisation.
type CommandRegistry struct {
	// commands is the map of registered commands keyed by name and alias.
	commands map[string]Command

	// mu guards commands for safe concurrent access.
	mu sync.RWMutex
}

// NewCommandRegistry returns an empty registry.
//
// Returns *CommandRegistry ready to receive Register calls.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{commands: make(map[string]Command)}
}

// Register stores cmd under its Name and every Alias. Duplicate
// registrations panic to surface configuration errors at start-up.
//
// Takes cmd (Command) which is the entry to register.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *CommandRegistry) Register(cmd Command) {
	r.mu.Lock()
	defer r.mu.Unlock()

	keys := append([]string{cmd.Name}, cmd.Aliases...)
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := r.commands[key]; exists {
			panic("tui_domain: command " + key + " is already registered")
		}
		r.commands[key] = cmd
	}
}

// Lookup returns the command registered under name (or any of its aliases).
//
// Takes name (string) which is the canonical name or an alias.
//
// Returns Command which is the matched entry (zero value when missing).
// Returns bool which is true when a command was found.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *CommandRegistry) Lookup(name string) (Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.commands[name]
	return cmd, ok
}

// Complete returns commands whose names start with prefix, sorted
// alphabetically by name. Aliases are not returned to avoid duplicates.
//
// Takes prefix (string) which is the typed prefix.
//
// Returns []Command which is the (possibly empty) sorted list of
// candidates.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *CommandRegistry) Complete(prefix string) []Command {
	prefix = strings.TrimSpace(prefix)
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]struct{}, len(r.commands))
	out := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		if _, dup := seen[cmd.Name]; dup {
			continue
		}
		if !strings.HasPrefix(cmd.Name, prefix) {
			continue
		}
		seen[cmd.Name] = struct{}{}
		out = append(out, cmd)
	}
	slices.SortFunc(out, func(a, b Command) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// Names returns every registered name and alias, sorted.
//
// Returns []string which lists every key.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *CommandRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Commands returns the unique registered commands, sorted by canonical
// name. Aliases are deduplicated so each command appears once.
//
// Returns []Command which is the sorted list of unique commands.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *CommandRegistry) Commands() []Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{}, len(r.commands))
	out := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		if _, dup := seen[cmd.Name]; dup {
			continue
		}
		seen[cmd.Name] = struct{}{}
		out = append(out, cmd)
	}
	slices.SortFunc(out, func(a, b Command) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// ParseCommand splits an input line of the form ":name arg1 arg2" into
// the command name and arguments.
//
// The leading colon is optional. Quoted arguments are not currently
// supported; arguments are split on whitespace.
//
// Takes line (string) which is the user-typed input.
//
// Returns string which is the command name (lower-cased, trimmed).
// Returns []string which is the list of arguments.
// Returns bool which is true when a non-empty name was parsed.
func ParseCommand(line string) (string, []string, bool) {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, ":")
	if trimmed == "" {
		return "", nil, false
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", nil, false
	}

	return strings.ToLower(parts[0]), parts[1:], true
}

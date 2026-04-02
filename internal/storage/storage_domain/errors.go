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

package storage_domain

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/storage/storage_dto"
)

// storageError represents a failure for a single storage operation on a
// specific object. It implements the error interface.
type storageError struct {
	// Err is the underlying error that caused the storage failure.
	Err error

	// Key is the storage key that caused the error.
	Key string

	// Repository identifies which repository the error occurred in.
	Repository string
}

// Error returns a formatted string representation of the storage error.
//
// Returns string which contains the key, repository, and underlying error.
func (e *storageError) Error() string {
	return fmt.Sprintf("storage error for key '%s' in repository '%s': %v", e.Key, e.Repository, e.Err)
}

// multiError collects multiple storageError instances, typically from a bulk
// operation. It implements the error interface and allows for partial success,
// where an operation can complete for some objects while returning detailed
// errors for those that failed.
type multiError struct {
	// Errors holds the list of storage errors found during validation.
	Errors []*storageError
}

// Add appends a new error to the multiError collection.
//
// Takes repo (string) which identifies the repository where the error occurred.
// Takes key (string) which specifies the key linked to the error.
// Takes err (error) which is the error to add.
func (me *multiError) Add(repo string, key string, err error) {
	me.Errors = append(me.Errors, &storageError{
		Repository: repo,
		Key:        key,
		Err:        err,
	})
}

// HasErrors reports whether the multiError contains one or more errors.
//
// Returns bool which is true if there are errors, false otherwise.
func (me *multiError) HasErrors() bool {
	return len(me.Errors) > 0
}

// Error implements the standard error interface, providing a summary of all
// contained errors.
//
// Returns string which contains "no storage errors" when empty, the single
// error message when only one error exists, or a formatted summary of all
// errors joined by semicolons.
func (me *multiError) Error() string {
	if !me.HasErrors() {
		return "no storage errors"
	}

	if len(me.Errors) == 1 {
		return me.Errors[0].Error()
	}

	errorStrings := make([]string, 0, len(me.Errors))
	for _, e := range me.Errors {
		errorStrings = append(errorStrings, e.Error())
	}

	return fmt.Sprintf("%d storage errors occurred: %s", len(me.Errors), strings.Join(errorStrings, "; "))
}

// newMultiError creates a new multiError.
//
// Returns *multiError which is an empty error collection ready to gather
// errors.
func newMultiError() *multiError {
	return &multiError{
		Errors: make([]*storageError, 0),
	}
}

// batchResultToMultiError converts a BatchResult to a multiError for use with
// existing error handling code.
//
// Takes repo (string) which identifies the repository for error messages.
// Takes result (*storage_dto.BatchResult) which holds the batch operation
// outcome.
//
// Returns *multiError which collects all errors from failed batch keys, or nil
// when result is nil or has no errors.
func batchResultToMultiError(repo string, result *storage_dto.BatchResult) *multiError {
	if result == nil || !result.HasErrors() {
		return nil
	}

	multiErr := newMultiError()
	for _, failure := range result.FailedKeys {
		multiErr.Add(repo, failure.Key, fmt.Errorf("%s", failure.Error))
	}
	return multiErr
}

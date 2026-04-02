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

package email_domain

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"piko.sh/piko/internal/email/email_dto"
)

// validateSingle validates a single SendParams object. It checks for required
// content, validates the syntax of all recipient email addresses, and enforces
// limits for DoS protection.
//
// Takes params (*email_dto.SendParams) which specifies the email to validate.
// Takes config (ServiceConfig) which provides validation limits and settings.
//
// Returns error when params is nil, required content is missing, recipient
// limits are exceeded, payload size is too large, or email addresses are
// invalid.
func validateSingle(params *email_dto.SendParams, config ServiceConfig) error {
	if params == nil {
		return errors.New("validation failed: params cannot be nil")
	}

	if params.BodyPlain == "" && params.BodyHTML == "" {
		return errors.New("validation failed: either BodyPlain or BodyHTML must be provided")
	}
	if len(params.To) == 0 {
		return errors.New("validation failed: at least one recipient in the 'To' field is required")
	}

	totalRecipients := len(params.To) + len(params.Cc) + len(params.Bcc)
	if totalRecipients > config.MaxTotalRecipients {
		return fmt.Errorf("validation failed: total number of recipients (%d) exceeds the limit of %d", totalRecipients, config.MaxTotalRecipients)
	}

	var totalSize int64
	totalSize += int64(len(params.BodyHTML))
	totalSize += int64(len(params.BodyPlain))
	for _, attachment := range params.Attachments {
		totalSize += int64(len(attachment.Content))
	}

	if totalSize > config.MaxPayloadSizeBytes {
		return fmt.Errorf("validation failed: total message size (%.2f MB) exceeds the limit of %.2f MB",
			float64(totalSize)/1024/1024, float64(config.MaxPayloadSizeBytes)/1024/1024)
	}

	if err := validateAddressList(params.To, "To"); err != nil {
		return fmt.Errorf("validating To addresses: %w", err)
	}
	if err := validateAddressList(params.Cc, "Cc"); err != nil {
		return fmt.Errorf("validating Cc addresses: %w", err)
	}
	if err := validateAddressList(params.Bcc, "Bcc"); err != nil {
		return fmt.Errorf("validating Bcc addresses: %w", err)
	}

	return nil
}

// validateAndSplitBulk validates a slice of emails and separates them into
// valid and invalid batches. This enables a partial success strategy for bulk
// sending, where valid emails can be processed while detailed, actionable
// errors are returned for the invalid ones.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to validate.
// Takes config (ServiceConfig) which provides the validation settings.
//
// Returns validEmails ([]*email_dto.SendParams) which contains emails that
// passed validation.
// Returns errs (*MultiError) which contains details for each validation
// failure, or nil if all emails are valid.
func validateAndSplitBulk(emails []*email_dto.SendParams, config ServiceConfig) (validEmails []*email_dto.SendParams, errs *MultiError) {
	if len(emails) == 0 {
		return nil, nil
	}

	validEmails = make([]*email_dto.SendParams, 0, len(emails))
	var validationErrors []EmailError

	for i, email := range emails {
		if email == nil {
			emailErr := EmailError{
				Email:       email_dto.SendParams{},
				Error:       fmt.Errorf("validation failed for email at original index %d: email cannot be nil", i),
				Attempt:     1,
				LastAttempt: time.Now(),
			}
			validationErrors = append(validationErrors, emailErr)
			continue
		}

		err := validateSingle(email, config)
		if err != nil {
			emailErr := EmailError{
				Email:       *email,
				Error:       fmt.Errorf("validation failed for email at original index %d: %w", i, err),
				Attempt:     1,
				LastAttempt: time.Now(),
			}
			validationErrors = append(validationErrors, emailErr)
		} else {
			validEmails = append(validEmails, email)
		}
	}

	if len(validationErrors) > 0 {
		errs = &MultiError{Errors: validationErrors}
	}

	return validEmails, errs
}

// validateAddressList checks a list of email addresses for valid format using
// the standard library's RFC 5322 parser. It gathers all errors into a single
// error for simpler handling.
//
// Takes addresses ([]string) which contains the email addresses to check.
// Takes fieldName (string) which names the field for error messages.
//
// Returns error when one or more addresses fail RFC 5322 parsing.
func validateAddressList(addresses []string, fieldName string) error {
	if len(addresses) == 0 {
		return nil
	}

	var invalidEntries []string
	for _, raw := range addresses {
		addr := strings.TrimSpace(raw)
		if addr == "" {
			continue
		}
		if _, err := mail.ParseAddress(addr); err != nil {
			invalidEntries = append(invalidEntries, fmt.Sprintf("'%s' (%v)", addr, err))
		}
	}

	if len(invalidEntries) > 0 {
		return fmt.Errorf("invalid email address(es) found in '%s' field: %s",
			fieldName, strings.Join(invalidEntries, "; "))
	}

	return nil
}

// normaliseRecipientList trims whitespace from each address and removes empty
// strings.
//
// Takes list ([]string) which contains the recipient addresses to clean.
//
// Returns []string which contains the cleaned addresses with empty entries
// removed.
func normaliseRecipientList(list []string) []string {
	if len(list) == 0 {
		return list
	}
	out := make([]string, 0, len(list))
	for _, raw := range list {
		addr := strings.TrimSpace(raw)
		if addr == "" {
			continue
		}
		out = append(out, addr)
	}
	return out
}

// deduplicateRecipients removes duplicate addresses from a list, skipping any
// that already appear in the seen map. It adds each new address to the seen
// map as it goes.
//
// Takes list ([]string) which contains the addresses to check.
// Takes seen (map[string]struct{}) which tracks addresses already processed.
//
// Returns []string which contains only addresses not already in seen.
func deduplicateRecipients(list []string, seen map[string]struct{}) []string {
	unique := make([]string, 0, len(list))
	for _, addr := range list {
		if _, exists := seen[addr]; !exists {
			seen[addr] = struct{}{}
			unique = append(unique, addr)
		}
	}
	return unique
}

// shouldSetRecipientField checks whether a recipient field should be set based
// on its original state and the available values.
//
// Takes wasNil (bool) which is true if the field was originally nil.
// Takes uniqueList ([]string) which holds the unique recipient values.
//
// Returns bool which is true if the field should be set.
func shouldSetRecipientField(wasNil bool, uniqueList []string) bool {
	return !wasNil || len(uniqueList) > 0
}

// allRecipientsEmpty checks if all recipient lists are empty.
//
// Takes to ([]string) which contains the main recipients.
// Takes cc ([]string) which contains the copy recipients.
// Takes bcc ([]string) which contains the hidden copy recipients.
//
// Returns bool which is true when all three recipient lists are empty.
func allRecipientsEmpty(to, cc, bcc []string) bool {
	return len(to) == 0 && len(cc) == 0 && len(bcc) == 0
}

// sanitiseRecipients removes duplicate addresses across the To, Cc, and Bcc
// fields so each address appears only once.
//
// It gives priority to To over Cc over Bcc when removing duplicates. This
// stops the same address from getting the email more than once and fixes
// common input errors.
//
// When params is nil, returns straight away.
//
// Takes params (*email_dto.SendParams) which holds the recipient fields to
// clean.
func sanitiseRecipients(params *email_dto.SendParams) {
	if params == nil {
		return
	}

	wasToNil := params.To == nil
	wasCcNil := params.Cc == nil
	wasBccNil := params.Bcc == nil

	toList := normaliseRecipientList(params.To)
	ccList := normaliseRecipientList(params.Cc)
	bccList := normaliseRecipientList(params.Bcc)

	seen := make(map[string]struct{})

	params.To = deduplicateRecipients(toList, seen)

	uniqueCc := deduplicateRecipients(ccList, seen)
	if shouldSetRecipientField(wasCcNil, uniqueCc) {
		params.Cc = uniqueCc
	}

	uniqueBcc := deduplicateRecipients(bccList, seen)
	if shouldSetRecipientField(wasBccNil, uniqueBcc) {
		params.Bcc = uniqueBcc
	}

	if wasToNil && wasCcNil && wasBccNil && allRecipientsEmpty(params.To, params.Cc, params.Bcc) {
		params.To = []string{}
		params.Cc = []string{}
		params.Bcc = []string{}
	}
}

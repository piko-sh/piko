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

package security_domain

// SandboxToken represents a CSP sandbox directive token.
//
// Unlike other CSP sources, sandbox tokens are not quoted in the header output.
// They represent capabilities that are explicitly allowed within a sandboxed
// context. An empty sandbox directive applies maximum restrictions, while each
// token added permits a specific capability.
type SandboxToken string

const (
	// SandboxAllowDownloads permits file downloads via the download attribute or
	// navigation.
	SandboxAllowDownloads SandboxToken = "allow-downloads"

	// SandboxAllowForms permits form submission in a sandboxed iframe.
	SandboxAllowForms SandboxToken = "allow-forms"

	// SandboxAllowModals permits modal dialogues such as alert, confirm, prompt,
	// and the dialog element.
	SandboxAllowModals SandboxToken = "allow-modals"

	// SandboxAllowOrientationLock permits the page to lock the screen orientation.
	SandboxAllowOrientationLock SandboxToken = "allow-orientation-lock"

	// SandboxAllowPointerLock permits use of the Pointer Lock API.
	SandboxAllowPointerLock SandboxToken = "allow-pointer-lock"

	// SandboxAllowPopups allows popup windows using window.open() or target="_blank".
	SandboxAllowPopups SandboxToken = "allow-popups"

	// SandboxAllowPopupsToEscapeSandbox allows popups to open windows that
	// are not themselves sandboxed.
	SandboxAllowPopupsToEscapeSandbox SandboxToken = "allow-popups-to-escape-sandbox"

	// SandboxAllowPresentation allows use of the Presentation API.
	SandboxAllowPresentation SandboxToken = "allow-presentation"

	// SandboxAllowSameOrigin allows content to be treated as from its normal
	// origin. Without this, content is treated as from a unique, hidden origin.
	SandboxAllowSameOrigin SandboxToken = "allow-same-origin"

	// SandboxAllowScripts allows JavaScript to run in the sandboxed iframe.
	// This does not allow popups unless SandboxAllowPopups is also set.
	SandboxAllowScripts SandboxToken = "allow-scripts"

	// SandboxAllowStorageAccessByUserActivation allows the Storage Access API
	// to request access to unpartitioned cookies (experimental).
	SandboxAllowStorageAccessByUserActivation SandboxToken = "allow-storage-access-by-user-activation"

	// SandboxAllowTopNavigation allows the sandboxed content to navigate the
	// top-level browsing context.
	SandboxAllowTopNavigation SandboxToken = "allow-top-navigation"

	// SandboxAllowTopNavigationByUserActivation allows top-level navigation
	// only when triggered by a user gesture (click, etc.).
	SandboxAllowTopNavigationByUserActivation SandboxToken = "allow-top-navigation-by-user-activation"

	// SandboxAllowTopNavigationToCustomProtocols allows navigation to
	// non-http(s) URL schemes (e.g., mailto:, tel:).
	SandboxAllowTopNavigationToCustomProtocols SandboxToken = "allow-top-navigation-to-custom-protocols"
)

// validSandboxTokens is the set of all valid sandbox tokens.
var validSandboxTokens = map[SandboxToken]bool{
	SandboxAllowDownloads:                      true,
	SandboxAllowForms:                          true,
	SandboxAllowModals:                         true,
	SandboxAllowOrientationLock:                true,
	SandboxAllowPointerLock:                    true,
	SandboxAllowPopups:                         true,
	SandboxAllowPopupsToEscapeSandbox:          true,
	SandboxAllowPresentation:                   true,
	SandboxAllowSameOrigin:                     true,
	SandboxAllowScripts:                        true,
	SandboxAllowStorageAccessByUserActivation:  true,
	SandboxAllowTopNavigation:                  true,
	SandboxAllowTopNavigationByUserActivation:  true,
	SandboxAllowTopNavigationToCustomProtocols: true,
}

// isValidSandboxToken returns true if the token is a valid CSP sandbox token.
//
// Takes t (SandboxToken) which is the token to validate.
//
// Returns bool which is true if the token is valid, false otherwise.
func isValidSandboxToken(t SandboxToken) bool {
	return validSandboxTokens[t]
}

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

package llm_dto

// ContentPartType identifies the kind of content within a content part.
type ContentPartType string

const (
	// ContentPartTypeText indicates that the content part contains text.
	ContentPartTypeText ContentPartType = "text"

	// ContentPartTypeImageURL indicates an image part that is given by URL.
	ContentPartTypeImageURL ContentPartType = "image_url"

	// ContentPartTypeImageData indicates inline base64-encoded image data.
	ContentPartTypeImageData ContentPartType = "image_data"
)

// ContentPart represents a single part of message content.
// It is used for multi-modal messages that combine text and images.
type ContentPart struct {
	// Text holds the text content when Type is ContentPartTypeText.
	Text *string

	// ImageURL contains image URL info (when Type is ContentPartTypeImageURL).
	ImageURL *ImageURL

	// ImageData holds inline image data when Type is ContentPartTypeImageData.
	ImageData *ImageData

	// Type identifies the kind of content this part contains.
	Type ContentPartType
}

// ImageURL holds the URL and detail settings for an image in a vision request.
type ImageURL struct {
	// Detail specifies the image detail level; valid options are "auto", "low",
	// or "high". If nil, defaults to "auto".
	Detail *string

	// URL is the web address of the image.
	URL string
}

// ImageData holds base64-encoded image data for inline images.
type ImageData struct {
	// MIMEType is the image MIME type (e.g., "image/png", "image/jpeg").
	MIMEType string

	// Data is the image content encoded in base64 format.
	Data string
}

// TextPart creates a content part that holds text.
//
// Takes text (string) which is the text content to store.
//
// Returns ContentPart which is set up as a text part.
func TextPart(text string) ContentPart {
	return ContentPart{
		Type: ContentPartTypeText,
		Text: &text,
	}
}

// ImageURLPart creates an image URL content part.
//
// Takes url (string) which is the image URL.
// Takes detail (...string) which is the optional detail level ("auto", "low",
// "high").
//
// Returns ContentPart configured as image URL.
func ImageURLPart(url string, detail ...string) ContentPart {
	cp := ContentPart{
		Type: ContentPartTypeImageURL,
		ImageURL: &ImageURL{
			URL: url,
		},
	}
	if len(detail) > 0 && detail[0] != "" {
		cp.ImageURL.Detail = &detail[0]
	}
	return cp
}

// ImageDataPart creates an inline image content part.
//
// Takes mimeType (string) which is the image MIME type.
// Takes base64Data (string) which is the base64-encoded image data.
//
// Returns ContentPart configured as inline image.
func ImageDataPart(mimeType, base64Data string) ContentPart {
	return ContentPart{
		Type: ContentPartTypeImageData,
		ImageData: &ImageData{
			MIMEType: mimeType,
			Data:     base64Data,
		},
	}
}

// ImageDetailAuto returns a pointer to "auto" for use in ImageURL.Detail.
//
// Returns *string which contains the value "auto".
func ImageDetailAuto() *string {
	return new("auto")
}

// ImageDetailLow returns a pointer to "low" for use in ImageURL.Detail.
//
// Returns *string which is a pointer to the string "low".
func ImageDetailLow() *string {
	return new("low")
}

// ImageDetailHigh returns a pointer to "high" for use in ImageURL.Detail.
//
// Returns *string which is a pointer to the string "high".
func ImageDetailHigh() *string {
	return new("high")
}

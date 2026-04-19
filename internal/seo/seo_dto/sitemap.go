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

package seo_dto

import (
	"encoding/xml"
	"time"
)

// Sitemap represents the root XML structure for a sitemap document.
// It follows the Sitemap protocol at https://www.sitemaps.org/protocol.html.
type Sitemap struct {
	// XMLName identifies the XML element name for the sitemap URL set.
	XMLName xml.Name `xml:"urlset"`

	// Xmlns is the XML namespace attribute for the sitemap.
	Xmlns string `xml:"xmlns,attr"`

	// XmlnsImage is the XML namespace URI for image elements.
	XmlnsImage string `xml:"xmlns:image,attr,omitempty"`

	// XmlnsXhtml is the XHTML namespace URI for alternate language links.
	XmlnsXhtml string `xml:"xmlns:xhtml,attr,omitempty"`

	// XmlnsVideo is the XML namespace URI for video elements.
	XmlnsVideo string `xml:"xmlns:video,attr,omitempty"`

	// XmlnsNews is the XML namespace URI for news elements.
	XmlnsNews string `xml:"xmlns:news,attr,omitempty"`

	// URLs holds the list of URL entries in this sitemap.
	URLs []SitemapURL `xml:"url"`
}

// SitemapURL represents a single URL entry in a sitemap.
type SitemapURL struct {
	// News holds the news article entry for this page. Google allows at most
	// one news entry per URL.
	News *NewsEntry `xml:"news:news,omitempty"`

	// Location is the full URL of the page. Required.
	Location string `xml:"loc"`

	// LastMod is the date of last modification in W3C Datetime format
	// (YYYY-MM-DD).
	LastMod string `xml:"lastmod,omitempty"`

	// ChangeFreq is a hint to crawlers about how often the page may change.
	// Valid values: always, hourly, daily, weekly, monthly, yearly, never.
	ChangeFreq string `xml:"changefreq,omitempty"`

	// Priority is how important this URL is compared to other URLs on the site.
	// Valid range: 0.0 to 1.0; default is 0.5.
	Priority string `xml:"priority,omitempty"`

	// Alternates holds links to this page in other languages.
	Alternates []AlternateLink `xml:"xhtml:link,omitempty"`

	// Images lists image links on this page for better image search ranking.
	Images []ImageEntry `xml:"image:image,omitempty"`

	// Videos lists video entries on this page for better video search ranking.
	Videos []VideoEntry `xml:"video:video,omitempty"`
}

// AlternateLink represents an i18n alternate language link. It follows the
// hreflang standard for declaring language and regional variants of a page.
type AlternateLink struct {
	// Rel specifies the link relation type; must be "alternate" for hreflang
	// links.
	Rel string `xml:"rel,attr"`

	// Hreflang is the language code (e.g. "en", "en-GB", "fr", "es-MX").
	Hreflang string `xml:"hreflang,attr"`

	// Href is the full URL to the alternate language version.
	Href string `xml:"href,attr"`
}

// ImageEntry represents an image within a sitemap URL entry. Including images
// in sitemaps helps search engines discover and index them more effectively.
type ImageEntry struct {
	// Location is the full URL of the image file. Required.
	Location string `xml:"image:loc"`

	// Caption is a descriptive caption for the image.
	Caption string `xml:"image:caption,omitempty"`

	// Title is the title of the image.
	Title string `xml:"image:title,omitempty"`

	// License is the URL of the licence for the image.
	License string `xml:"image:license,omitempty"`

	// GeoLocation is the geographic location of the image (e.g. "London, United
	// Kingdom").
	GeoLocation string `xml:"image:geo_location,omitempty"`
}

// VideoEntry represents a video within a sitemap URL entry.
// It follows Google's video sitemap specification at
// developers.google.com/search/docs/crawling-indexing/sitemaps/video-sitemaps.
//
// Field order must match VideoInputEntry exactly so that direct type
// conversion (VideoEntry(input)) works without a field-by-field copy.
type VideoEntry struct {
	// ExpirationDate is the date the video becomes unavailable; W3C format.
	ExpirationDate string `xml:"video:expiration_date,omitempty"`

	// FamilyFriendly indicates whether the video is suitable for all audiences.
	// Valid values: "yes" (default) or "no".
	FamilyFriendly string `xml:"video:family_friendly,omitempty"`

	// Description is the text describing the video; required, maximum 2048
	// characters.
	Description string `xml:"video:description"`

	// ContentLocation is a direct URL to the video media file (e.g. .mp4).
	ContentLocation string `xml:"video:content_loc,omitempty"`

	// PlayerLocation is the URL of an embeddable player for the video.
	PlayerLocation string `xml:"video:player_loc,omitempty"`

	// Live indicates whether the video is a live stream.
	// Valid values: "yes" or "no".
	Live string `xml:"video:live,omitempty"`

	// Title is the video title. Required; max 100 characters recommended.
	Title string `xml:"video:title"`

	// Uploader is the name of the video uploader.
	Uploader string `xml:"video:uploader,omitempty"`

	// ThumbnailLocation is the URL of the video thumbnail image. Required.
	ThumbnailLocation string `xml:"video:thumbnail_loc"`

	// PublicationDate is the date the video was first published (W3C format).
	PublicationDate string `xml:"video:publication_date,omitempty"`

	// RequiresSubscription indicates whether a subscription is required.
	// Valid values: "yes" or "no".
	RequiresSubscription string `xml:"video:requires_subscription,omitempty"`

	// Tags is a list of tags associated with the video. Max 32 tags.
	Tags []string `xml:"video:tag,omitempty"`

	// ViewCount is the number of times the video has been viewed.
	ViewCount int `xml:"video:view_count,omitempty"`

	// Duration is the video duration in seconds. Recommended range: 1-28800.
	Duration int `xml:"video:duration,omitempty"`

	// Rating is the rating of the video. Allowed range: 0.0-5.0.
	Rating float32 `xml:"video:rating,omitempty"`
}

// NewsEntry represents a news article within a sitemap URL entry.
// It follows Google's news sitemap specification and permits at most one
// entry per URL.
type NewsEntry struct {
	// Publication identifies the news publication (name and language). Required.
	Publication NewsPublication `xml:"news:publication"`

	// PublicationDate is the article publication date in W3C format. Required.
	PublicationDate string `xml:"news:publication_date"`

	// Title is the title of the news article. Required.
	Title string `xml:"news:title"`
}

// NewsPublication identifies a news publication for Google News sitemaps.
type NewsPublication struct {
	// Name is the required name of the news publication (e.g. "The Example
	// Times").
	Name string `xml:"news:name"`

	// Language is the required language code of the publication (e.g. "en").
	Language string `xml:"news:language"`
}

// SitemapURLInput represents the JSON format for dynamic URL sources.
// External APIs such as a headless CMS should return a list of these objects.
type SitemapURLInput struct {
	// News holds the news article metadata for this page.
	News *NewsInputEntry `json:"news,omitempty"`

	// Location is the URL path, which can be relative or absolute.
	Location string `json:"loc"`

	// LastMod is the last modification date in any format that can be parsed.
	LastMod string `json:"lastmod,omitempty"`

	// ChangeFreq is a hint for crawlers (always, hourly, daily, weekly, monthly,
	// yearly, never).
	ChangeFreq string `json:"changefreq,omitempty"`

	// Images is a list of full image URLs for this page. For richer image
	// metadata (caption, title, licence), use ImageEntries instead.
	Images []string `json:"images,omitempty"`

	// Videos is a list of video metadata entries for this page.
	Videos []VideoInputEntry `json:"videos,omitempty"`

	// ImageEntries provides rich image metadata beyond simple URLs.
	// When populated, these take precedence over the Images string list.
	ImageEntries []ImageInputEntry `json:"imageEntries,omitempty"`

	// Priority is a value from 0.0 to 1.0; higher values mean more important.
	Priority float32 `json:"priority,omitempty"`
}

// VideoInputEntry is the JSON input format for video entries from dynamic
// sources.
//
// Field order must match VideoEntry exactly so that direct type conversion
// (VideoEntry(input)) works without a field-by-field copy.
type VideoInputEntry struct {
	// ExpirationDate is when the video expires (W3C format).
	ExpirationDate string `json:"expirationDate,omitempty"`

	// FamilyFriendly indicates suitability for all audiences ("yes" or "no").
	FamilyFriendly string `json:"familyFriendly,omitempty"`

	// Description is a description of the video. Required.
	Description string `json:"description"`

	// ContentLocation is a direct URL to the video media file.
	ContentLocation string `json:"contentLoc,omitempty"`

	// PlayerLocation is the URL of an embeddable player.
	PlayerLocation string `json:"playerLoc,omitempty"`

	// Live indicates whether this is a live stream ("yes" or "no").
	Live string `json:"live,omitempty"`

	// Title is the title of the video. Required.
	Title string `json:"title"`

	// Uploader is the name of the video uploader.
	Uploader string `json:"uploader,omitempty"`

	// ThumbnailLocation is the URL of the video thumbnail. Required.
	ThumbnailLocation string `json:"thumbnailLoc"`

	// PublicationDate is when the video was published (W3C format).
	PublicationDate string `json:"publicationDate,omitempty"`

	// RequiresSubscription indicates subscription requirement ("yes" or "no").
	RequiresSubscription string `json:"requiresSubscription,omitempty"`

	// Tags is a list of descriptive tags. Max 32 tags.
	Tags []string `json:"tags,omitempty"`

	// ViewCount is the number of views.
	ViewCount int `json:"viewCount,omitempty"`

	// Duration is the video duration in seconds.
	Duration int `json:"duration,omitempty"`

	// Rating is the video rating (0.0-5.0).
	Rating float32 `json:"rating,omitempty"`
}

// NewsInputEntry is the JSON input format for news entries from dynamic
// sources.
type NewsInputEntry struct {
	// PublicationName is the name of the news publication. Required.
	PublicationName string `json:"publicationName"`

	// PublicationLanguage is the required language code (e.g. "en").
	PublicationLanguage string `json:"publicationLanguage"`

	// PublicationDate is the article publication date (W3C format). Required.
	PublicationDate string `json:"publicationDate"`

	// Title is the title of the article. Required.
	Title string `json:"title"`
}

// ImageInputEntry is the JSON input format for rich image metadata from dynamic
// sources.
type ImageInputEntry struct {
	// Location is the full URL of the image. Required.
	Location string `json:"loc"`

	// Caption is a descriptive caption.
	Caption string `json:"caption,omitempty"`

	// Title is the title of the image.
	Title string `json:"title,omitempty"`

	// License is the URL of the image licence.
	License string `json:"license,omitempty"`

	// GeoLocation is the geographic location (e.g. "London, United Kingdom").
	GeoLocation string `json:"geoLocation,omitempty"`
}

// PageSEOMetadata contains SEO-relevant metadata extracted from a component's
// render function. This avoids the need for full page rendering when only
// metadata is required.
type PageSEOMetadata struct {
	// LastModified is the date when the page was last changed.
	// If nil, the file's modification time is used instead.
	LastModified *time.Time

	// RobotsRule specifies the robots meta tag value (e.g., "noindex, nofollow").
	// If empty, sensible defaults are used based on environment.
	RobotsRule string

	// SupportedLocales lists the language codes this page supports.
	// Used to build hreflang alternate links in the sitemap.
	SupportedLocales []string

	// ImageURLs is a list of image URLs discovered in the component's asset
	// references.
	ImageURLs []string
}

// SitemapIndex represents the root structure for a sitemap index file.
// It is used when a site has multiple sitemap files, typically for sites
// with 50,000+ URLs.
type SitemapIndex struct {
	// XMLName marks the type as an XML sitemapindex element.
	XMLName xml.Name `xml:"sitemapindex"`

	// Xmlns is the XML namespace attribute for the sitemap index.
	Xmlns string `xml:"xmlns,attr"`

	// Sitemaps holds the list of sitemap references in this index.
	Sitemaps []SitemapRef `xml:"sitemap"`
}

// SitemapRef represents a reference to a sitemap file within a sitemap index.
type SitemapRef struct {
	// Location is the full URL of the sitemap file.
	Location string `xml:"loc"`

	// LastMod is the date when the sitemap file was last changed.
	LastMod string `xml:"lastmod,omitempty"`
}

// SitemapBuildResult holds generated sitemaps and an optional index for a site.
//
// For small sites (URLs <= MaxURLsPerSitemap), only Sitemaps[0] is
// populated and Index is nil. For large sites, multiple Sitemaps are
// populated and Index contains references to them.
type SitemapBuildResult struct {
	// Index is the sitemap index file; nil when only one sitemap is needed.
	Index *SitemapIndex

	// Sitemaps holds one or more sitemap files for the build result. A single
	// sitemap produces one entry; split sitemaps produce multiple entries.
	Sitemaps []Sitemap
}

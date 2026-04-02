package main

// NavItem represents a single navigation link.
type NavItem struct {
	Label  string
	URL    string
	Active bool
}

// SiteConfig holds site-wide configuration used by all pages.
type SiteConfig struct {
	Title string
	Nav   []NavItem
}

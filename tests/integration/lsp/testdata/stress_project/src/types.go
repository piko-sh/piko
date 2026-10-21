package main

type NavItem struct {
	Label  string
	URL    string
	Active bool
}

type SiteConfig struct {
	Title string
	Nav   []NavItem
}

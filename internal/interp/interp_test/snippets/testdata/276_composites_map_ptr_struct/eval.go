package main

type SectionInfo struct {
	IconName    string
	Title       string
	Description string
}

func getSection(id string) *SectionInfo {
	sections := map[string]*SectionInfo{
		"get-started": {IconName: "rocket", Title: "Get Started", Description: "intro"},
		"guide":       {IconName: "book-open", Title: "Guide", Description: "learn"},
		"examples":    {IconName: "lightbulb", Title: "Examples", Description: "patterns"},
	}
	if info, ok := sections[id]; ok {
		return info
	}
	return &SectionInfo{IconName: "library", Title: "Docs", Description: ""}
}

func run() string {
	got := getSection("guide")
	fallback := getSection("missing")
	return got.Title + "/" + got.IconName + "/" + fallback.IconName
}

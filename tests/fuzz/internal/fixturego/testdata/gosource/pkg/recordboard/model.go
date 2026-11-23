package recordboard

import (
	"net/url"
	"os"
	"strconv"
)

type Record struct {
	ID, Label, Caption, Href, Name string
	Score                          int
}
type Board struct {
	Rows     []Record
	Revision int
}

func Revision() (int, error) {
	b, err := os.ReadFile("records-revision.txt")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(b))
}
func List() (Board, error) {
	rev, err := Revision()
	if err != nil {
		return Board{}, err
	}
	labels := []string{"First label", "Second label"}
	keys := []string{"amber", "blue"}
	selected := "amber"
	var rows []Record
	order := []int{0, 1}
	if rev >= 2 {
		order = []int{1, 0}
	}
	for _, i := range order {
		key := keys[i]
		label := labels[i]
		score := 1 + i
		if key == selected && rev >= 1 {
			label = "Edited label"
			score += 10
		}
		if rev == 3 && key != selected {
			continue
		}
		rows = append(rows, Record{ID: key, Label: label, Caption: key + " / " + label, Score: score, Name: "Open " + key, Href: "/record-detail?key=" + key + "&q=" + url.QueryEscape("sort order") + "#summary"})
	}
	return Board{Rows: rows, Revision: rev}, nil
}

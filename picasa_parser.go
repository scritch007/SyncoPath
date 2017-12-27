package main

import (
	"encoding/json"
	//"fmt"
)

type picasaTFeed struct {
	Value string `json:"$t"`
}

type picasaCategory struct {
	Term   string `json:"term"`
	Scheme string `json:"scheme"`
}

type picasaImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Medium string `json:"medium"`
	Type   string `json:"type"`
}
type picasaMediaGroup struct {
	Icon    []picasaImage `json:"media$thumbnail"`
	Content []picasaImage `json:"media$content"`
}

type picasaEntry struct {
	Category []picasaCategory `json:"category"`
	ID       picasaTFeed      `json:"gphoto$id"`
	Name     picasaTFeed      `json:"gphoto$name"`
	Width    picasaTFeed      `json:"gphoto$width"`
	Height   picasaTFeed      `json:"gphoto$height"`
	Media    picasaMediaGroup `json:"media$group"`
	Title    picasaTFeed      `json:"title"`
}

type picasaFeed struct {
	Entries []picasaEntry `json:"entry"`
	Title   picasaTFeed   `json:"title"`
	ID      picasaTFeed   `json:"title"`
}

type picasaMainResponse struct {
	Feed    picasaFeed `json:"feed"`
	Version string     `json:"version"`
}

func picasaParse(input []byte) (*picasaMainResponse, error) {
	m := new(picasaMainResponse)
	err := json.Unmarshal(input, m)
	//fmt.Printf("\n\n***\n%s\n****\n\n", string(input))
	return m, err
}

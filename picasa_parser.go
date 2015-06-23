package main

import (
	"encoding/json"
	//"fmt"
)

type PicasaTFeed struct {
	Value string `json:"$t"`
}

type PicasaCategory struct {
	Term   string `json:"term"`
	Scheme string `json:"scheme"`
}

type PicasaImage struct {
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Medium string `json:"medium"`
	Type   string `json:"type"`
}
type PicasaMediaGroup struct {
	Icon    []PicasaImage `json:"media$thumbnail"`
	Content []PicasaImage `json:"media$content"`
}

type PicasaEntry struct {
	Category []PicasaCategory `json:"category"`
	Id       PicasaTFeed      `json:"gphoto$id"`
	Name     PicasaTFeed      `json:"gphoto$name"`
	Width    PicasaTFeed      `json:"gphoto$width"`
	Height   PicasaTFeed      `json:"gphoto$height"`
	Media    PicasaMediaGroup `json:"media$group"`
	Title    PicasaTFeed      `json:"title"`
}

type PicasaFeed struct {
	Entries []PicasaEntry `json:"entry"`
	Title   PicasaTFeed   `json:"title"`
	Id      PicasaTFeed   `json:"title"`
}

type PicasaMainResponse struct {
	Feed    PicasaFeed `json:"feed"`
	Version string     `json:"version"`
}

func PicasaParse(input []byte) (*PicasaMainResponse, error) {
	m := new(PicasaMainResponse)
	err := json.Unmarshal(input, m)
	//fmt.Printf("\n\n***\n%s\n****\n\n", string(input))
	return m, err
}

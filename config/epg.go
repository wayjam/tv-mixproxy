package config

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

type EPG struct {
	XMLName   xml.Name       `xml:"tv"`
	Channel   []EPGChannel   `xml:"channel"`
	Programme []EPGProgramme `xml:"programme"`
}

type EPGChannel struct {
	ID          string   `xml:"id,attr"`
	DisplayName EPGTxt   `xml:"display-name"`
	URL         string   `xml:"url,omitempty"`
	Icon        *EPGIcon `xml:"icon,omitempty"`
}

type EPGProgramme struct {
	Channel         string    `xml:"channel,attr"`
	Start           string    `xml:"start,attr"`
	Stop            string    `xml:"stop,attr"`
	Title           EPGTxt    `xml:"title"`
	Desc            EPGTxt    `xml:"desc"`
	SubTitle        EPGTxt    `xml:"sub-title,omitempty"`
	Category        EPGTxt    `xml:"category,omitempty"`
	Icon            *EPGIcon  `xml:"icon,omitempty"`
	Subtitles       string    `xml:"subtitles,omitempty"`
	Premiere        string    `xml:"premiere,omitempty"`
	EpisodeNum      EPGValue  `xml:"episode-num,omitempty"`
	Rating          *EPGValue `xml:"rating,omitempty"`
	Date            string    `xml:"date,omitempty"`
	StarRating      *EPGValue `xml:"star-rating,omitempty"`
	PreviouslyShown string    `xml:"previously-shown,omitempty"`
}

type EPGIcon struct {
	Text string `xml:",chardata"`
	Src  string `xml:"src,attr"`
}

type EPGTxt struct {
	Text string `xml:",chardata"`
	Lang string `xml:"lang,attr"`
}

type EPGValue struct {
	Text   string `xml:",chardata"`
	Value  string `xml:"value"`
	System string `xml:"system,attr"`
}

func LoadEPGData(uri string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("failed to fetch data from URL: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func ParseEPGConfig(r io.Reader) (*EPG, error) {
	var epg EPG
	decoder := xml.NewDecoder(r)
	err := decoder.Decode(&epg)
	if err != nil {
		return nil, err
	}
	return &epg, nil
}

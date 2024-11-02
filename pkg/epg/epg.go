package epg

import (
	"encoding/xml"
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

func Unmarshal(data []byte) (*EPG, error) {
	var epg EPG
	err := xml.Unmarshal(data, &epg)
	if err != nil {
		return nil, err
	}
	return &epg, nil
}

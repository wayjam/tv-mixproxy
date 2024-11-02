package epg

import (
	"encoding/xml"
)

type EPG struct {
	XMLName   xml.Name    `xml:"tv"`
	Channel   []Channel   `xml:"channel"`
	Programme []Programme `xml:"programme"`
}

type Channel struct {
	ID          string `xml:"id,attr"`
	DisplayName Txt    `xml:"display-name"`
	URL         string `xml:"url,omitempty"`
	Icon        *Icon  `xml:"icon,omitempty"`
}

type Programme struct {
	Channel         string `xml:"channel,attr"`
	Start           string `xml:"start,attr"`
	Stop            string `xml:"stop,attr"`
	Title           Txt    `xml:"title"`
	Desc            Txt    `xml:"desc"`
	SubTitle        Txt    `xml:"sub-title,omitempty"`
	Category        Txt    `xml:"category,omitempty"`
	Icon            *Icon  `xml:"icon,omitempty"`
	Subtitles       string `xml:"subtitles,omitempty"`
	Premiere        string `xml:"premiere,omitempty"`
	EpisodeNum      *Value `xml:"episode-num,omitempty"`
	Rating          *Value `xml:"rating,omitempty"`
	Date            string `xml:"date,omitempty"`
	StarRating      *Value `xml:"star-rating,omitempty"`
	PreviouslyShown string `xml:"previously-shown,omitempty"`
}

type Icon struct {
	Text string `xml:",chardata"`
	Src  string `xml:"src,attr"`
}

type Txt struct {
	Text string `xml:",chardata"`
	Lang string `xml:"lang,attr"`
}

type Value struct {
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

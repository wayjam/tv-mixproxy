package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// FlexInt 是一个灵活的整数类型，可以从 JSON 中的数字或字符串解析
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	str := strings.TrimSpace(string(data))
	if str == "null" || str == "" {
		*fi = 0
		return nil
	}
	if str == "true" {
		*fi = 1
		return nil
	}
	if str == "false" {
		*fi = 0
		return nil
	}
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
		if str == "true" {
			*fi = 1
			return nil
		}
		if str == "false" || str == "" {
			*fi = 0
			return nil
		}
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		f, errF := strconv.ParseFloat(str, 64)
		if errF == nil {
			*fi = FlexInt(int(f))
			return nil
		}
		return fmt.Errorf("FlexInt: %w", err)
	}
	*fi = FlexInt(i)
	return nil
}

// MarshalJSON 实现了 json.Marshaler 接口
func (fi FlexInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(fi))
}

type TvBoxMultiRepoConfig struct {
	Repos []TvBoxRepoURLConfig `json:"urls"`
}

type TvBoxRepoURLConfig struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

type TvBoxRepoConfig struct {
	Spider    string       `json:"spider,omitempty"`
	Lives     []TvBoxLive  `json:"lives,omitempty"`
	Wallpaper string       `json:"wallpaper,omitempty"`
	Sites     []TvBoxSite  `json:"sites,omitempty"`
	Parses    []TvBoxParse `json:"parses,omitempty"`
	Flags     []string     `json:"flags,omitempty"`
	DOH       []TvBoxDOH   `json:"doh,omitempty"`
	Rules     []TvBoxRule  `json:"rules,omitempty"`
	Ads       []string     `json:"ads,omitempty"`
	Logo      string       `json:"logo,omitempty"` // 保留原有字段
}

type TvBoxSite struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Type        FlexInt `json:"type"`
	API         string  `json:"api,omitempty"`
	Searchable  FlexInt `json:"searchable,omitempty"`
	QuickSearch FlexInt `json:"quickSearch,omitempty"`
	Filterable  FlexInt `json:"filterable,omitempty"`
	Ext         any     `json:"ext,omitempty"`
	Jar         string  `json:"jar,omitempty"`
	PlayerType  FlexInt `json:"playerType,omitempty"`
	Changeable  FlexInt `json:"changeable,omitempty"`
	Timeout     FlexInt `json:"timeout,omitempty"`
}

type TvBoxStyle struct {
	Type  string  `json:"type"`
	Ratio float64 `json:"ratio,omitempty"`
}

type TvBoxDOH struct {
	Name string   `json:"name"`
	URL  string   `json:"url"`
	IPs  []string `json:"ips"`
}

type TvBoxLive struct {
	Name       string  `json:"name"`
	Type       FlexInt `json:"type"`
	URL        string  `json:"url"`
	PlayerType FlexInt `json:"playerType"`
	UA         string  `json:"ua,omitempty"`
	EPG        string  `json:"epg,omitempty"`
	Logo       string  `json:"logo,omitempty"`
	Timeout    FlexInt `json:"timeout,omitempty"`
}

type TvBoxParse struct {
	Name string  `json:"name"`
	Type FlexInt `json:"type"`
	URL  string  `json:"url"`
	Ext  any     `json:"ext,omitempty"`
}

type TvBoxRule struct {
	Name   string   `json:"name"`
	Hosts  []string `json:"hosts"`
	Regex  []string `json:"regex,omitempty"`
	Script []string `json:"script,omitempty"`
}

func LoadTvBoxData(uri string) ([]byte, error) {
	data, err := FetchData(uri)
	if err != nil {
		return nil, err
	}

	data = StripJSONComments(data)
	return data, nil
}

func ParseTvBoxMultiRepoConfig(r io.Reader) (*TvBoxMultiRepoConfig, error) {
	var config TvBoxMultiRepoConfig
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	return &config, nil
}

func ParseTvBoxConfig(r io.Reader) (*TvBoxRepoConfig, error) {
	var config TvBoxRepoConfig
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	return &config, nil
}

func LoadTvBoxMultiRepoConfig(uri string) (*TvBoxMultiRepoConfig, error) {
	data, err := LoadTvBoxData(uri)
	if err != nil {
		return nil, err
	}
	return ParseTvBoxMultiRepoConfig(bytes.NewReader(data))
}

func LoadTvBoxConfig(uri string) (*TvBoxRepoConfig, error) {
	data, err := LoadTvBoxData(uri)
	if err != nil {
		return nil, err
	}
	return ParseTvBoxConfig(bytes.NewReader(data))
}

// StripJSONComments removes both block and inline comments while preserving strings (including URLs)
func StripJSONComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	n := len(data)
	inString := false
	escaped := false

	for i := 0; i < n; i++ {
		c := data[i]

		if inString {
			out = append(out, c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}

		if c == '/' && i+1 < n && data[i+1] == '/' {
			i += 2
			for i < n && data[i] != '\n' {
				i++
			}
			if i < n {
				out = append(out, data[i])
			}
			continue
		}

		if c == '/' && i+1 < n && data[i+1] == '*' {
			i += 2
			for i+1 < n && !(data[i] == '*' && data[i+1] == '/') {
				i++
			}
			i++
			continue
		}

		out = append(out, c)
	}

	return out
}

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type MultiRepoConfig struct {
	Repos []RepoURLConfig `json:"urls"`
}

type RepoURLConfig struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

type TVBoxConfig struct {
	Spider    string `json:"spider"`
	Wallpaper string `json:"wallpaper"`
	Logo      string `json:"logo"`
	Sites     []Site `json:"sites"`
	DOH       []DOH  `json:"doh"`
	Lives     []Live `json:"lives"`
}

type Site struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        int    `json:"type"`
	API         string `json:"api"`
	Searchable  int    `json:"searchable"`
	QuickSearch int    `json:"quickSearch"`
	Filterable  int    `json:"filterable"`
	Changeable  int    `json:"changeable,omitempty"`
	PlayerType  int    `json:"playerType,omitempty"`
	Ext         any    `json:"ext,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
	Style       *Style `json:"style,omitempty"`
}

type Style struct {
	Type  string  `json:"type"`
	Ratio float64 `json:"ratio,omitempty"`
}

type DOH struct {
	Name string   `json:"name"`
	URL  string   `json:"url"`
	IPs  []string `json:"ips"`
}

type Live struct {
	Name       string `json:"name"`
	Type       int    `json:"type"`
	URL        string `json:"url"`
	PlayerType int    `json:"playerType,omitempty"`
	UA         string `json:"ua,omitempty"`
}

func LoadData(uri string) ([]byte, error) {
	var data []byte
	var err error

	if strings.HasPrefix(uri, "file://") {
		// Load from local file
		data, err = os.ReadFile(strings.TrimPrefix(uri, "file://"))
	} else if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		// Load from network URL
		resp, err := http.Get(uri)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch data from URL: %v", err)
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read data: %v", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported URI scheme: %s", uri)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read data: %v", err)
	}

	// Remove comments from JSON
	re := regexp.MustCompile(`(?m)^\s*//.*$|/\*[\s\S]*?\*/`)
	data = re.ReplaceAll(data, []byte{})

	return data, nil
}

func ParseMultiRepoConfig(data []byte) (*MultiRepoConfig, error) {
	var config MultiRepoConfig
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	return &config, nil
}

func ParseTvBoxConfig(data []byte) (*TVBoxConfig, error) {
	var config TVBoxConfig
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	return &config, nil
}

func LoadMultiRepoConfig(uri string) (*MultiRepoConfig, error) {
	data, err := LoadData(uri)
	if err != nil {
		return nil, err
	}
	return ParseMultiRepoConfig(data)
}

func LoadTvBoxConfig(uri string) (*TVBoxConfig, error) {
	data, err := LoadData(uri)
	if err != nil {
		return nil, err
	}
	return ParseTvBoxConfig(data)
}

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// FlexInt 是一个灵活的整数类型，可以从 JSON 中的数字或字符串解析
type FlexInt int

// UnmarshalJSON 实现了 json.Unmarshaler 接口
func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		// 如果是字符串，去掉引号
		data = data[1 : len(data)-1]
	}

	// 尝试将数据解析为整数
	i, err := strconv.Atoi(string(data))
	if err != nil {
		return fmt.Errorf("FlexInt: %w", err)
	}

	*fi = FlexInt(i)
	return nil
}

// MarshalJSON 实现了 json.Marshaler 接口
func (fi FlexInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(fi))
}

type MultiRepoConfig struct {
	Repos []RepoURLConfig `json:"urls"`
}

type RepoURLConfig struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

type RepoConfig struct {
	Spider    string   `json:"spider"`
	Lives     []Live   `json:"lives"`
	Wallpaper string   `json:"wallpaper"`
	Sites     []Site   `json:"sites"`
	Parses    []Parse  `json:"parses"`
	Flags     []string `json:"flags"`
	DOH       []DOH    `json:"doh"`
	Rules     []Rule   `json:"rules"`
	Ads       []string `json:"ads"`
	Logo      string   `json:"logo,omitempty"` // 保留原有字段
}

type Site struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Type        FlexInt `json:"type"`
	API         string  `json:"api"`
	Searchable  FlexInt `json:"searchable"`
	QuickSearch FlexInt `json:"quickSearch"`
	Filterable  FlexInt `json:"filterable"`
	Ext         any     `json:"ext,omitempty"`
	Jar         string  `json:"jar,omitempty"`
	PlayerType  FlexInt `json:"playerType,omitempty"`
	Changeable  FlexInt `json:"changeable,omitempty"`
	Timeout     FlexInt `json:"timeout,omitempty"`
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
	Name       string  `json:"name"`
	Type       FlexInt `json:"type"`
	URL        string  `json:"url"`
	PlayerType FlexInt `json:"playerType"`
	UA         string  `json:"ua,omitempty"`
	EPG        string  `json:"epg,omitempty"`
	Logo       string  `json:"logo,omitempty"`
	Timeout    FlexInt `json:"timeout,omitempty"`
}

type Parse struct {
	Name string  `json:"name"`
	Type FlexInt `json:"type"`
	URL  string  `json:"url"`
	Ext  any     `json:"ext,omitempty"`
}

type Rule struct {
	Name   string   `json:"name"`
	Hosts  []string `json:"hosts"`
	Regex  []string `json:"regex,omitempty"`
	Script []string `json:"script,omitempty"`
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

func ParseTvBoxConfig(data []byte) (*RepoConfig, error) {
	var config RepoConfig
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

func LoadTvBoxConfig(uri string) (*RepoConfig, error) {
	data, err := LoadData(uri)
	if err != nil {
		return nil, err
	}
	return ParseTvBoxConfig(data)
}

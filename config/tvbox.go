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

func ParseTvBoxMultiRepoConfig(data []byte) (*TvBoxMultiRepoConfig, error) {
	var config TvBoxMultiRepoConfig
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	return &config, nil
}

func ParseTvBoxConfig(data []byte) (*TvBoxRepoConfig, error) {
	var config TvBoxRepoConfig
	err := json.Unmarshal(data, &config)
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
	return ParseTvBoxMultiRepoConfig(data)
}

func LoadTvBoxConfig(uri string) (*TvBoxRepoConfig, error) {
	data, err := LoadTvBoxData(uri)
	if err != nil {
		return nil, err
	}
	return ParseTvBoxConfig(data)
}

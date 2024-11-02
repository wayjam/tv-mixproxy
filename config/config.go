package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort         int                `mapstructure:"server_port"`           // 服务端口, 默认 8080
	ExternalURL        string             `mapstructure:"external_url"`          // 外部访问地址, eg. http://localhost:8080
	Log                LogOpt             `mapstructure:"log"`                   // 日志配置
	Sources            []Source           `mapstructure:"sources"`               // 源配置
	TvBoxSingleRepoOpt TvBoxSingleRepoOpt `mapstructure:"tvbox_single_repo_opt"` // TvBox单仓源配置
	TvBoxMultiRepoOpt  TvBoxMultiRepoOpt  `mapstructure:"tvbox_multi_repo_opt"`  // TvBox多仓源配置
	EPGOpt             EPGOpt             `mapstructure:"epg"`                   // EPG源配置
	M3UOpt             M3UOpt             `mapstructure:"m3u"`                   // M3U源配置
}

func (c *Config) Fixture() {
	c.TvBoxSingleRepoOpt.Spider.Field = "spider"
	c.TvBoxSingleRepoOpt.Wallpaper.Field = "wallpaper"
	c.TvBoxSingleRepoOpt.Logo.Field = "logo"
	c.TvBoxSingleRepoOpt.Sites.Field = "sites"
	c.TvBoxSingleRepoOpt.DOH.Field = "doh"
	c.TvBoxSingleRepoOpt.Lives.Field = "lives"
	c.TvBoxSingleRepoOpt.Parses.Field = "parses"
	c.TvBoxSingleRepoOpt.Flags.Field = "flags"
	c.TvBoxSingleRepoOpt.Rules.Field = "rules"
	c.TvBoxSingleRepoOpt.Ads.Field = "ads"

	for i := range c.TvBoxMultiRepoOpt.Repos {
		c.TvBoxMultiRepoOpt.Repos[i].Field = "urls"
		c.TvBoxMultiRepoOpt.Repos[i].FilterBy = "name"
	}

	if c.TvBoxSingleRepoOpt.Fallback.SourceName != "" {
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.Spider)
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.Wallpaper)
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.Logo)
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.Sites.MixOpt)
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.DOH.MixOpt)
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.Lives.MixOpt)
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.Parses.MixOpt)
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.Flags.MixOpt)
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.Rules.MixOpt)
		c.fillFallbackSourceName(&c.TvBoxSingleRepoOpt.Ads.MixOpt)
	}

	// Set default interval for sources
	for i := range c.Sources {
		if c.Sources[i].Interval == 0 {
			c.Sources[i].Interval = 60
		}
	}
}

func (c *Config) fillFallbackSourceName(opt *MixOpt) {
	if opt.SourceName == "" {
		opt.SourceName = c.TvBoxSingleRepoOpt.Fallback.SourceName
	}
}

type LogOpt struct {
	Output string `mapstructure:"output"` // 日志输出路径, stdout 表示输出到标准输出
	Level  int    `mapstructure:"level"`  // 日志级别, 0: Trace, 1: Debug, 2: Info, 3: Warn, 4: Error, 5: Fatal, 6: Panic
}

type TvBoxSingleRepoOpt struct {
	Disable   bool        `mapstructure:"disable"` // 是否禁用单仓源
	Spider    MixOpt      `mapstructure:"spider"`
	Wallpaper MixOpt      `mapstructure:"wallpaper"`
	Logo      MixOpt      `mapstructure:"logo"`
	Sites     ArrayMixOpt `mapstructure:"sites"`
	DOH       ArrayMixOpt `mapstructure:"doh"`
	Lives     ArrayMixOpt `mapstructure:"lives"`
	Parses    ArrayMixOpt `mapstructure:"parses"`
	Flags     ArrayMixOpt `mapstructure:"flags"`
	Rules     ArrayMixOpt `mapstructure:"rules"`
	Ads       ArrayMixOpt `mapstructure:"ads"`
	Fallback  MixOpt      `mapstructure:"fallback"` // 降级配置
}

type TvBoxMultiRepoOpt struct {
	Disable           bool          `mapstructure:"disable"`             // 是否禁用多仓源
	IncludeSingleRepo bool          `mapstructure:"include_single_repo"` // 是否包含代理的单仓源
	Repos             []ArrayMixOpt `mapstructure:"repos"`               // 仓库配置
}

type EPGFilterType string

const (
	EPGFilterTypeChannelID    EPGFilterType = "channel_id"
	EPGFilterTypeProgramTitle EPGFilterType = "program_title"
)

type EPGOpt struct {
	Disable bool `mapstructure:"disable"` // 是否禁用EPG源
	// 过滤频道配置
	// 可根据 channel_id 或者 program_title 过滤
	// 支持多个源
	Filters []ArrayMixOpt `mapstructure:"filters"`
}

type M3UOpt struct {
	Disable               bool          `mapstructure:"disable"`                 // 是否禁用M3U源
	MediaPlaylistFallback MixOpt        `mapstructure:"media_playlist_fallback"` // 媒体播放列表降级配置
	MediaPlaylistFilters  []ArrayMixOpt `mapstructure:"media_playlist_filters"`  // 媒体播放列表过滤配置
	// MasterPlaylistFilters []ArrayMixOpt `mapstructure:"master_playlist_filters"` // 主播放列表过滤配置
}

type MixOpt struct {
	SourceName string `mapstructure:"source_name"`
	Field      string `mapstructure:"field"`    // 内部使用，无需配置
	Disabled   bool   `mapstructure:"disabled"` // 是否禁用该字段
}

type ArrayMixOpt struct {
	MixOpt   `mapstructure:",squash"`
	FilterBy string `mapstructure:"filter_by"` // 过滤依据 key
	Include  string `mapstructure:"include"`   // 包含, 正则
	Exclude  string `mapstructure:"exclude"`   // 排除, 正则
}

type Source struct {
	Name     string     `mapstructure:"name"`     // 源名称, 唯一标识， 用来标识用在配置中
	URL      string     `mapstructure:"url"`      // 源地址
	Type     SourceType `mapstructure:"type"`     // 源类型
	Interval int        `mapstructure:"interval"` // 源更新频率，单位为秒, 默认 60 秒
}

type SourceType string

const (
	SourceTypeTvBoxSingle SourceType = "tvbox_single" // tvbox单仓源
	SourceTypeTvBoxMulti  SourceType = "tvbox_multi"  // tvbox多仓源
	SourceTypeEPG         SourceType = "epg"          // epg源
	SourceTypeM3U         SourceType = "m3u"          // m3u源
)

func LoadServerConfig(cfgFile string) (*Config, error) {
	v := viper.New()

	if cfgFile != "" {
		// Use config file from the flag
		v.SetConfigFile(cfgFile)
	} else {
		// Search for config in the current directory with name "tv_mixproxy.yaml"
		v.AddConfigPath(".")
		v.SetConfigName("tv_mixproxy")
		v.SetConfigType("yaml")
	}

	v.SetEnvPrefix("TV_MIXPROXY")
	v.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in
	if err := v.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", v.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		// Config file was found but another error was produced
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %v", err)
	}

	cfg.Fixture()

	return &cfg, nil
}

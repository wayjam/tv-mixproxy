package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort    int           `mapstructure:"server_port"`     // 服务端口, 默认 8080
	ExternalURL   string        `mapstructure:"external_url"`    // 外部访问地址, eg. http://localhost:8080
	Log           LogOpt        `mapstructure:"log"`             // 日志配置
	Sources       []Source      `mapstructure:"sources"`         // 源配置
	SingleRepoOpt SingleRepoOpt `mapstructure:"single_repo_opt"` // 单仓源配置
	MultiRepoOpt  MultiRepoOpt  `mapstructure:"multi_repo_opt"`  // 多仓源配置
}

func (c *Config) Fixture() {
	c.SingleRepoOpt.Spider.Field = "spider"
	c.SingleRepoOpt.Wallpaper.Field = "wallpaper"
	c.SingleRepoOpt.Logo.Field = "logo"
	c.SingleRepoOpt.Sites.Field = "sites"
	c.SingleRepoOpt.DOH.Field = "doh"
	c.SingleRepoOpt.Lives.Field = "lives"
	c.SingleRepoOpt.Parses.Field = "parses"
	c.SingleRepoOpt.Flags.Field = "flags"
	c.SingleRepoOpt.Rules.Field = "rules"
	c.SingleRepoOpt.Ads.Field = "ads"

	for i := range c.MultiRepoOpt.Repos {
		c.MultiRepoOpt.Repos[i].Field = "urls"
		c.MultiRepoOpt.Repos[i].FilterBy = "name"
	}

	if c.SingleRepoOpt.Fallback.SourceName != "" {
		c.fillFallbackSourceName(&c.SingleRepoOpt.Spider)
		c.fillFallbackSourceName(&c.SingleRepoOpt.Wallpaper)
		c.fillFallbackSourceName(&c.SingleRepoOpt.Logo)
		c.fillFallbackSourceName(&c.SingleRepoOpt.Sites.MixOpt)
		c.fillFallbackSourceName(&c.SingleRepoOpt.DOH.MixOpt)
		c.fillFallbackSourceName(&c.SingleRepoOpt.Lives.MixOpt)
		c.fillFallbackSourceName(&c.SingleRepoOpt.Parses.MixOpt)
		c.fillFallbackSourceName(&c.SingleRepoOpt.Flags.MixOpt)
		c.fillFallbackSourceName(&c.SingleRepoOpt.Rules.MixOpt)
		c.fillFallbackSourceName(&c.SingleRepoOpt.Ads.MixOpt)
	}
}

func (c *Config) fillFallbackSourceName(opt *MixOpt) {
	if opt.SourceName == "" {
		opt.SourceName = c.SingleRepoOpt.Fallback.SourceName
	}
}

type LogOpt struct {
	Output string `mapstructure:"output"` // 日志输出路径, stdout 表示输出到标准输出
	Level  int    `mapstructure:"level"`  // 日志级别, 0: Trace, 1: Debug, 2: Info, 3: Warn, 4: Error, 5: Fatal, 6: Panic
}

type SingleRepoOpt struct {
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

type MultiRepoOpt struct {
	Disable           bool          `mapstructure:"disable"`             // 是否禁用多仓源
	IncludeSingleRepo bool          `mapstructure:"include_single_repo"` // 是否包含代理的单仓源
	Repos             []ArrayMixOpt `mapstructure:"repos"`               // 仓库配置
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
	Interval int        `mapstructure:"interval"` // 源更新频率，单位为秒
}

type SourceType string

const (
	SourceTypeSingle SourceType = "single" // 单仓源
	SourceTypeMulti  SourceType = "multi"  // 多仓源
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

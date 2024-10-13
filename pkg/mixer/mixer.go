package mixer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"github.com/tidwall/gjson"

	"github.com/wayjam/tvbox-mixproxy/config"
)

var (
	nullHandler fiber.Handler = func(c fiber.Ctx) error {
		return nil
	}
)

func NewMixURLHandler(
	mixOpt config.MixOpt, sourcer Sourcer,
) (fiber.Handler, error) {
	if mixOpt.Disabled || mixOpt.SourceName == "" {
		return nullHandler, nil
	}

	// 如果 source_name 以 file:// 开头，则返回一个 fiber 处理器，该处理器从文件系统中读取文件内容
	if strings.HasPrefix(mixOpt.SourceName, "file://") {
		// Extract file path from the source name
		filePath := strings.TrimPrefix(mixOpt.SourceName, "file://")

		// Return a fiber handler that serves the file content
		return func(c fiber.Ctx) error {
			return c.SendFile(filePath)
		}, nil
	}

	url, source, err := mixFieldAndGetSource(mixOpt, sourcer)
	if err != nil {
		return nullHandler, fmt.Errorf("mixing url: %w", err)
	}

	if source.Type() != config.SourceTypeSingle {
		return nullHandler, fmt.Errorf("source %s should be a single source", mixOpt.SourceName)
	}

	if url == "" {
		return nullHandler, nil
	}

	// 移除 URL 中可能存在的校验信息
	url = strings.Split(url, ";")[0]
	// 如果是相对路径，则返回一个 proxy 处理器，该处理器将请求转发到相对路径
	url = fullFillURL(url, source)

	return proxy.Forward(url), nil
}

// MixRepo 函数根据配置混合多个单仓源
func MixRepo(
	cfg *config.Config, sourcer Sourcer,
) (*config.RepoConfig, error) {
	result := &config.RepoConfig{
		Wallpaper: getExternalURL(cfg) + "/wallpaper?bg_color=333333&border_width=5&border_color=666666",
		Logo:      getExternalURL(cfg) + "/logo",
		Spider:    getExternalURL(cfg) + "/v1/spider",
	}
	singleRepoOpt := cfg.SingleRepoOpt

	// 混合 spider 字段
	if !singleRepoOpt.Spider.Disabled && singleRepoOpt.Spider.SourceName != "" {
		spider, source, err := mixFieldAndGetSource(singleRepoOpt.Spider, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing spider: %w", err)
		}
		if spider != "" {
			spider = fullFillURL(spider, source)
			result.Spider = spider
		}
	}

	// 混合 wallpaper 字段
	if !singleRepoOpt.Wallpaper.Disabled && singleRepoOpt.Wallpaper.SourceName != "" {
		wallpaper, err := mixField(singleRepoOpt.Wallpaper, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing wallpaper: %w", err)
		}
		result.Wallpaper = wallpaper
	}

	// 混合 logo 字段
	if !singleRepoOpt.Logo.Disabled && singleRepoOpt.Logo.SourceName != "" {
		logo, err := mixField(singleRepoOpt.Logo, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing logo: %w", err)
		}
		result.Logo = logo
	}

	// 混合 sites 数组
	if !singleRepoOpt.Sites.Disabled && singleRepoOpt.Sites.SourceName != "" {
		sites, source, err := mixArrayFieldAndGetSource[config.Site](singleRepoOpt.Sites, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing sites: %w", err)
		}
		// 处理 Site 结构体的特殊字段
		for i := range sites {
			site := processSiteFields(sites[i], source)
			result.Sites = append(result.Sites, site)
		}
	}

	// 混合 doh 数组
	if !singleRepoOpt.DOH.Disabled && singleRepoOpt.DOH.SourceName != "" {
		doh, source, err := mixArrayFieldAndGetSource[config.DOH](singleRepoOpt.DOH, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing doh: %w", err)
		}
		// 处理 DOH 结构体的特殊字段
		for i := range doh {
			dohItem := processDOHFields(doh[i], source)
			result.DOH = append(result.DOH, dohItem)
		}
	}

	// 混合 lives 数组
	if !singleRepoOpt.Lives.Disabled && singleRepoOpt.Lives.SourceName != "" {
		lives, source, err := mixArrayFieldAndGetSource[config.Live](singleRepoOpt.Lives, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing lives: %w", err)
		}
		// 处理 Site 结构体的特殊字段
		for i := range lives {
			live := processLiveFields(lives[i], source)
			result.Lives = append(result.Lives, live)
		}
	}

	// 混合 parses 数组
	if !singleRepoOpt.Parses.Disabled && singleRepoOpt.Parses.SourceName != "" {
		parses, source, err := mixArrayFieldAndGetSource[config.Parse](singleRepoOpt.Parses, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing parses: %w", err)
		}
		// 处理 Parse 结构体的特殊字段
		for i := range parses {
			parse := processParseFields(parses[i], source)
			result.Parses = append(result.Parses, parse)
		}
	}

	// 混合 flags 数组
	if !singleRepoOpt.Flags.Disabled && singleRepoOpt.Flags.SourceName != "" {
		flags, err := mixArrayField[string](singleRepoOpt.Flags, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing flags: %w", err)
		}
		result.Flags = flags
	}

	// 混合 rules 数组
	if !singleRepoOpt.Rules.Disabled && singleRepoOpt.Rules.SourceName != "" {
		rules, err := mixArrayField[config.Rule](singleRepoOpt.Rules, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing rules: %w", err)
		}
		result.Rules = rules
	}

	// 混合 ads 数组
	if !singleRepoOpt.Ads.Disabled && singleRepoOpt.Ads.SourceName != "" {
		ads, err := mixArrayField[string](singleRepoOpt.Ads, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing ads: %w", err)
		}
		result.Ads = ads
	}

	return result, nil
}

// mixField 混合单个字段
func mixField(opt config.MixOpt, sourcer Sourcer) (string, error) {
	value, _, err := mixFieldAndGetSource(opt, sourcer)
	if err != nil {
		return "", err
	}

	return value, nil
}

// mixFieldAndGetSource 混合单个字段并返回源
func mixFieldAndGetSource(opt config.MixOpt, sourcer Sourcer) (string, *Source, error) {
	source, err := sourcer.GetSource(opt.SourceName)
	if err != nil {
		return "", nil, fmt.Errorf("getting source %s: %w", opt.SourceName, err)
	}

	value := gjson.GetBytes(source.Data(), opt.Field)
	if !value.Exists() {
		// 如果字段不存在，返回空字符串而不是错误
		return "", source, nil
	}

	return value.String(), source, nil
}

// mixArrayField 混合数组字段
func mixArrayField[T any](opt config.ArrayMixOpt, sourcer Sourcer) ([]T, error) {
	array, _, err := mixArrayFieldAndGetSource[T](opt, sourcer)
	if err != nil {
		return nil, err
	}

	return array, nil
}

func mixArrayFieldAndGetSource[T any](opt config.ArrayMixOpt, sourcer Sourcer) ([]T, *Source, error) {
	source, err := sourcer.GetSource(opt.SourceName)
	if err != nil {
		return nil, nil, fmt.Errorf("getting source %s: %w", opt.SourceName, err)
	}

	array := gjson.GetBytes(source.Data(), opt.Field)
	if !array.Exists() || !array.IsArray() {
		// 如果字段不存在或不是数组，返回空切片而不是错误
		return []T{}, source, nil
	}

	filteredArray, err := filterArray(array.Array(), opt)
	if err != nil {
		return nil, source, fmt.Errorf("filtering array: %w", err)
	}

	var result []T
	for _, item := range filteredArray {
		var t T
		err := json.Unmarshal([]byte(item.Raw), &t)
		if err != nil {
			return nil, source, fmt.Errorf("unmarshal error: %w", err)
		}
		result = append(result, t)
	}

	return result, source, nil
}

// filterArray 根据配置过滤数组
func filterArray(array []gjson.Result, opt config.ArrayMixOpt) ([]gjson.Result, error) {
	var includeRegex, excludeRegex *regexp.Regexp
	var err error

	if opt.Include != "" {
		includeRegex, err = regexp.Compile(opt.Include)
		if err != nil {
			return nil, fmt.Errorf("invalid include regex: %w", err)
		}
	}

	if opt.Exclude != "" {
		excludeRegex, err = regexp.Compile(opt.Exclude)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude regex: %w", err)
		}
	}

	var result []gjson.Result
	for _, item := range array {
		if includeRegex == nil && excludeRegex == nil {
			result = append(result, item)
			continue
		}

		value := item.Get(opt.FilterBy).String()

		// 如果 include 为空或者匹配，并且 exclude 为空或者不匹配，则保留该项
		if (includeRegex == nil || includeRegex.MatchString(value)) &&
			(excludeRegex == nil || !excludeRegex.MatchString(value)) {
			result = append(result, item)
		}
	}

	return result, nil
}

// MixMultiRepo 函数根据配置混合多个多仓源
func MixMultiRepo(
	cfg *config.Config, sourcer Sourcer,
) (*config.MultiRepoConfig, error) {
	multiRepoOpt := cfg.MultiRepoOpt

	result := &config.MultiRepoConfig{
		Repos: make([]config.RepoURLConfig, 0),
	}

	// 如果需要包含单仓源
	if multiRepoOpt.IncludeSingleRepo {
		result.Repos = append(result.Repos, config.RepoURLConfig{
			Name: "TvBox MixProxy",
			URL:  getExternalURL(cfg) + "/v1/repo",
		})
	}

	for _, repoMixOpt := range multiRepoOpt.Repos {
		if !repoMixOpt.Disabled {
			repos, source, err := mixArrayFieldAndGetSource[config.RepoURLConfig](repoMixOpt, sourcer)
			if err != nil {
				return result, fmt.Errorf("mixing repos: %w", err)
			}
			for i := range repos {
				repo := processMultiRepoFields(repos[i], source)
				result.Repos = append(result.Repos, repo)
			}
		}
	}

	return result, nil
}

func getExternalURL(cfg *config.Config) (url string) {
	if cfg.ExternalURL == "" {
		url = fmt.Sprintf("http://localhost:%d", cfg.ServerPort)
	} else {
		url = cfg.ExternalURL
	}
	return
}

// fullFillURL 将相对路径转换为绝对路径
func fullFillURL(url string, source *Source) string {
	if strings.HasPrefix(url, "./") {
		baseURL := source.URL()
		lastSlashIndex := strings.LastIndex(baseURL, "/")
		if lastSlashIndex != -1 {
			baseURL = baseURL[:lastSlashIndex+1]
		}
		url = baseURL + strings.TrimPrefix(url, "./")
	}

	return url
}

func processSiteFields(item config.Site, source *Source) config.Site {
	if strings.HasPrefix(item.API, "./") {
		item.API = fullFillURL(item.API, source)
	}

	if strings.HasPrefix(item.Jar, "./") {
		item.Jar = fullFillURL(item.Jar, source)
	}

	switch ext := item.Ext.(type) {
	case string:
		if strings.HasPrefix(ext, "./") {
			item.Ext = fullFillURL(ext, source)
		}
	}

	return item
}

func processLiveFields(item config.Live, source *Source) config.Live {
	if strings.HasPrefix(item.URL, "./") {
		item.URL = fullFillURL(item.URL, source)
	}

	return item
}

func processMultiRepoFields(item config.RepoURLConfig, source *Source) config.RepoURLConfig {
	if strings.HasPrefix(item.URL, "./") {
		item.URL = fullFillURL(item.URL, source)
	}

	return item
}

func processDOHFields(item config.DOH, source *Source) config.DOH {
	if strings.HasPrefix(item.URL, "./") {
		item.URL = fullFillURL(item.URL, source)
	}
	return item
}

func processParseFields(item config.Parse, source *Source) config.Parse {
	if strings.HasPrefix(item.URL, "./") {
		item.URL = fullFillURL(item.URL, source)
	}
	return item
}

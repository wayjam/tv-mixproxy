package mixer

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/wayjam/tv-mixproxy/config"
)

// MixTvBoxRepo 函数根据配置混合多个单仓源
func MixTvBoxRepo(
	cfg *config.Config, sourcer Sourcer, opts ...MixOptions,
) (*config.TvBoxRepoConfig, error) {
	var token string
	if len(opts) > 0 {
		token = opts[0].Token
	}
	tokenParam := cfg.SecurityOpt.TokenParam
	if tokenParam == "" {
		tokenParam = "token"
	}

	spiderURL := getExternalURL(cfg) + "/v1/tvbox/spider"
	if token != "" {
		spiderURL = appendToken(spiderURL, token, tokenParam)
	}

	result := &config.TvBoxRepoConfig{
		Wallpaper: getExternalURL(cfg) + "/wallpaper?bg_color=333333&border_width=5&border_color=666666",
		Logo:      getExternalURL(cfg) + "/logo",
		Spider:    spiderURL,
	}
	singleRepoOpt := cfg.TvBoxSingleRepoOpt

	// 混合 spider 字段
	if singleRepoOpt.Spider.Disabled {
		result.Spider = ""
	} else if singleRepoOpt.Spider.SourceName != "" {
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
		if wallpaper != "" {
			result.Wallpaper = wallpaper
		}
	}

	// 混合 logo 字段
	if !singleRepoOpt.Logo.Disabled && singleRepoOpt.Logo.SourceName != "" {
		logo, err := mixField(singleRepoOpt.Logo, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing logo: %w", err)
		}
		if logo != "" {
			result.Logo = logo
		}
	}

	// Mix sites array
	for _, siteOpt := range singleRepoOpt.Sites {
		sites, source, err := mixArrayFieldAndGetSource[config.TvBoxSite](siteOpt, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing sites: %w", err)
		}
		for i := range sites {
			site := processSiteFields(sites[i], source)
			result.Sites = append(result.Sites, site)
		}
	}

	// Mix DOH array
	for _, dohOpt := range singleRepoOpt.DOH {
		doh, source, err := mixArrayFieldAndGetSource[config.TvBoxDOH](dohOpt, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing doh: %w", err)
		}
		for i := range doh {
			dohItem := processDOHFields(doh[i], source)
			result.DOH = append(result.DOH, dohItem)
		}
	}

	// Mix lives array
	for _, liveOpt := range singleRepoOpt.Lives {
		if liveOpt.Disabled {
			continue
		}
		if liveOpt.SourceName == "internal/m3u" {
			if !cfg.M3UOpt.Disable {
				epgURL := ""
				if !cfg.EPGOpt.Disable {
					epgURL = getExternalURL(cfg) + "/v1/epg.xml"
					if token != "" {
						epgURL = appendToken(epgURL, token, tokenParam)
					}
				}
				mediaURL := getExternalURL(cfg) + "/v1/m3u/media_playlist"
				if token != "" {
					mediaURL = appendToken(mediaURL, token, tokenParam)
				}
				result.Lives = append(result.Lives, config.TvBoxLive{
					Name:       "TV MixProxy",
					Type:       0,
					URL:        mediaURL,
					PlayerType: 1,
					EPG:        epgURL,
					Logo:       getExternalURL(cfg) + "/logo",
				})
			}
			continue
		}

		lives, source, err := mixArrayFieldAndGetSource[config.TvBoxLive](liveOpt, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing lives: %w", err)
		}
		for i := range lives {
			live := processLiveFields(lives[i], source)
			result.Lives = append(result.Lives, live)
		}
	}

	// Mix parses array
	for _, parseOpt := range singleRepoOpt.Parses {
		parses, source, err := mixArrayFieldAndGetSource[config.TvBoxParse](parseOpt, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing parses: %w", err)
		}
		for i := range parses {
			parse := processParseFields(parses[i], source)
			result.Parses = append(result.Parses, parse)
		}
	}

	// Mix flags array
	for _, flagOpt := range singleRepoOpt.Flags {
		flags, err := mixArrayField[string](flagOpt, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing flags: %w", err)
		}
		result.Flags = append(result.Flags, flags...)
	}

	// Mix rules array
	for _, ruleOpt := range singleRepoOpt.Rules {
		rules, err := mixArrayField[config.TvBoxRule](ruleOpt, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing rules: %w", err)
		}
		result.Rules = append(result.Rules, rules...)
	}

	// Mix ads array
	for _, adOpt := range singleRepoOpt.Ads {
		ads, err := mixArrayField[string](adOpt, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing ads: %w", err)
		}
		result.Ads = append(result.Ads, ads...)
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
	cfg *config.Config, sourcer Sourcer, opts ...MixOptions,
) (*config.TvBoxMultiRepoConfig, error) {
	var token string
	if len(opts) > 0 {
		token = opts[0].Token
	}
	tokenParam := cfg.SecurityOpt.TokenParam
	if tokenParam == "" {
		tokenParam = "token"
	}
	multiRepoOpt := cfg.TvBoxMultiRepoOpt

	result := &config.TvBoxMultiRepoConfig{
		Repos: make([]config.TvBoxRepoURLConfig, 0),
	}

	// 如果需要包含单仓源
	if multiRepoOpt.IncludeSingleRepo {
		singleRepoURL := getExternalURL(cfg) + "/v1/tvbox/repo"
		if token != "" {
			singleRepoURL = appendToken(singleRepoURL, token, tokenParam)
		}
		result.Repos = append(result.Repos, config.TvBoxRepoURLConfig{
			Name: "Tv MixProxy",
			URL:  singleRepoURL,
		})
	}

	for _, repoMixOpt := range multiRepoOpt.Repos {
		if !repoMixOpt.Disabled {
			repos, source, err := mixArrayFieldAndGetSource[config.TvBoxRepoURLConfig](repoMixOpt, sourcer)
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

func isRelativePath(raw string) bool {
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") {
		return true
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return true
	}
	return false
}

// fullFillURL 将相对路径转换为绝对路径，并保留 ;md5; 校验后缀
func fullFillURL(rawURL string, source *Source) string {
	if rawURL == "" || source == nil || source.URL() == "" {
		return rawURL
	}

	lower := strings.ToLower(rawURL)
	if strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "file://") ||
		strings.HasPrefix(lower, "clan://") ||
		strings.HasPrefix(lower, "tvbox://") {
		return rawURL
	}

	parts := strings.Split(rawURL, ";")
	relPath := parts[0]
	md5Suffix := ""
	if len(parts) > 1 {
		md5Suffix = ";" + strings.Join(parts[1:], ";")
	}

	baseURLStr := source.URL()
	if strings.HasPrefix(baseURLStr, "file://") {
		basePath := strings.TrimPrefix(baseURLStr, "file://")
		dir := filepath.Dir(basePath)
		resolvedPath := filepath.Clean(filepath.Join(dir, relPath))
		return "file://" + resolvedPath + md5Suffix
	}

	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return rawURL
	}

	relURL, err := url.Parse(relPath)
	if err != nil {
		return rawURL
	}

	return baseURL.ResolveReference(relURL).String() + md5Suffix
}

func processSiteFields(item config.TvBoxSite, source *Source) config.TvBoxSite {
	if isRelativePath(item.API) {
		item.API = fullFillURL(item.API, source)
	}

	if item.Jar != "" && !strings.Contains(item.Jar, "://") {
		item.Jar = fullFillURL(item.Jar, source)
	}

	switch ext := item.Ext.(type) {
	case string:
		if isRelativePath(ext) {
			item.Ext = fullFillURL(ext, source)
		}
	}

	return item
}

func processLiveFields(item config.TvBoxLive, source *Source) config.TvBoxLive {
	if isRelativePath(item.URL) {
		item.URL = fullFillURL(item.URL, source)
	}
	if isRelativePath(item.EPG) {
		item.EPG = fullFillURL(item.EPG, source)
	}
	if isRelativePath(item.Logo) {
		item.Logo = fullFillURL(item.Logo, source)
	}

	return item
}

func processMultiRepoFields(item config.TvBoxRepoURLConfig, source *Source) config.TvBoxRepoURLConfig {
	if isRelativePath(item.URL) {
		item.URL = fullFillURL(item.URL, source)
	}

	return item
}

func processDOHFields(item config.TvBoxDOH, source *Source) config.TvBoxDOH {
	if isRelativePath(item.URL) {
		item.URL = fullFillURL(item.URL, source)
	}
	return item
}

func processParseFields(item config.TvBoxParse, source *Source) config.TvBoxParse {
	if isRelativePath(item.URL) {
		item.URL = fullFillURL(item.URL, source)
	}
	return item
}

package mixer

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/tidwall/gjson"

	"github.com/wayjam/tvbox-mixproxy/config"
)

// MixRepo 函数根据配置混合多个单仓源
func MixRepo(
	cfg *config.Config, sourcer Sourcer,
) (*config.TVBoxConfig, error) {
	result := &config.TVBoxConfig{
		Wallpaper: getExternalURL(cfg) + "/wallpaper?bg_color=333333&border_width=5&border_color=666666",
		Logo:      getExternalURL(cfg) + "/logo",
	}
	singleRepoOpt := cfg.SingleRepoOpt

	// 混合 spider 字段
	if singleRepoOpt.Spider.SourceName != "" {
		spider, err := mixField(singleRepoOpt.Spider, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing spider: %w", err)
		}
		result.Spider = spider
	}

	// 混合 wallpaper 字段
	if singleRepoOpt.Wallpaper.SourceName != "" {
		wallpaper, err := mixField(singleRepoOpt.Wallpaper, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing wallpaper: %w", err)
		}
		result.Wallpaper = wallpaper
	}

	// 混合 logo 字段
	if singleRepoOpt.Logo.SourceName != "" {
		logo, err := mixField(singleRepoOpt.Logo, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing logo: %w", err)
		}
		result.Logo = logo
	}

	// 混合 sites 数组
	if singleRepoOpt.Sites.SourceName != "" {
		sites, err := mixArrayField[config.Site](singleRepoOpt.Sites, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing sites: %w", err)
		}
		result.Sites = sites
	}

	// 混合 doh 数组
	if singleRepoOpt.DOH.SourceName != "" {
		doh, err := mixArrayField[config.DOH](singleRepoOpt.DOH, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing doh: %w", err)
		}
		result.DOH = doh
	}

	// 混合 lives 数组
	if singleRepoOpt.Lives.SourceName != "" {
		lives, err := mixArrayField[config.Live](singleRepoOpt.Lives, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing lives: %w", err)
		}
		result.Lives = lives
	}

	return result, nil
}

// mixField 混合单个字段
func mixField(opt config.MixOpt, sourcer Sourcer) (string, error) {
	source, err := sourcer.GetSource(opt.SourceName)
	if err != nil {
		return "", fmt.Errorf("getting source %s: %w", opt.SourceName, err)
	}

	value := gjson.GetBytes(source, opt.Field)
	if !value.Exists() {
		return "", fmt.Errorf("field %s not found in source %s", opt.Field, opt.SourceName)
	}

	return value.String(), nil
}

// mixArrayField 混合数组字段
func mixArrayField[T any](opt config.ArrayMixOpt, sourcer Sourcer) ([]T, error) {
	source, err := sourcer.GetSource(opt.SourceName)
	if err != nil {
		return nil, fmt.Errorf("getting source %s: %w", opt.SourceName, err)
	}

	array := gjson.GetBytes(source, opt.Field)
	if !array.Exists() || !array.IsArray() {
		return nil, fmt.Errorf("field %s not found or not an array in source %s", opt.Field, opt.SourceName)
	}

	filteredArray, err := filterArray(array.Array(), opt)
	if err != nil {
		return nil, fmt.Errorf("filtering array: %w", err)
	}

	var result []T
	for _, item := range filteredArray {
		var t T
		err := json.Unmarshal([]byte(item.Raw), &t)
		if err != nil {
			return nil, fmt.Errorf("unmarshal error: %w", err)
		}
		result = append(result, t)
	}

	return result, nil
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
		repos, err := mixArrayField[config.RepoURLConfig](repoMixOpt, sourcer)
		if err != nil {
			return result, fmt.Errorf("mixing repos: %w", err)
		}
		result.Repos = append(result.Repos, repos...)
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

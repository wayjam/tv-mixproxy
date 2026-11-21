package cleaner

import (
	"regexp"
	"strings"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/m3u"
)

var defaultStripPatterns = []string{
	`(?i)\[?(?:IPv[46]|1080P|720P|2K|4K|8K|FHD|HD|SD|HEVC|H\.26[45]|60fps)\]?`,
	`(?i)(超清|高清|全高清|蓝光|标清|频道|台)`,
}

var defaultAliases = []config.AliasRule{
	{Canonical: "CCTV-1", Matches: []string{"CCTV1", "CCTV1综合", "CCTV-1 综合", "CCTV-1综合", "中央一台", "中央1台"}},
	{Canonical: "CCTV-2", Matches: []string{"CCTV2", "CCTV2财经", "CCTV-2 财经", "CCTV-2财经", "中央二台", "中央2台"}},
	{Canonical: "CCTV-3", Matches: []string{"CCTV3", "CCTV3综艺", "CCTV-3 综艺", "CCTV-3综艺", "中央三台", "中央3台"}},
	{Canonical: "CCTV-4", Matches: []string{"CCTV4", "CCTV4中文国际", "CCTV-4 中文国际", "CCTV-4中文国际", "中央四台", "中央4台"}},
	{Canonical: "CCTV-5", Matches: []string{"CCTV5", "CCTV5体育", "CCTV-5 体育", "CCTV-5体育", "中央五台", "中央5台"}},
	{Canonical: "CCTV-5+", Matches: []string{"CCTV5+", "CCTV5PLUS", "CCTV-5+ 体育赛事", "CCTV-5+体育赛事"}},
	{Canonical: "CCTV-6", Matches: []string{"CCTV6", "CCTV6电影", "CCTV-6 电影", "CCTV-6电影", "中央六台", "中央6台"}},
	{Canonical: "CCTV-7", Matches: []string{"CCTV7", "CCTV7国防军事", "CCTV-7 国防军事", "CCTV-7国防军事", "中央七台", "中央7台"}},
	{Canonical: "CCTV-8", Matches: []string{"CCTV8", "CCTV8电视剧", "CCTV-8 电视剧", "CCTV-8电视剧", "中央八台", "中央8台"}},
	{Canonical: "CCTV-9", Matches: []string{"CCTV9", "CCTV9纪录", "CCTV-9 纪录", "CCTV-9纪录", "中央九台", "中央9台"}},
	{Canonical: "CCTV-10", Matches: []string{"CCTV10", "CCTV10科教", "CCTV-10 科教", "CCTV-10科教", "中央十台", "中央10台"}},
	{Canonical: "CCTV-11", Matches: []string{"CCTV11", "CCTV11戏曲", "CCTV-11 戏曲", "CCTV-11戏曲", "中央十一台", "中央11台"}},
	{Canonical: "CCTV-12", Matches: []string{"CCTV12", "CCTV12社会与法", "CCTV-12 社会与法", "CCTV-12社会与法", "中央十二台", "中央12台"}},
	{Canonical: "CCTV-13", Matches: []string{"CCTV13", "CCTV13新闻", "CCTV-13 新闻", "CCTV-13新闻", "中央十三台", "中央13台", "新闻频道"}},
	{Canonical: "CCTV-14", Matches: []string{"CCTV14", "CCTV14少儿", "CCTV-14 少儿", "CCTV-14少儿", "中央十四台", "中央14台"}},
	{Canonical: "CCTV-15", Matches: []string{"CCTV15", "CCTV15音乐", "CCTV-15 音乐", "CCTV-15音乐", "中央十五台", "中央15台"}},
	{Canonical: "CCTV-16", Matches: []string{"CCTV16", "CCTV16奥林匹克", "CCTV-16 奥林匹克", "CCTV-16奥林匹克", "奥林匹克频道"}},
	{Canonical: "CCTV-17", Matches: []string{"CCTV17", "CCTV17农业农村", "CCTV-17 农业农村", "CCTV-17农业农村"}},
	{Canonical: "湖南卫视", Matches: []string{"湖南高清", "湖南卫视 HD", "湖南卫视高清"}},
	{Canonical: "浙江卫视", Matches: []string{"浙江高清", "浙江卫视 HD", "浙江卫视高清"}},
	{Canonical: "江苏卫视", Matches: []string{"江苏高清", "江苏卫视 HD", "江苏卫视高清"}},
	{Canonical: "东方卫视", Matches: []string{"东方高清", "东方卫视 HD", "东方卫视高清"}},
	{Canonical: "北京卫视", Matches: []string{"北京高清", "北京卫视 HD", "北京卫视高清"}},
	{Canonical: "广东卫视", Matches: []string{"广东高清", "广东卫视 HD", "广东卫视高清"}},
	{Canonical: "深圳卫视", Matches: []string{"深圳高清", "深圳卫视 HD", "深圳卫视高清"}},
}

// Engine 负责频道名称规范化与 EPG/台标自动补齐
type Engine struct {
	cleanerOpt    config.CleanerOpt
	epgMappingOpt config.EPGMappingOpt
	stripRegexes  []*regexp.Regexp
	aliasMap      map[string]string // key 为清洗后规范化名称, val 为 Canonical
}

// NewEngine 创建清洗与映射引擎
func NewEngine(cleanerOpt config.CleanerOpt, epgMappingOpt config.EPGMappingOpt) *Engine {
	e := &Engine{
		cleanerOpt:    cleanerOpt,
		epgMappingOpt: epgMappingOpt,
		aliasMap:      make(map[string]string),
	}

	patterns := cleanerOpt.StripPatterns
	if len(patterns) == 0 {
		patterns = defaultStripPatterns
	}

	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			e.stripRegexes = append(e.stripRegexes, re)
		}
	}

	registerAlias := func(rule config.AliasRule) {
		for _, m := range rule.Matches {
			e.aliasMap[normalizeKey(m)] = rule.Canonical
			stripped := m
			for _, re := range e.stripRegexes {
				stripped = re.ReplaceAllString(stripped, "")
			}
			stripped = strings.TrimSpace(stripped)
			stripped = strings.Trim(stripped, "-_ []()")
			if stripped != "" {
				e.aliasMap[normalizeKey(stripped)] = rule.Canonical
			}
		}
		e.aliasMap[normalizeKey(rule.Canonical)] = rule.Canonical
	}

	// 注册默认别名规则
	for _, rule := range defaultAliases {
		registerAlias(rule)
	}

	// 用户自定义别名覆盖
	for _, rule := range cleanerOpt.Aliases {
		registerAlias(rule)
	}

	return e
}

// CleanName 清洗频道名称
func (e *Engine) CleanName(rawName string) string {
	if !e.cleanerOpt.Enable {
		return rawName
	}

	if canonical, ok := e.aliasMap[normalizeKey(rawName)]; ok {
		return canonical
	}

	name := rawName
	for _, re := range e.stripRegexes {
		name = re.ReplaceAllString(name, "")
	}

	name = strings.TrimSpace(name)
	name = strings.Trim(name, "-_ []()")

	if canonical, ok := e.aliasMap[normalizeKey(name)]; ok {
		return canonical
	}

	if name == "" {
		return rawName
	}

	return name
}

// ProcessTrack 对 Track 进行频道名清洗、tvg-name、tvg-id 及 tvg-logo 自动补齐
func (e *Engine) ProcessTrack(track *m3u.Track) {
	if track == nil {
		return
	}

	cleanedName := track.Name
	if e.cleanerOpt.Enable {
		cleanedName = e.CleanName(track.Name)
		track.Name = cleanedName
	}

	if track.GetTag("tvg-name") == "" && cleanedName != "" {
		track.SetTag("tvg-name", cleanedName)
	}

	if e.epgMappingOpt.Enable {
		// 优先使用明确映射，未配置映射时补齐缺失的 tvg-id
		if id, ok := e.epgMappingOpt.Mappings[cleanedName]; ok && id != "" {
			track.SetTag("tvg-id", id)
		} else if track.GetTag("tvg-id") == "" {
			track.SetTag("tvg-id", cleanedName)
		}
		// 补齐 tvg-logo
		if track.GetTag("tvg-logo") == "" && e.epgMappingOpt.LogoURLTemplate != "" {
			logoURL := strings.ReplaceAll(e.epgMappingOpt.LogoURLTemplate, "{name}", cleanedName)
			track.SetTag("tvg-logo", logoURL)
		}
	}
}

func normalizeKey(s string) string {
	s = strings.ToUpper(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

package mixer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"

	"github.com/wayjam/tv-mixproxy/config"
)

func compileRegex(pattern string) *regexp.Regexp {
	if pattern == "" {
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		// 如果正则表达式无效，返回nil
		return nil
	}
	return re
}

func matchFilter(value string, includeRegex, excludeRegex *regexp.Regexp) bool {
	if includeRegex != nil && !includeRegex.MatchString(value) {
		return false
	}
	if excludeRegex != nil && excludeRegex.MatchString(value) {
		return false
	}
	return true
}

var nullHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

func NewMixURLHandler(
	mixOpt config.MixOpt, sourcer Sourcer,
) (http.Handler, error) {
	if mixOpt.Disabled || mixOpt.SourceName == "" {
		return nullHandler, nil
	}

	link, source, err := mixFieldAndGetSource(mixOpt, sourcer)
	if err != nil {
		return nullHandler, fmt.Errorf("mixing url: %w", err)
	}

	if source.Type() != config.SourceTypeTvBoxSingle {
		return nullHandler, fmt.Errorf("source %s should be a single source", mixOpt.SourceName)
	}

	if link == "" {
		return nullHandler, nil
	}

	// 移除 URL 中可能存在的校验信息
	link = strings.Split(link, ";")[0]
	// 如果是相对路径，则返回一个 proxy 处理器，该处理器将请求转发到相对路径
	link = fullFillURL(link, source)

	targetURL, err := url.Parse(link)
	if err != nil {
		return nullHandler, fmt.Errorf("parsing target url: %w", err)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Host = targetURL.Host
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path, req.URL.RawPath = targetURL.Path, targetURL.RawPath
		},
	}

	// 可以自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, err.Error(), http.StatusBadGateway)
	}

	return proxy, nil
}

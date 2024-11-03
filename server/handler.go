package server

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"image/png"
	"net/http"
	"strconv"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/imageutil"
	"github.com/wayjam/tv-mixproxy/pkg/m3u"
	"github.com/wayjam/tv-mixproxy/pkg/mixer"
)

func Home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, TV MixProxy 📺!"))
}

func Logo(w http.ResponseWriter, r *http.Request) {
	svgLogo := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
		<rect x="10" y="20" width="80" height="60" rx="5" ry="5" fill="#333"/>
		<rect x="15" y="25" width="70" height="50" rx="3" ry="3" fill="#4CAF50"/>
		<circle cx="50" cy="75" r="5" fill="#333"/>
		<line x1="30" y1="15" x2="40" y2="5" stroke="#333" stroke-width="2"/>
		<line x1="70" y1="15" x2="60" y2="5" stroke="#333" stroke-width="2"/>
	</svg>`
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write([]byte(svgLogo))
}

func Wallpaper(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Parse query parameters
	width, _ := strconv.Atoi(query.Get("width"))
	if width == 0 {
		width = 800
	}
	height, _ := strconv.Atoi(query.Get("height"))
	if height == 0 {
		height = 600
	}
	pattern := query.Get("pattern")
	if pattern == "" {
		pattern = "solid"
	}
	opacity, _ := strconv.ParseFloat(query.Get("opacity"), 64)
	if opacity == 0 {
		opacity = 1.0
	}
	borderWidth, _ := strconv.Atoi(query.Get("border_width"))

	// Parse colors
	bgColor := imageutil.ParseColor(query.Get("bg_color"))
	if bgColor == nil {
		bgColor = imageutil.ParseColor("FFFFFF")
	}
	borderColor := imageutil.ParseColor(query.Get("border_color"))
	if borderColor == nil {
		borderColor = imageutil.ParseColor("000000")
	}

	params := imageutil.ImageParams{
		BackgroundColor: bgColor,
		Width:           width,
		Height:          height,
		Pattern:         pattern,
		Opacity:         opacity,
		BorderWidth:     borderWidth,
		BorderColor:     borderColor,
	}

	img := imageutil.GenerateImage(params)
	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, img)
}

func NewRepoHandler(cfg *config.Config, sourceManager *mixer.SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.TvBoxSingleRepoOpt.Disable {
			http.Error(w, "SingleRepo is disabled", http.StatusNotImplemented)
			return
		}

		result, err := mixer.MixTvBoxRepo(cfg, sourceManager)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func NewMultiRepoHandler(cfg *config.Config, sourceManager *mixer.SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.TvBoxMultiRepoOpt.Disable {
			http.Error(w, "MultiRepo is disabled", http.StatusNotImplemented)
			return
		}

		result, err := mixer.MixMultiRepo(cfg, sourceManager)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func NewSpiderHandler(cfg *config.Config, sourceManager *mixer.SourceManager) http.HandlerFunc {
	handler, err := mixer.NewMixURLHandler(cfg.TvBoxSingleRepoOpt.Spider, sourceManager)

	return func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		handler.ServeHTTP(w, r)
	}
}

func NewEPGHandler(cfg *config.Config, sourceManager *mixer.SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.EPGOpt.Disable {
			http.Error(w, "EPG is disabled", http.StatusNotImplemented)
			return
		}

		result, err := mixer.MixEPG(cfg, sourceManager)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		format := r.URL.Query().Get("format")
		if format == "" {
			format = "xml"
		}

		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Encoding", "gzip")

		if format == "gz" {
			w.Header().Set("Content-Disposition", "attachment; filename=epg.xml.gz")
		}

		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()

		encoder := xml.NewEncoder(gzipWriter)
		if err := encoder.Encode(result); err != nil {
			http.Error(w, "Error encoding and compressing XML", http.StatusInternalServerError)
			return
		}
	}
}

func NewM3UMediaHandler(cfg *config.Config, sourceManager *mixer.SourceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.M3UOpt.Disable {
			http.Error(w, "M3U is disabled", http.StatusNotImplemented)
			return
		}

		result, err := mixer.MixM3UMediaPlayList(cfg, sourceManager)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		encoder := m3u.NewEncoder(w)
		if err := encoder.Encode(result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

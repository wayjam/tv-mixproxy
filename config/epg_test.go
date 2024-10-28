package config

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

const epgData = `
<tv>
	<channel id="example.com">
		<display-name lang="en">Example Channel</display-name>
		<url>http://example.com</url>
	</channel>
	<programme channel="example.com" start="20230101000000 +0000" stop="20230101010000 +0000">
		<title lang="en">Example Show</title>
		<desc lang="en">This is an example show.</desc>
	</programme>
</tv>
`

func TestLoadEPGData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<tv></tv>"))
	}))
	defer server.Close()

	data, err := LoadEPGData(server.URL)
	if err != nil {
		t.Fatalf("LoadEPGData failed: %v", err)
	}

	if string(data) != "<tv></tv>" {
		t.Errorf("Unexpected data: %s", string(data))
	}
}

func TestParseEPGConfig(t *testing.T) {
	reader := bytes.NewReader([]byte(epgData))
	epg, err := ParseEPGConfig(reader)
	if err != nil {
		t.Fatalf("ParseEPGConfig failed: %v", err)
	}

	if len(epg.Channel) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(epg.Channel))
	}
	if epg.Channel[0].ID != "example.com" {
		t.Errorf("Expected channel ID 'example.com', got '%s'", epg.Channel[0].ID)
	}

	if len(epg.Programme) != 1 {
		t.Errorf("Expected 1 programme, got %d", len(epg.Programme))
	}
	if epg.Programme[0].Channel != "example.com" {
		t.Errorf("Expected programme channel 'example.com', got '%s'", epg.Programme[0].Channel)
	}
}

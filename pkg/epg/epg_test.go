package epg

import (
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

func TestUnmarshal(t *testing.T) {
	epg, err := Unmarshal([]byte(epgData))
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
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

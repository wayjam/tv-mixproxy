package mixer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"github.com/wayjam/tvbox-mixproxy/config"
)

// MockSourcer 是一个模拟的 Sourcer 实现
type MockSourcer struct {
	sources map[string]*Source
}

func (m *MockSourcer) GetSource(name string) (*Source, error) {
	source, ok := m.sources[name]
	if !ok {
		return nil, fmt.Errorf("source not found: %s", name)
	}
	return source, nil
}

func TestMixRepo(t *testing.T) {
	mockSourcer := &MockSourcer{
		sources: map[string]*Source{
			"source1": {
				data: []byte(`{"spider":"spider1","wallpaper":"wall1","logo":"logo1","sites":[{"key":"site1","name":"Site 1"}],"doh":[{"name":"doh1"}],"lives":[{"name":"live1"}]}`),
			},
			"source2": {
				data: []byte(`{"spider":"spider2","wallpaper":"wall2","logo":"logo2","sites":[{"key":"site2","name":"Site 2"}],"doh":[{"name":"doh2"}],"lives":[{"name":"live2"}]}`),
			},
		},
	}

	cfg := &config.Config{
		SingleRepoOpt: config.SingleRepoOpt{
			Spider:    config.MixOpt{SourceName: "source1", Field: "spider"},
			Wallpaper: config.MixOpt{SourceName: "source2", Field: "wallpaper"},
			Logo:      config.MixOpt{}, // Empty source, should not be mixed
			Sites:     config.ArrayMixOpt{MixOpt: config.MixOpt{SourceName: "source1", Field: "sites"}},
			DOH:       config.ArrayMixOpt{MixOpt: config.MixOpt{}}, // Empty source, should not be mixed
			Lives:     config.ArrayMixOpt{MixOpt: config.MixOpt{SourceName: "source1", Field: "lives"}},
		},
	}

	result, err := MixRepo(cfg, mockSourcer)
	assert.NoError(t, err)
	assert.Equal(t, "spider1", result.Spider)
	assert.Equal(t, "wall2", result.Wallpaper)
	assert.Equal(t, result.Logo, "http://localhost:0/logo")
	assert.Len(t, result.Sites, 1)
	assert.Equal(t, "Site 1", result.Sites[0].Name)
	assert.Empty(t, result.DOH) // Should be empty as it was not mixed
	assert.Len(t, result.Lives, 1)
	assert.Equal(t, "live1", result.Lives[0].Name)
}

func TestMixField(t *testing.T) {
	mockSourcer := &MockSourcer{
		sources: map[string]*Source{
			"source1": {
				data: []byte(`{"field1":"value1","field2":"value2"}`),
			},
		},
	}

	result, err := mixField(config.MixOpt{SourceName: "source1", Field: "field1"}, mockSourcer)
	assert.NoError(t, err)
	assert.Equal(t, "value1", result)

	_, err = mixField(config.MixOpt{SourceName: "source1", Field: "non_existent"}, mockSourcer)
	assert.Error(t, err)
}

func TestMixArrayField(t *testing.T) {
	mockSourcer := &MockSourcer{
		sources: map[string]*Source{
			"source1": {
				data: []byte(`{"array":[{"key":"item1","value":"value1"},{"key":"item2","value":"value2"}]}`),
			},
		},
	}

	type TestItem struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	result, err := mixArrayField[TestItem](config.ArrayMixOpt{
		MixOpt:   config.MixOpt{SourceName: "source1", Field: "array"},
		FilterBy: "key",
		Include:  "item1",
	}, mockSourcer)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "item1", result[0].Key)
	assert.Equal(t, "value1", result[0].Value)
}

func TestFilterArray(t *testing.T) {
	array := []gjson.Result{
		gjson.Parse(`{"key":"item1","value":"value1"}`),
		gjson.Parse(`{"key":"item2","value":"value2"}`),
		gjson.Parse(`{"key":"item3","value":"value3"}`),
	}

	result, err := filterArray(array, config.ArrayMixOpt{
		FilterBy: "key",
		Include:  "item[12]",
		Exclude:  "item2",
	})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "item1", result[0].Get("key").String())

	// Test with empty include and exclude
	result, err = filterArray(array, config.ArrayMixOpt{
		FilterBy: "key",
	})

	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestMixMultiRepo(t *testing.T) {
	mockSourcer := &MockSourcer{
		sources: map[string]*Source{
			"multi_source": {
				data: []byte(`{"urls":[
					{"url":"http://example1.com","name":"Repo 1"},
					{"url":"http://example2.com","name":"Repo 2"},
					{"url":"http://example3.com","name":"Repo 3"}
				]}`),
			},
			"single_source": {
				data: []byte(`{"spider":"spider1","wallpaper":"wall1","logo":"logo1","sites":[{"key":"site1","name":"Site 1"}],"doh":[{"name":"doh1"}],"lives":[{"name":"live1"}]}`),
			},
		},
	}

	// Test case 1: Without filtering
	cfg := &config.Config{
		MultiRepoOpt: config.MultiRepoOpt{
			IncludeSingleRepo: true,
			Repos: []config.ArrayMixOpt{
				{
					MixOpt: config.MixOpt{
						SourceName: "multi_source",
					},
				},
			},
		},
		SingleRepoOpt: config.SingleRepoOpt{
			Spider: config.MixOpt{SourceName: "single_source", Field: "spider"},
		},
	}

	cfg.Fixture()

	result, err := MixMultiRepo(cfg, mockSourcer)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.Len(t, result.Repos, 4) // 3 from multi_source + 1 single repo
	assert.Equal(t, "TvBox MixProxy", result.Repos[0].Name)
	assert.Contains(t, result.Repos[0].URL, "/repo")
	assert.Equal(t, "Repo 1", result.Repos[1].Name)
	assert.Equal(t, "Repo 2", result.Repos[2].Name)
	assert.Equal(t, "Repo 3", result.Repos[3].Name)

	// Test case 2: With filtering
	cfg.MultiRepoOpt.Repos[0].Include = "Repo [12]"
	cfg.MultiRepoOpt.Repos[0].Exclude = "Repo 2"

	filteredResult, err := MixMultiRepo(cfg, mockSourcer)
	assert.NoError(t, err)
	assert.NotNil(t, filteredResult)
	assert.Len(t, filteredResult.Repos, 2) // 1 filtered from multi_source + 1 single repo
	assert.Equal(t, "TvBox MixProxy", filteredResult.Repos[0].Name)
	assert.Contains(t, filteredResult.Repos[0].URL, "/repo")
	assert.Equal(t, "Repo 1", filteredResult.Repos[1].Name)
}

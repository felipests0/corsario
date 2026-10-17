package docker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchHubAndGetImageTags(t *testing.T) {
	var requestedTagPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/search/repositories":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"results":[{"repo_name":"redis","short_description":"Redis key-value store","star_count":12500,"is_official":true}]}`)
		case "/v2/repositories/library/postgres/tags":
			requestedTagPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"results":[{"name":"15-alpine"},{"name":"16-alpine"},{"name":"latest"}]}`)
		case "/v2/repositories/zabbix/zabbix-appliance/tags":
			requestedTagPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"results":[{"name":"alpine-latest"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	origURL := hubBaseURL
	hubBaseURL = srv.URL
	defer func() {
		hubBaseURL = origURL
	}()

	results, err := SearchHub("redis")
	if err != nil {
		t.Fatalf("erro inesperado em SearchHub: %v", err)
	}
	if len(results) != 1 || results[0].Name != "redis" || results[0].StarCount != 12500 {
		t.Errorf("resultado inesperado em SearchHub: %+v", results)
	}

	tags, err := GetImageTags("postgres:15-alpine")
	if err != nil {
		t.Fatalf("erro inesperado em GetImageTags (library): %v", err)
	}
	if requestedTagPath != "/v2/repositories/library/postgres/tags" {
		t.Errorf("esperava path '/v2/repositories/library/postgres/tags', obteve '%s'", requestedTagPath)
	}
	if len(tags) != 3 || tags[0] != "15-alpine" {
		t.Errorf("tags inesperadas: %+v", tags)
	}

	nsTags, err := GetImageTags("zabbix/zabbix-appliance")
	if err != nil {
		t.Fatalf("erro inesperado em GetImageTags (namespace): %v", err)
	}
	if requestedTagPath != "/v2/repositories/zabbix/zabbix-appliance/tags" {
		t.Errorf("esperava path '/v2/repositories/zabbix/zabbix-appliance/tags', obteve '%s'", requestedTagPath)
	}
	if len(nsTags) != 1 || nsTags[0] != "alpine-latest" {
		t.Errorf("tags inesperadas para namespace: %+v", nsTags)
	}
}

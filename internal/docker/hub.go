package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var hubBaseURL = "https://hub.docker.com"

var hubHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

type HubSearchResult struct {
	Name        string `json:"repo_name"`
	Description string `json:"short_description"`
	StarCount   int    `json:"star_count"`
	IsOfficial  bool   `json:"is_official"`
}

type hubSearchResponse struct {
	Results []HubSearchResult `json:"results"`
}

type hubTagItem struct {
	Name string `json:"name"`
}

type hubTagsResponse struct {
	Results []hubTagItem `json:"results"`
}

func SearchHub(query string) ([]HubSearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []HubSearchResult{}, nil
	}

	endpoint := fmt.Sprintf("%s/v2/search/repositories?query=%s&page_size=12", hubBaseURL, url.QueryEscape(q))
	resp, err := hubHTTPClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro ao pesquisar no Docker Hub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker Hub retornou status %d", resp.StatusCode)
	}

	var payload hubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta do Docker Hub: %w", err)
	}

	return payload.Results, nil
}

func GetImageTags(image string) ([]string, error) {
	img := strings.TrimSpace(image)
	if img == "" {
		return nil, fmt.Errorf("nome da imagem vazio")
	}

	// Remove eventual sufixo :tag (ex: postgres:15-alpine -> postgres)
	if idx := strings.Index(img, ":"); idx != -1 {
		img = img[:idx]
	}

	namespace := "library"
	repo := img
	if strings.Contains(img, "/") {
		parts := strings.SplitN(img, "/", 2)
		namespace = parts[0]
		repo = parts[1]
	}

	endpoint := fmt.Sprintf("%s/v2/repositories/%s/%s/tags?page_size=25", hubBaseURL, url.PathEscape(namespace), url.PathEscape(repo))
	resp, err := hubHTTPClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar tags no Docker Hub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker Hub tags retornou status %d", resp.StatusCode)
	}

	var payload hubTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("erro ao decodificar tags do Docker Hub: %w", err)
	}

	tags := make([]string, 0, len(payload.Results))
	for _, t := range payload.Results {
		if t.Name != "" {
			tags = append(tags, t.Name)
		}
	}

	return tags, nil
}

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Saurabh825/gfg-cli/internal/config"

	"github.com/PuerkitoBio/goquery"
)

func GetHeaders() map[string]string {
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64; rv:148.0) Gecko/20100101 Firefox/148.0",
		"Accept":          "*/*",
		"Accept-Language": "en-US,en;q=0.5",
		"Referer":         "https://www.geeksforgeeks.org/",
		"Origin":          "https://www.geeksforgeeks.org",
		"Connection":      "keep-alive",
	}

	if config.Cfg.CookieString != "" {
		headers["Cookie"] = config.Cfg.CookieString
	} else {
		var cookieParts []string
		if config.Cfg.SessionID != "" {
			cookieParts = append(cookieParts, "sessionid="+config.Cfg.SessionID)
		}
		if config.Cfg.GfgUserName != "" {
			cookieParts = append(cookieParts, "gfguserName="+config.Cfg.GfgUserName)
		}
		if len(cookieParts) > 0 {
			headers["Cookie"] = strings.Join(cookieParts, "; ")
		}
	}
	return headers
}

func FetchProblemDetails(slug string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://www.geeksforgeeks.org/problems/%s/1", slug)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range GetHeaders() {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error fetching problem: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	scriptContent := doc.Find("script#__NEXT_DATA__").Text()
	if scriptContent == "" {
		return nil, fmt.Errorf("could not find problem data in HTML")
	}

	var rawJSON map[string]interface{}
	if err := json.Unmarshal([]byte(scriptContent), &rawJSON); err != nil {
		return nil, err
	}

	props, ok := rawJSON["props"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing props")
	}
	pageProps, ok := props["pageProps"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing pageProps")
	}
	initialState, ok := pageProps["initialState"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing initialState")
	}
	problemApi, ok := initialState["problemApi"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing problemApi")
	}
	queries, ok := problemApi["queries"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing queries")
	}

	for key, val := range queries {
		if strings.Contains(key, "getProblemDetails") {
			vMap, ok := val.(map[string]interface{})
			if ok {
				if data, ok := vMap["data"].(map[string]interface{}); ok {
					return data, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("problem details not found")
}

func FetchMetaInfo(slug string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://practiceapi.geeksforgeeks.org/api/latest/problems/%s/metainfo/?page=1&sortBy=submissions", slug)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range GetHeaders() {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error fetching meta info: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, nil
}

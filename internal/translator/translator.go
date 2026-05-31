package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Translator は文字列を指定言語に翻訳します。
type Translator interface {
	Translate(text, targetLang string) (string, error)
}

type azureTranslator struct {
	httpClient *http.Client
	endpoint   string
	region     string
	key        string
}

// NewAzureTranslator は Azure AI Translator v3.0 を叩く Translator を返します。
func NewAzureTranslator(endpoint, region, key string) Translator {
	return &azureTranslator{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		endpoint:   endpoint,
		region:     region,
		key:        key,
	}
}

type translateRequestItem struct {
	Text string `json:"Text"`
}

type translateResponseItem struct {
	Translations []struct {
		Text string `json:"text"`
		To   string `json:"to"`
	} `json:"translations"`
}

func (t *azureTranslator) Translate(text, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	payload, err := json.Marshal([]translateRequestItem{{Text: text}})
	if err != nil {
		return "", fmt.Errorf("failed to marshal translator request: %w", err)
	}

	q := url.Values{}
	q.Set("api-version", "3.0")
	q.Set("to", targetLang)
	reqURL := t.endpoint + "/translate?" + q.Encode()

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create translator request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Ocp-Apim-Subscription-Key", t.key)
	req.Header.Set("Ocp-Apim-Subscription-Region", t.region)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute translator request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read translator response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("translator returned status %d: %s", resp.StatusCode, string(body))
	}

	var items []translateResponseItem
	if err := json.Unmarshal(body, &items); err != nil {
		return "", fmt.Errorf("failed to decode translator response: %w", err)
	}
	if len(items) == 0 || len(items[0].Translations) == 0 || items[0].Translations[0].Text == "" {
		return "", fmt.Errorf("translator response missing translation: %s", string(body))
	}
	return items[0].Translations[0].Text, nil
}

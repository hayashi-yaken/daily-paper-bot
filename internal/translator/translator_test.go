package translator

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAzureTranslator_Translate(t *testing.T) {
	t.Run("success returns translation", func(t *testing.T) {
		var receivedAuthKey, receivedRegion, receivedQuery string
		var receivedBody []byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuthKey = r.Header.Get("Ocp-Apim-Subscription-Key")
			receivedRegion = r.Header.Get("Ocp-Apim-Subscription-Region")
			receivedQuery = r.URL.RawQuery
			receivedBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"translations":[{"text":"こんにちは","to":"ja"}]}]`))
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "secret-key")
		got, err := tr.Translate("hello", "ja")
		if err != nil {
			t.Fatalf("Translate failed: %v", err)
		}
		if got != "こんにちは" {
			t.Errorf("expected translation 'こんにちは', got %q", got)
		}
		if receivedAuthKey != "secret-key" {
			t.Errorf("expected Ocp-Apim-Subscription-Key=secret-key, got %q", receivedAuthKey)
		}
		if receivedRegion != "japaneast" {
			t.Errorf("expected Ocp-Apim-Subscription-Region=japaneast, got %q", receivedRegion)
		}
		if !strings.Contains(receivedQuery, "api-version=3.0") || !strings.Contains(receivedQuery, "to=ja") {
			t.Errorf("expected query to contain api-version=3.0 and to=ja, got %q", receivedQuery)
		}
		var bodyPayload []struct{ Text string }
		if err := json.Unmarshal(receivedBody, &bodyPayload); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if len(bodyPayload) != 1 || bodyPayload[0].Text != "hello" {
			t.Errorf("expected body [{Text:hello}], got %+v", bodyPayload)
		}
	})

	t.Run("empty text skips API call", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "k")
		got, err := tr.Translate("", "ja")
		if err != nil {
			t.Fatalf("Translate returned error for empty input: %v", err)
		}
		if got != "" {
			t.Errorf("expected empty translation, got %q", got)
		}
		if callCount != 0 {
			t.Errorf("expected API not to be called, but got %d call(s)", callCount)
		}
	})

	t.Run("non-2xx returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":401000,"message":"invalid key"}}`))
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "k")
		_, err := tr.Translate("hello", "ja")
		if err == nil {
			t.Fatal("expected error for 401 response")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("expected error to mention status 401, got %v", err)
		}
	})

	t.Run("malformed response returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "k")
		_, err := tr.Translate("hello", "ja")
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})

	t.Run("empty translations array returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`[]`))
		}))
		defer server.Close()

		tr := NewAzureTranslator(server.URL, "japaneast", "k")
		_, err := tr.Translate("hello", "ja")
		if err == nil {
			t.Fatal("expected error for empty top-level array")
		}
	})
}

package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectStripsInjectedHeaders(t *testing.T) {
	var gotSecret string
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Secret")
		w.WriteHeader(200)
	}))
	defer dest.Close()
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, dest.URL, http.StatusFound)
	}))
	defer src.Close()

	injected := http.Header{"X-Secret": {"shh"}}
	client, err := newHTTPClient(&downloadFlags{}, injected)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, src.URL, nil)
	req.Header.Set("X-Secret", "shh")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotSecret != "" {
		t.Fatalf("secret header leaked across host redirect: %q", gotSecret)
	}
}

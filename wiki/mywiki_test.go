package wiki

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTitleValid(t *testing.T) {
	tests := []struct {
		path      string
		wantTitle string
	}{
		{"/view/TestPage", "TestPage"},
		{"/edit/MyPage", "MyPage"},
		{"/save/AnotherPage", "AnotherPage"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		w := httptest.NewRecorder()

		title, err := getTitle(w, req)
		if err != nil {
			t.Errorf("getTitle(%q) unexpected error: %v", tt.path, err)
		}
		if title != tt.wantTitle {
			t.Errorf("getTitle(%q) = %q, want %q", tt.path, title, tt.wantTitle)
		}
	}
}

func TestGetTitleInvalid(t *testing.T) {
	invalidPaths := []string{
		"/",
		"/invalid/",
		"/view/",
		"/view/has/extra",
	}
	for _, path := range invalidPaths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()

		_, err := getTitle(w, req)
		if err == nil {
			t.Errorf("getTitle(%q) expected error, got nil", path)
		}
	}
}

func TestMakeHandlerRedirectsInvalidPath(t *testing.T) {
	handler := makeHandler(func(w http.ResponseWriter, r *http.Request, title string) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/invalid/path", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/view/Home" {
		t.Errorf("expected redirect to /view/Home, got %q", loc)
	}
}

func TestMakeHandlerCallsFnWithTitle(t *testing.T) {
	called := false
	handler := makeHandler(func(w http.ResponseWriter, r *http.Request, title string) {
		called = true
		if title != "TestPage" {
			t.Errorf("expected title 'TestPage', got %q", title)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/view/TestPage", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Error("inner handler was not called for valid path")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestValidPath(t *testing.T) {
	valid := []string{
		"/view/Hello",
		"/edit/MyPage123",
		"/save/ABC",
	}
	for _, p := range valid {
		if validPath.FindStringSubmatch(p) == nil {
			t.Errorf("expected %q to match validPath", p)
		}
	}

	invalid := []string{
		"/view/",
		"/delete/Page",
		"/view/has spaces",
		"/view/has/slash",
	}
	for _, p := range invalid {
		if validPath.FindStringSubmatch(p) != nil {
			t.Errorf("expected %q to NOT match validPath", p)
		}
	}
}

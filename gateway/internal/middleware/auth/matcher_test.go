package auth

import "testing"

func TestMatches(t *testing.T) {
	tests := []struct {
		pattern, path string
		want          bool
	}{
		{pattern: "/user/api/v1/login", path: "/user/api/v1/login", want: true},
		{pattern: "/public/*", path: "/public/assets/app.js", want: true},
		{pattern: "*.css", path: "/assets/app.css", want: true},
		{pattern: "/public/*.css", path: "/public/assets/app.css", want: true},
		{pattern: "*", path: "/anything", want: true},
		{pattern: "/public/*.css", path: "/private/app.css", want: false},
	}
	for _, test := range tests {
		if got := matches(test.pattern, test.path); got != test.want {
			t.Errorf("matches(%q, %q) = %v, want %v", test.pattern, test.path, got, test.want)
		}
	}
}

func TestBearerToken(t *testing.T) {
	if value, ok := bearerToken("Bearer token-value"); !ok || value != "token-value" {
		t.Fatalf("bearerToken() = %q, %v", value, ok)
	}
	for _, value := range []string{"", "token-value", "Basic token-value", "Bearer"} {
		if _, ok := bearerToken(value); ok {
			t.Errorf("bearerToken(%q) unexpectedly succeeded", value)
		}
	}
}

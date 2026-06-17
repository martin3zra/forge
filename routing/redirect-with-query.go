package routing

import (
	"net/http"
	"net/url"
)

func RedirectWithQuery(w http.ResponseWriter, r *http.Request, basePath string, params map[string]string) {
	// Parse the base URL
	targetURL, err := url.Parse(basePath)
	if err != nil {
		http.Error(w, "Invalid redirect URL", http.StatusInternalServerError)
		return
	}

	// Add query parameters
	query := targetURL.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	targetURL.RawQuery = query.Encode()

	// Redirect
	http.Redirect(w, r, targetURL.String(), http.StatusFound)
}

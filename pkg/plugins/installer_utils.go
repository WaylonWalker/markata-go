package plugins

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// getLatestGitHubReleaseVersion fetches the latest release version tag from a GitHub repo
// by following the /releases/latest redirect.
func getLatestGitHubReleaseVersion(baseURL string) (string, error) {
	currentURL := baseURL + "/latest"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	const maxRedirects = 10
	for redirectCount := 0; redirectCount < maxRedirects; redirectCount++ {
		req, err := http.NewRequestWithContext(ctx, "HEAD", currentURL, http.NoBody)
		if err != nil {
			return "", err
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		resp.Body.Close()

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := resp.Header.Get("Location")
			if location == "" {
				return "", err
			}

			parts := strings.Split(location, "/")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				if strings.HasPrefix(lastPart, "v") && strings.Contains(lastPart, ".") {
					return lastPart, nil
				}
			}

			currentURL = location
			continue
		}

		parts := strings.Split(currentURL, "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			if strings.HasPrefix(lastPart, "v") && strings.Contains(lastPart, ".") {
				return lastPart, nil
			}
		}

		return "", nil
	}

	return "", nil
}

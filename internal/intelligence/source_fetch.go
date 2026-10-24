package intelligence

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func sourceClient() *http.Client {
	return &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 4 {
				return errors.New("too many source redirects")
			}
			if _, err := validateSourceURL(req.URL.String()); err != nil {
				return fmt.Errorf("unsafe source redirect: %w", err)
			}
			if len(via) > 0 && !strings.EqualFold(req.URL.Hostname(), via[0].URL.Hostname()) {
				return errors.New("cross-host source redirect rejected")
			}
			return nil
		},
	}
}

func fetchSource(ctx context.Context, client *http.Client, source Source) ([]byte, string, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, "", "", false, err
	}
	req.Header.Set("Accept", "text/markdown,text/plain,application/json,text/html;q=0.5")
	req.Header.Set("User-Agent", "personal-ai-brain/0.2")
	if source.ETag != nil {
		req.Header.Set("If-None-Match", *source.ETag)
	}
	if source.LastModified != nil {
		req.Header.Set("If-Modified-Since", *source.LastModified)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, "", "", false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotModified {
		return nil, response.Header.Get("ETag"), response.Header.Get("Last-Modified"), false, nil
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, "", "", false, fmt.Errorf("source returned HTTP %d", response.StatusCode)
	}
	reader := io.LimitReader(response.Body, maxSourceBytes+1)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", "", false, err
	}
	if len(body) > maxSourceBytes {
		return nil, "", "", false, errors.New("source exceeds 2 MiB limit")
	}
	return body, response.Header.Get("ETag"), response.Header.Get("Last-Modified"), true, nil
}

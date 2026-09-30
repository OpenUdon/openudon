package elicitor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/registry"
	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/netpolicy"
)

type browserRegistryLookupOptions struct {
	Location         string
	Query            string
	At               time.Time
	Remote           bool
	HTTPClient       *http.Client
	AllowUnsafeHosts bool
	Timeout          time.Duration
}

type browserRegistryPullAttempt struct {
	Match  registry.SearchResult
	Pulled registry.PullResult
	Err    error
}

func browserRegistryLookup(ctx context.Context, opts browserRegistryLookupOptions) (registry.SearchReport, []browserRegistryPullAttempt, error) {
	if !opts.Remote {
		client := localBrowserRegistryClient(opts.Timeout)
		search, err := client.Search(ctx, registry.SearchOptions{
			Location: opts.Location, Query: opts.Query, Limit: registry.DefaultMaxResults, At: opts.At,
		})
		if err != nil {
			return registry.SearchReport{}, nil, err
		}
		return pullBrowserRegistryMatches(ctx, client, opts.Location, opts.At, search, nil)
	}
	return remoteBrowserRegistryLookup(ctx, opts)
}

func remoteBrowserRegistryLookup(ctx context.Context, opts browserRegistryLookupOptions) (registry.SearchReport, []browserRegistryPullAttempt, error) {
	base, err := remoteBrowserRegistryBase(opts.Location, opts.AllowUnsafeHosts)
	if err != nil {
		return registry.SearchReport{}, nil, err
	}
	if opts.HTTPClient == nil {
		return registry.SearchReport{}, nil, fmt.Errorf("remote browser registry requires the DNS-pinned HTTP client")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = registry.DefaultTimeout
	}
	bounded, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	mirror, err := os.MkdirTemp("", "openudon-browser-registry.")
	if err != nil {
		return registry.SearchReport{}, nil, err
	}
	defer os.RemoveAll(mirror)

	indexTarget := base.ResolveReference(&url.URL{Path: registry.IndexName})
	indexData, finalIndex, err := downloadBrowserRegistryFile(bounded, opts.HTTPClient, indexTarget, registry.DefaultMaxBytes, opts.AllowUnsafeHosts)
	if err != nil {
		return registry.SearchReport{}, nil, err
	}
	if filepath.Base(finalIndex.Path) != registry.IndexName {
		return registry.SearchReport{}, nil, fmt.Errorf("remote browser registry index redirect must end in %s", registry.IndexName)
	}
	if _, err := registry.ParseIndex(indexData); err != nil {
		return registry.SearchReport{}, nil, err
	}
	if err := atomicfile.Write(filepath.Join(mirror, registry.IndexName), indexData, 0o600); err != nil {
		return registry.SearchReport{}, nil, err
	}

	client := localBrowserRegistryClient(opts.Timeout)
	search, err := client.Search(bounded, registry.SearchOptions{
		Location: mirror, Query: opts.Query, Limit: registry.DefaultMaxResults, At: opts.At,
	})
	if err != nil {
		return registry.SearchReport{}, nil, err
	}
	search.Registry = opts.Location
	finalBase := *finalIndex
	finalBase.RawQuery, finalBase.Fragment = "", ""
	finalBase.Path = strings.TrimSuffix(finalBase.Path, registry.IndexName)
	if !strings.HasSuffix(finalBase.Path, "/") {
		finalBase.Path += "/"
	}
	return pullBrowserRegistryMatches(bounded, client, mirror, opts.At, search, func(match registry.SearchResult) (string, error) {
		relative, err := registry.BlobPath(match.Entry.Bundle.Digest)
		if err != nil {
			return "", err
		}
		target := finalBase.ResolveReference(&url.URL{Path: relative})
		data, finalURL, err := downloadBrowserRegistryFile(bounded, opts.HTTPClient, target, registry.DefaultMaxBytes, opts.AllowUnsafeHosts)
		if err != nil {
			return "", err
		}
		if err := atomicfile.Write(filepath.Join(mirror, filepath.FromSlash(relative)), data, 0o600); err != nil {
			return "", err
		}
		return finalURL.String(), nil
	})
}

func localBrowserRegistryClient(timeout time.Duration) *registry.Client {
	return &registry.Client{
		NetworkPolicy: registry.NetworkNever,
		Timeout:       timeout, MaxBytes: registry.DefaultMaxBytes, MaxResults: registry.DefaultMaxResults,
	}
}

func pullBrowserRegistryMatches(ctx context.Context, client *registry.Client, location string, at time.Time, search registry.SearchReport, materialize func(registry.SearchResult) (string, error)) (registry.SearchReport, []browserRegistryPullAttempt, error) {
	attempts := make([]browserRegistryPullAttempt, 0, len(search.Results))
	for _, match := range search.Results {
		if err := ctx.Err(); err != nil {
			return registry.SearchReport{}, nil, err
		}
		source := ""
		if materialize != nil {
			var err error
			source, err = materialize(match)
			if err != nil {
				attempts = append(attempts, browserRegistryPullAttempt{Match: match, Err: err})
				continue
			}
		}
		coordinate := registry.Coordinate{ID: match.Entry.ID, Release: match.Entry.Release}
		pulled, err := client.Pull(ctx, registry.PullOptions{Location: location, Coordinate: &coordinate, At: at})
		if err == nil && source != "" {
			pulled.Source = source
		}
		attempts = append(attempts, browserRegistryPullAttempt{Match: match, Pulled: pulled, Err: err})
	}
	return search, attempts, nil
}

func remoteBrowserRegistryBase(raw string, allowUnsafe bool) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid browser registry URL")
	}
	if err := netpolicy.ValidateURL(parsed, allowUnsafe); err != nil {
		return nil, err
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("browser registry URL must not contain query or fragment")
	}
	if strings.HasSuffix(parsed.Path, "/"+registry.IndexName) || parsed.Path == registry.IndexName {
		parsed.Path = strings.TrimSuffix(parsed.Path, registry.IndexName)
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	return parsed, nil
}

func downloadBrowserRegistryFile(ctx context.Context, client *http.Client, target *url.URL, limit int64, allowUnsafe bool) ([]byte, *url.URL, error) {
	if err := netpolicy.ValidateURL(target, allowUnsafe); err != nil {
		return nil, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create browser registry request")
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("browser registry returned HTTP status %d", response.StatusCode)
	}
	if response.ContentLength > limit {
		return nil, nil, fmt.Errorf("browser registry response exceeds %d bytes", limit)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, nil, err
	}
	if int64(len(data)) > limit {
		return nil, nil, fmt.Errorf("browser registry response exceeds %d bytes", limit)
	}
	finalURL := target
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL
	}
	if err := netpolicy.ValidateURL(finalURL, allowUnsafe); err != nil {
		return nil, nil, err
	}
	if finalURL.RawQuery != "" || finalURL.Fragment != "" {
		return nil, nil, fmt.Errorf("browser registry redirect must not add query or fragment")
	}
	return data, finalURL, nil
}

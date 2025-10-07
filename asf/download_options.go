package asf

import (
	"github.com/example/go-asf/asf/download"
)

type downloadConfig struct {
	concurrency int
	verify      bool
	progress    download.ProgressFunc
	downloader  download.Manager
	basicAuth   *download.BasicAuth
}

// DownloadOption customises how product files are downloaded.
type DownloadOption func(*downloadConfig)

// WithDownloadConcurrency specifies the number of files to fetch in parallel.
func WithDownloadConcurrency(n int) DownloadOption {
	return func(cfg *downloadConfig) {
		if n > 0 {
			cfg.concurrency = n
		}
	}
}

// WithProgress registers a callback to receive download progress notifications.
func WithProgress(fn download.ProgressFunc) DownloadOption {
	return func(cfg *downloadConfig) {
		cfg.progress = fn
	}
}

// WithoutChecksum disables checksum verification after downloads.
func WithoutChecksum() DownloadOption {
	return func(cfg *downloadConfig) {
		cfg.verify = false
	}
}

// WithDownloader allows providing a custom download.Manager implementation.
func WithDownloader(m download.Manager) DownloadOption {
	return func(cfg *downloadConfig) {
		if m != nil {
			cfg.downloader = m
		}
	}
}

func (c *downloadConfig) ensureDefaults(auth *download.BasicAuth) {
	if c.concurrency <= 0 {
		c.concurrency = 2
	}
	if c.downloader == nil {
		c.downloader = download.NewManager(download.Config{
			Concurrency: c.concurrency,
			Verify:      c.verify,
			Progress:    c.progress,
			BasicAuth:   auth,
		})
	}
}

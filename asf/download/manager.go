package download

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	internalhttp "github.com/example/go-asf/asf/internal/http"
	"github.com/example/go-asf/asf/model"
	"golang.org/x/sync/errgroup"
)

// BasicAuth holds credentials for HTTP basic authentication.
type BasicAuth struct {
	Username string
	Password string
}

// ProgressFunc is invoked as bytes are written for an individual file.
type ProgressFunc func(FileProgress)

// FileProgress reports download progress for a single file.
type FileProgress struct {
	ProductID  string
	FileName   string
	URL        string
	Downloaded int64
	Total      int64
}

// Config controls how downloads are executed.
type Config struct {
	Concurrency int
	Verify      bool
	Progress    ProgressFunc
	BasicAuth   *BasicAuth
}

// Manager is responsible for downloading product files.
type Manager interface {
	Download(ctx context.Context, client *http.Client, userAgent string, product model.Product, destDir string) error
}

type manager struct {
	cfg Config
}

// NewManager constructs a download manager with the provided configuration.
func NewManager(cfg Config) Manager {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	return &manager{cfg: cfg}
}

func (m *manager) Download(ctx context.Context, client *http.Client, userAgent string, product model.Product, destDir string) error {
	if client == nil {
		return errors.New("http client is required")
	}
	if destDir == "" {
		return errors.New("destination directory is required")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}
	if len(product.Files) == 0 {
		return errors.New("product contains no downloadable files")
	}

	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, m.cfg.Concurrency)

	for _, file := range product.Files {
		f := file
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			return m.downloadFile(ctx, client, userAgent, product.ID, destDir, f)
		})
	}

	return g.Wait()
}

func (m *manager) downloadFile(ctx context.Context, client *http.Client, userAgent, productID, destDir string, file model.File) (err error) {
	if file.URL == "" {
		return errors.New("file missing URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.URL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if m.cfg.BasicAuth != nil && m.cfg.BasicAuth.Username != "" {
		req.SetBasicAuth(m.cfg.BasicAuth.Username, m.cfg.BasicAuth.Password)
	}

	resp, err := internalhttp.Do(ctx, client, req, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return internalhttp.HTTPError(resp)
	}

	name := file.Name
	if name == "" {
		if u, err := url.Parse(file.URL); err == nil {
			base := filepath.Base(u.Path)
			if base != "" && base != "." && base != "/" {
				name = base
			}
		}
	}
	if name == "" {
		return errors.New("could not determine filename")
	}

	finalPath := filepath.Join(destDir, name)
	tmpPath := finalPath + ".part"

	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		out.Close()
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	total := resp.ContentLength
	if total < 0 {
		total = file.Size
	}

	writer := newProgressWriter(out, m.cfg.Progress, FileProgress{
		ProductID: productID,
		FileName:  name,
		URL:       file.URL,
		Total:     total,
	})

	var hash hash.Hash
	if m.cfg.Verify && file.Checksum != "" {
		switch strings.ToLower(file.ChecksumType) {
		case "", "md5":
			hash = md5.New()
		case "sha1":
			hash = sha1.New()
		}
	}

	if hash != nil {
		writer.SetHasher(hash)
	}

	if _, err = io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	if hash != nil {
		sum := hex.EncodeToString(hash.Sum(nil))
		if !strings.EqualFold(sum, file.Checksum) {
			return fmt.Errorf("checksum mismatch for %s: expected %s got %s", name, file.Checksum, sum)
		}
	}

	if err = out.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err = os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// (rest of file unchanged)

type progressWriter struct {
	dst      io.Writer
	progress ProgressFunc
	meta     FileProgress
	hasher   hash.Hash
}

func newProgressWriter(dst io.Writer, fn ProgressFunc, meta FileProgress) *progressWriter {
	return &progressWriter{dst: dst, progress: fn, meta: meta}
}

func (w *progressWriter) SetHasher(h hash.Hash) {
	w.hasher = h
}

func (w *progressWriter) Write(p []byte) (int, error) {
	if w.hasher != nil {
		if _, err := w.hasher.Write(p); err != nil {
			return 0, err
		}
	}

	n, err := w.dst.Write(p)
	if n > 0 {
		w.meta.Downloaded += int64(n)
		if w.progress != nil {
			w.progress(w.meta)
		}
	}
	return n, err
}

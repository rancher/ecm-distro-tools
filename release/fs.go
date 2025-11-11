package release

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v77/github"
)

// FS implements fs.FS for GitHub release assets
type FS struct {
	ctx     context.Context
	client  *github.Client
	owner   string
	repo    string
	tag     string
	release *github.RepositoryRelease
	assets  map[string]*github.ReleaseAsset
}

// NewFS creates a new filesystem for accessing GitHub release assets
func NewFS(ctx context.Context, client *github.Client, owner, repo, tag string) (*FS, error) {
	if tag == "" {
		return nil, errors.New("invalid tag provided")
	}

	release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("getting release: %w", err)
	}

	fs := &FS{
		ctx:     ctx,
		client:  client,
		owner:   owner,
		repo:    repo,
		tag:     tag,
		release: release,
		assets:  make(map[string]*github.ReleaseAsset),
	}

	// Index assets by name for quick lookup
	for _, asset := range release.Assets {
		fs.assets[asset.GetName()] = asset
	}

	return fs, nil
}

// Open implements fs.FS for a GitHub release, treating assets as a filesystem
func (r *FS) Open(name string) (fs.File, error) {
	// Clean and normalize the path
	name = filepath.Clean(name)
	name = strings.TrimPrefix(name, "/")
	if name == "." {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	asset, ok := r.assets[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	rc, _, err := r.client.Repositories.DownloadReleaseAsset(r.ctx, r.owner, r.repo, asset.GetID(), http.DefaultClient)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	return &releaseFile{
		asset:      asset,
		readCloser: rc,
	}, nil
}

// releaseFile implements fs.File for a GitHub release asset
type releaseFile struct {
	asset      *github.ReleaseAsset
	readCloser io.ReadCloser
}

func (f *releaseFile) Stat() (fs.FileInfo, error) {
	return &releaseFileInfo{asset: f.asset}, nil
}

func (f *releaseFile) Read(b []byte) (int, error) {
	return f.readCloser.Read(b)
}

func (f *releaseFile) Close() error {
	return f.readCloser.Close()
}

// releaseFileInfo implements fs.FileInfo for a GitHub release asset
type releaseFileInfo struct {
	asset *github.ReleaseAsset
}

func (r *releaseFileInfo) Name() string       { return r.asset.GetName() }
func (r *releaseFileInfo) Size() int64        { return int64(r.asset.GetSize()) }
func (r *releaseFileInfo) Mode() fs.FileMode  { return 0444 } // read only
func (r *releaseFileInfo) ModTime() time.Time { return r.asset.GetCreatedAt().Time }
func (r *releaseFileInfo) IsDir() bool        { return false }
func (r *releaseFileInfo) Sys() interface{}   { return r.asset }

func (r *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	name = filepath.Clean(name)
	name = strings.TrimPrefix(name, "/")
	if name != "." {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}

	entries := make([]fs.DirEntry, 0, len(r.assets))
	for _, asset := range r.assets {
		entries = append(entries, &releaseFileInfo{asset: asset})
	}
	return entries, nil
}

// releaseFileInfo implements both fs.FileInfo and fs.DirEntry
func (r *releaseFileInfo) Type() fs.FileMode {
	return r.Mode()
}

func (r *releaseFileInfo) Info() (fs.FileInfo, error) {
	return r, nil
}

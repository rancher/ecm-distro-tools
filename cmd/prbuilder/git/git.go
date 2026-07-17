package git

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repository wraps go-git repository and worktree
type Repository struct {
	r  *git.Repository
	wt *git.Worktree
}

// Open opens an existing git repository at the given path
// This is equivalent to running git commands in an existing repo directory
func Open(path string) (*Repository, error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository at %s: %w", path, err)
	}

	wt, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	return &Repository{
		r:  r,
		wt: wt,
	}, nil
}

// Clone clones a repository to the specified path
// Supports shallow clones when depth > 0
// This replaces: gh repo clone <repo> <path> --branch=<branch> --depth=<depth>
func Clone(url, path, branch string, depth int) (*Repository, error) {
	auth, err := ResolveAuth(url)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth for %s: %w", url, err)
	}

	cloneOpts := &git.CloneOptions{
		URL:           url,
		Auth:          auth,
		Progress:      os.Stdout,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
	}

	if depth > 0 {
		cloneOpts.Depth = depth
	}

	r, err := git.PlainClone(path, false, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to clone %s: %w", url, err)
	}

	wt, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	return &Repository{
		r:  r,
		wt: wt,
	}, nil
}

// EnsureRemote creates a remote if it doesn't exist, or updates the URL if it does
// This handles the case where a remote might already exist with a different URL
func (r *Repository) EnsureRemote(name, url string) error {
	// Try to get existing remote
	remote, err := r.r.Remote(name)
	if err == nil {
		// Remote exists, check if URL matches
		urls := remote.Config().URLs
		if len(urls) > 0 && urls[0] == url {
			// URL matches, nothing to do
			return nil
		}
		// URL doesn't match, delete and recreate
		if err := r.r.DeleteRemote(name); err != nil {
			return fmt.Errorf("failed to delete existing remote %s: %w", name, err)
		}
	}

	// Create the remote
	_, err = r.r.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})
	if err != nil && !errors.Is(err, git.ErrRemoteExists) {
		return fmt.Errorf("failed to create remote %s: %w", name, err)
	}

	return nil
}

// GetRemoteURL returns the URL of the specified remote
func (r *Repository) GetRemoteURL(remoteName string) (string, error) {
	remote, err := r.r.Remote(remoteName)
	if err != nil {
		return "", fmt.Errorf("failed to get remote %s: %w", remoteName, err)
	}
	if len(remote.Config().URLs) == 0 {
		return "", fmt.Errorf("remote %s has no URL configured", remoteName)
	}
	return remote.Config().URLs[0], nil
}

// Fetch fetches from the specified remote
// This updates all remote-tracking branches
func (r *Repository) Fetch(remoteName string) error {
	// Get the remote URL to resolve auth
	remote, err := r.r.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("failed to get remote %s: %w", remoteName, err)
	}
	if len(remote.Config().URLs) == 0 {
		return fmt.Errorf("remote %s has no URL configured", remoteName)
	}
	remoteURL := remote.Config().URLs[0]

	// Resolve auth for the remote URL
	auth, err := ResolveAuth(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to resolve auth for %s: %w", remoteURL, err)
	}

	err = r.r.Fetch(&git.FetchOptions{
		RemoteName: remoteName,
		Auth:       auth,
		Progress:   os.Stdout,
		Tags:       git.AllTags,
		Force:      true, // Force update of remote-tracking branches
	})

	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to fetch from %s: %w", remoteName, err)
	}

	return nil
}

// CheckoutRemoteBranch checks out a remote branch, creating or resetting the local branch
// This is equivalent to: git checkout -B <localBranch> <remote>/<remoteBranch>
// This is the critical fix for stale branch issues - it always creates/resets the local
// branch from the remote reference, ensuring we're working with the latest code
//
// This pattern is proven in repository/repository.go lines 347-376
func (r *Repository) CheckoutRemoteBranch(remote, remoteBranch, localBranch string) error {
	// Construct the remote reference name (e.g., refs/remotes/origin/dev-v2.14)
	remoteRef := plumbing.NewRemoteReferenceName(remote, remoteBranch)

	// Check if the remote reference exists
	_, err := r.r.Reference(remoteRef, true)
	if err != nil {
		return fmt.Errorf("branch %s does not exist on remote %s: %w", remoteBranch, remote, err)
	}

	// Checkout the remote reference (this creates a detached HEAD at the remote branch)
	err = r.wt.Checkout(&git.CheckoutOptions{
		Branch: remoteRef,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout remote reference %s: %w", remoteRef, err)
	}

	// Get the current HEAD (now pointing to the remote branch's commit)
	headRef, err := r.r.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Create or update the local branch to point to the same commit
	localRef := plumbing.NewBranchReferenceName(localBranch)
	ref := plumbing.NewHashReference(localRef, headRef.Hash())
	err = r.r.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create/update local branch %s: %w", localBranch, err)
	}

	// Checkout the local branch
	err = r.wt.Checkout(&git.CheckoutOptions{
		Branch: localRef,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout local branch %s: %w", localBranch, err)
	}

	return nil
}

// CreateBranch creates a new branch from the current HEAD
// This is equivalent to: git checkout -b <branchName>
func (r *Repository) CreateBranch(branchName string) error {
	// Get the current HEAD
	headRef, err := r.r.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Create the new branch reference
	branchRef := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(branchRef, headRef.Hash())
	err = r.r.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	// Checkout the new branch
	err = r.wt.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	return nil
}

// HasChanges returns true if there are uncommitted changes in the working tree
// This is equivalent to: git status --porcelain (checking if output is empty)
func (r *Repository) HasChanges() (bool, error) {
	status, err := r.wt.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get repository status: %w", err)
	}

	return !status.IsClean(), nil
}

// AddAll stages all changes in the working tree
// This is equivalent to: git add -A
func (r *Repository) AddAll() error {
	err := r.wt.AddWithOptions(&git.AddOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	return nil
}

// CommitOptions holds options for creating a commit
type CommitOptions struct {
	Message     string
	AuthorName  string
	AuthorEmail string
}

// Commit creates a commit with the staged changes
// This is equivalent to: git commit -m <message>
// The author information is set in the commit, not the repository config
func (r *Repository) Commit(opts CommitOptions) error {
	if opts.Message == "" {
		return errors.New("commit message is required")
	}
	if opts.AuthorName == "" {
		return errors.New("author name is required")
	}
	if opts.AuthorEmail == "" {
		return errors.New("author email is required")
	}

	author := &object.Signature{
		Name:  opts.AuthorName,
		Email: opts.AuthorEmail,
		When:  time.Now(),
	}

	_, err := r.wt.Commit(opts.Message, &git.CommitOptions{
		Author:    author,
		Committer: author,
		All:       true,
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// Push pushes the current branch to the specified remote
// This is equivalent to: git push <remote> <current-branch>
func (r *Repository) Push(remoteName string) error {
	// Get the current branch name
	headRef, err := r.r.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	if !headRef.Name().IsBranch() {
		return errors.New("HEAD is not pointing to a branch (detached HEAD state)")
	}

	// Get the remote URL to resolve auth
	remote, err := r.r.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("failed to get remote %s: %w", remoteName, err)
	}
	if len(remote.Config().URLs) == 0 {
		return fmt.Errorf("remote %s has no URL configured", remoteName)
	}
	remoteURL := remote.Config().URLs[0]

	// Resolve auth for the remote URL
	auth, err := ResolveAuth(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to resolve auth for %s: %w", remoteURL, err)
	}

	// Push with authentication
	err = r.r.Push(&git.PushOptions{
		RemoteName: remoteName,
		Auth:       auth,
		Progress:   os.Stdout,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to push to %s: %w", remoteName, err)
	}

	return nil
}

// WorkingDirectory returns the path to the repository's working directory
func (r *Repository) WorkingDirectory() string {
	return r.wt.Filesystem.Root()
}

// CurrentBranch returns the name of the current branch
// Returns an error if HEAD is detached
func (r *Repository) CurrentBranch() (string, error) {
	headRef, err := r.r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	if !headRef.Name().IsBranch() {
		return "", errors.New("HEAD is detached, not on a branch")
	}

	return headRef.Name().Short(), nil
}

// CheckoutBranch checks out an existing branch by name
// This is equivalent to: git checkout <branchName>
func (r *Repository) CheckoutBranch(branchName string) error {
	branchRef := plumbing.NewBranchReferenceName(branchName)

	// Verify the branch exists
	_, err := r.r.Reference(branchRef, true)
	if err != nil {
		return fmt.Errorf("branch %s does not exist: %w", branchName, err)
	}

	// Checkout the branch
	err = r.wt.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	return nil
}

// DeleteBranch deletes a local branch
// This is equivalent to: git branch -D <branchName>
// The branch must not be currently checked out
func (r *Repository) DeleteBranch(branchName string) error {
	// Verify we're not on the branch we're trying to delete
	currentBranch, err := r.CurrentBranch()
	if err == nil && currentBranch == branchName {
		return fmt.Errorf("cannot delete branch %s: currently checked out", branchName)
	}

	branchRef := plumbing.NewBranchReferenceName(branchName)

	// Delete the branch reference
	err = r.r.Storer.RemoveReference(branchRef)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}

	return nil
}

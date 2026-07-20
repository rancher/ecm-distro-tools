package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// createTestRepo creates a temporary git repository for testing
func createTestRepo(t *testing.T) (string, *git.Repository) {
	t.Helper()

	// Create temp directory
	dir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	r, err := git.PlainInit(dir, false)
	if err != nil {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Create initial commit
	wt, err := r.Worktree()
	if err != nil {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o600); err != nil {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = wt.Add("test.txt")
	if err != nil {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
		t.Fatalf("failed to add test file: %v", err)
	}

	author := &object.Signature{
		Name:  "Test User",
		Email: "test@example.com",
	}

	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author:    author,
		Committer: author,
	})
	if err != nil {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
		t.Fatalf("failed to create initial commit: %v", err)
	}

	return dir, r
}

// Auth resolution tests are skipped because they require:
// - SSH keys configured
// - git credential helper configured
// - Or GH_TOKEN/GITHUB_TOKEN environment variables
// These vary per environment and are tested in the gitauth package

func TestOpen(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if repo == nil {
		t.Fatal("Open() returned nil repository")
	}

	if repo.r == nil {
		t.Error("Open() repository has nil git.Repository")
	}

	if repo.wt == nil {
		t.Error("Open() repository has nil worktree")
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open("/nonexistent/path")
	if err == nil {
		t.Error("Open() with invalid path should return error")
	}
}

// TestOpen_NoToken removed - auth is now resolved automatically from environment

func TestEnsureRemote(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Test creating a new remote
	err = repo.EnsureRemote("origin", "https://github.com/test/repo.git")
	if err != nil {
		t.Errorf("EnsureRemote() failed to create remote: %v", err)
	}

	// Verify remote was created
	remote, err := repo.r.Remote("origin")
	if err != nil {
		t.Errorf("Failed to get remote: %v", err)
	}

	urls := remote.Config().URLs
	if len(urls) == 0 || urls[0] != "https://github.com/test/repo.git" {
		t.Errorf("Remote URL = %v, want https://github.com/test/repo.git", urls)
	}

	// Test updating existing remote with different URL
	err = repo.EnsureRemote("origin", "https://github.com/test/other-repo.git")
	if err != nil {
		t.Errorf("EnsureRemote() failed to update remote: %v", err)
	}

	// Verify URL was updated
	remote, err = repo.r.Remote("origin")
	if err != nil {
		t.Errorf("Failed to get updated remote: %v", err)
	}

	urls = remote.Config().URLs
	if len(urls) == 0 || urls[0] != "https://github.com/test/other-repo.git" {
		t.Errorf("Updated remote URL = %v, want https://github.com/test/other-repo.git", urls)
	}
}

func TestCreateBranch(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Create a new branch
	err = repo.CreateBranch("test-branch")
	if err != nil {
		t.Errorf("CreateBranch() failed: %v", err)
	}

	// Verify we're on the new branch
	head, err := repo.r.Head()
	if err != nil {
		t.Errorf("Failed to get HEAD: %v", err)
	}

	if !head.Name().IsBranch() || head.Name().Short() != "test-branch" {
		t.Errorf("HEAD = %v, want refs/heads/test-branch", head.Name())
	}
}

func TestHasChanges(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Clean repo should have no changes
	hasChanges, err := repo.HasChanges()
	if err != nil {
		t.Errorf("HasChanges() failed: %v", err)
	}
	if hasChanges {
		t.Error("HasChanges() = true, want false for clean repo")
	}

	// Modify a file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("modified content"), 0o600); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Should now have changes
	hasChanges, err = repo.HasChanges()
	if err != nil {
		t.Errorf("HasChanges() failed: %v", err)
	}
	if !hasChanges {
		t.Error("HasChanges() = false, want true after file modification")
	}
}

func TestAddAll(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Create a new file
	newFile := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0o600); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Add all changes
	err = repo.AddAll()
	if err != nil {
		t.Errorf("AddAll() failed: %v", err)
	}

	// Verify file is staged
	status, err := repo.wt.Status()
	if err != nil {
		t.Errorf("Failed to get status: %v", err)
	}

	fileStatus := status.File("new.txt")
	if fileStatus.Staging != git.Added {
		t.Errorf("File staging status = %v, want Added", fileStatus.Staging)
	}
}

func TestCommit(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Create and stage a new file
	newFile := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0o600); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	if err := repo.AddAll(); err != nil {
		t.Fatalf("AddAll() failed: %v", err)
	}

	// Test successful commit
	err = repo.Commit(CommitOptions{
		Message:     "Test commit",
		AuthorName:  "Test User",
		AuthorEmail: "test@example.com",
	})
	if err != nil {
		t.Errorf("Commit() failed: %v", err)
	}

	// Verify commit was created
	head, err := repo.r.Head()
	if err != nil {
		t.Errorf("Failed to get HEAD: %v", err)
	}

	commit, err := repo.r.CommitObject(head.Hash())
	if err != nil {
		t.Errorf("Failed to get commit: %v", err)
	}

	if commit.Message != "Test commit" {
		t.Errorf("Commit message = %q, want %q", commit.Message, "Test commit")
	}

	if commit.Author.Name != "Test User" {
		t.Errorf("Author name = %q, want %q", commit.Author.Name, "Test User")
	}
}

func TestCommit_MissingFields(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	tests := []struct {
		name string
		opts CommitOptions
	}{
		{
			name: "missing message",
			opts: CommitOptions{
				AuthorName:  "Test User",
				AuthorEmail: "test@example.com",
			},
		},
		{
			name: "missing author name",
			opts: CommitOptions{
				Message:     "Test commit",
				AuthorEmail: "test@example.com",
			},
		},
		{
			name: "missing author email",
			opts: CommitOptions{
				Message:    "Test commit",
				AuthorName: "Test User",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Commit(tt.opts)
			if err == nil {
				t.Error("Commit() expected error for missing field, got nil")
			}
		})
	}
}

func TestCheckoutRemoteBranch(t *testing.T) {
	dir, r := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	// Create a remote and a remote-tracking branch manually
	_, err := r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/test/repo.git"},
	})
	if err != nil {
		t.Fatalf("Failed to create remote: %v", err)
	}

	// Get HEAD commit
	head, err := r.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}

	// Create a remote-tracking branch reference
	remoteRef := plumbing.NewRemoteReferenceName("origin", "main")
	ref := plumbing.NewHashReference(remoteRef, head.Hash())
	err = r.Storer.SetReference(ref)
	if err != nil {
		t.Fatalf("Failed to create remote ref: %v", err)
	}

	repo := &Repository{
		r:  r,
		wt: nil,
	}
	repo.wt, err = r.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	// Test checking out the remote branch
	err = repo.CheckoutRemoteBranch("origin", "main", "main")
	if err != nil {
		t.Errorf("CheckoutRemoteBranch() failed: %v", err)
	}

	// Verify we're on the local main branch
	head, err = r.Head()
	if err != nil {
		t.Errorf("Failed to get HEAD: %v", err)
	}

	if !head.Name().IsBranch() || head.Name().Short() != "main" {
		t.Errorf("HEAD = %v, want refs/heads/main", head.Name())
	}
}

func TestWorkingDirectory(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	workDir := repo.WorkingDirectory()

	// Resolve both paths to handle symlinks (e.g., /var -> /private/var on macOS)
	expectedPath, err := filepath.EvalSymlinks(dir)
	if err != nil {
		expectedPath = dir
	}

	actualPath, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		actualPath = workDir
	}

	if actualPath != expectedPath {
		t.Errorf("WorkingDirectory() = %q, want %q", actualPath, expectedPath)
	}
}

func TestCurrentBranch(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Should be on master or main initially
	branch, err := repo.CurrentBranch()
	if err != nil {
		t.Errorf("CurrentBranch() failed: %v", err)
	}

	if branch != "master" && branch != "main" {
		t.Errorf("CurrentBranch() = %q, want master or main", branch)
	}
}

func TestCheckoutBranch(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Create a test branch
	err = repo.CreateBranch("test-branch")
	if err != nil {
		t.Fatalf("CreateBranch() failed: %v", err)
	}

	// Get initial branch
	initialBranch, err := repo.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch() failed: %v", err)
	}

	if initialBranch != "test-branch" {
		t.Errorf("After CreateBranch, CurrentBranch() = %q, want test-branch", initialBranch)
	}

	// Create another branch to switch back to
	err = repo.CreateBranch("other-branch")
	if err != nil {
		t.Fatalf("CreateBranch() failed: %v", err)
	}

	// Checkout the first branch again
	err = repo.CheckoutBranch("test-branch")
	if err != nil {
		t.Errorf("CheckoutBranch() failed: %v", err)
	}

	// Verify we're on the right branch
	currentBranch, err := repo.CurrentBranch()
	if err != nil {
		t.Errorf("CurrentBranch() failed: %v", err)
	}

	if currentBranch != "test-branch" {
		t.Errorf("After CheckoutBranch, CurrentBranch() = %q, want test-branch", currentBranch)
	}
}

func TestCheckoutBranch_NonExistent(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Try to checkout a non-existent branch
	err = repo.CheckoutBranch("does-not-exist")
	if err == nil {
		t.Error("CheckoutBranch() expected error for non-existent branch, got nil")
	}
}

func TestDeleteBranch(t *testing.T) {
	dir, r := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Create a test branch
	err = repo.CreateBranch("to-delete")
	if err != nil {
		t.Fatalf("CreateBranch() failed: %v", err)
	}

	// Create another branch and switch to it
	err = repo.CreateBranch("keep-this")
	if err != nil {
		t.Fatalf("CreateBranch() failed: %v", err)
	}

	// Delete the first branch
	err = repo.DeleteBranch("to-delete")
	if err != nil {
		t.Errorf("DeleteBranch() failed: %v", err)
	}

	// Verify the branch is gone
	branchRef := plumbing.NewBranchReferenceName("to-delete")
	_, err = r.Reference(branchRef, true)
	if err == nil {
		t.Error("Branch still exists after DeleteBranch()")
	}
}

func TestDeleteBranch_CurrentBranch(t *testing.T) {
	dir, _ := createTestRepo(t)
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Create and checkout a branch
	err = repo.CreateBranch("current")
	if err != nil {
		t.Fatalf("CreateBranch() failed: %v", err)
	}

	// Try to delete the current branch
	err = repo.DeleteBranch("current")
	if err == nil {
		t.Error("DeleteBranch() expected error when deleting current branch, got nil")
	}
}

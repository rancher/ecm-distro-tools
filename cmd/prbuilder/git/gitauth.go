package git

// This file contains git authentication resolution logic that works like native git.
//
// It resolves SSH/HTTPS credentials for git remote URLs using the same precedence
// rules as native git, with added support for GitHub tokens in CI/CD environments.
//
// Authentication resolution order:
//  1. GH_TOKEN or GITHUB_TOKEN environment variable (for CI/CD)
//  2. SSH resolution for ssh:// or git@host:path URLs:
//     - ~/.ssh/config (Host alias, User override, IdentityFile)
//     - Default identity files: id_ed25519, id_ecdsa, id_rsa, id_dsa
//     - Running ssh-agent (SSH_AUTH_SOCK)
//  3. HTTPS resolution: shells out to `git credential fill`, which uses
//     whatever credential helper the user has configured (osxkeychain,
//     manager-core, libsecret, cache, store, ...)
//

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kevinburke/ssh_config"
	gossh "golang.org/x/crypto/ssh"
)

// ResolveAuth resolves authentication for a git remote URL.
//
// This uses the same authentication resolution as native git:
//  1. GH_TOKEN or GITHUB_TOKEN environment variable (for CI/CD)
//  2. SSH keys from ~/.ssh/config, default key files, or ssh-agent
//  3. HTTPS credentials from `git credential fill` (user's configured helper)
//
// This means users don't need to configure anything beyond what they
// already have set up for git itself.
func ResolveAuth(remoteURL string) (transport.AuthMethod, error) {
	mgr := newAuthManager()
	return mgr.resolve(remoteURL)
}

// authManager resolves transport.AuthMethod values for git remote URLs.
type authManager struct {
	getSSHConfig func(host, key string) string
	homeDir      func() (string, error)
	readFile     func(string) ([]byte, error)
	getEnv       func(string) string
}

// newAuthManager creates an authManager wired to real OS and SSH-config implementations.
func newAuthManager() *authManager {
	return &authManager{
		getSSHConfig: sshConfigGet,
		homeDir:      os.UserHomeDir,
		readFile:     os.ReadFile,
		getEnv:       os.Getenv,
	}
}

// resolve picks the right auth method for a remote URL.
//
// Priority order:
//  1. If GH_TOKEN or GITHUB_TOKEN is set, use it for HTTPS (CI/CD mode)
//  2. Otherwise, follow native git precedence for the protocol
func (m *authManager) resolve(remoteURL string) (transport.AuthMethod, error) {
	scheme, host, user, _ := ParseRemote(remoteURL)

	// Priority 1: Check for GitHub token in environment (CI/CD mode)
	if scheme == "http" || scheme == "https" {
		if token := m.getGitHubToken(); token != "" {
			return &githttp.BasicAuth{
				Username: "x-access-token",
				Password: token,
			}, nil
		}
	}

	// Priority 2: Use protocol-specific resolution (local dev mode)
	switch scheme {
	case "http", "https":
		return m.resolveHTTPAuth(scheme, host)
	case "ssh", "":
		return m.resolveSSHAuth(host, user)
	default:
		return nil, fmt.Errorf("unsupported remote scheme %q in %q", scheme, remoteURL)
	}
}

// getGitHubToken retrieves the GitHub token from environment variables
// Checks GH_TOKEN first (GitHub CLI convention), then GITHUB_TOKEN (Actions)
func (m *authManager) getGitHubToken() string {
	if token := m.getEnv("GH_TOKEN"); token != "" {
		return token
	}
	return m.getEnv("GITHUB_TOKEN")
}

// ParseRemote understands the three URL shapes git accepts:
//
//	https://host/path
//	ssh://user@host/path
//	user@host:path        (scp-like shorthand, e.g. git@github.com:org/repo.git)
func ParseRemote(remote string) (scheme, host, user, path string) {
	if strings.HasPrefix(remote, "http://") || strings.HasPrefix(remote, "https://") {
		u, err := url.Parse(remote)
		if err != nil {
			return "", "", "", ""
		}
		return u.Scheme, u.Hostname(), u.User.Username(), u.Path
	}
	if strings.HasPrefix(remote, "ssh://") {
		u, err := url.Parse(remote)
		if err != nil {
			return "", "", "", ""
		}
		return "ssh", u.Hostname(), u.User.Username(), u.Path
	}
	// SCP-like shorthand: git@github.com:org/repo.git
	if at := strings.Index(remote, "@"); at >= 0 && strings.Contains(remote[at:], ":") {
		user = remote[:at]
		rest := remote[at+1:]
		colon := strings.Index(rest, ":")
		host = rest[:colon]
		path = rest[colon+1:]
		return "ssh", host, user, path
	}
	return "", remote, "", ""
}

// resolveSSHAuth implements SSH key resolution following native ssh precedence:
//  1. ~/.ssh/config for Host-specific settings
//  2. Explicit IdentityFile from config, or default key files
//  3. Running ssh-agent via SSH_AUTH_SOCK
func (m *authManager) resolveSSHAuth(host, user string) (transport.AuthMethod, error) {
	if user == "" {
		user = "git"
	}

	home, err := m.homeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	// 1. ~/.ssh/config: Host alias may rename the user and point at a
	//    specific IdentityFile.
	if cfgUser := m.getSSHConfig(host, "User"); cfgUser != "" {
		user = cfgUser
	}

	// 2. Build the full list of candidate key paths. When ssh config names an
	//    explicit IdentityFile we use only that; otherwise we probe all the
	//    default names in the same priority order native ssh uses.
	var candidates []string
	if keyPath := m.getSSHConfig(host, "IdentityFile"); keyPath != "" && keyPath != "~/.ssh/identity" {
		candidates = append(candidates, expandHome(keyPath, home))
	} else {
		for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa", "id_dsa"} {
			candidates = append(candidates, filepath.Join(home, ".ssh", name))
		}
	}

	// 3. Parse every readable key file into a signer. We collect them all so
	//    the SSH handshake can offer each one to the server — mirroring what
	//    native ssh does rather than stopping at the first parseable key.
	var fileSigners []gossh.Signer
	for _, path := range candidates {
		keyBytes, err := m.readFile(path)
		if err != nil {
			continue
		}
		pk, err := gitssh.NewPublicKeys(user, keyBytes, "")
		if err != nil {
			continue
		}
		fileSigners = append(fileSigners, pk.Signer)
	}

	// 4. Capture the agent callback lazily so it is evaluated at handshake
	//    time, not at resolve time, matching how native ssh uses the agent.
	var agentCallback func() ([]gossh.Signer, error)
	if m.getEnv("SSH_AUTH_SOCK") != "" {
		if agentAuth, err := gitssh.NewSSHAgentAuth(user); err == nil {
			agentCallback = agentAuth.Callback
		}
	}

	if len(fileSigners) == 0 && agentCallback == nil {
		return nil, fmt.Errorf(
			"no usable SSH key or agent found for host %q (checked ~/.ssh/config, default key files in ~/.ssh, and ssh-agent)",
			host,
		)
	}

	return &gitssh.PublicKeysCallback{
		User: user,
		Callback: func() ([]gossh.Signer, error) {
			all := make([]gossh.Signer, 0, len(fileSigners))
			copy(all, fileSigners)
			if agentCallback != nil {
				if agentSigners, err := agentCallback(); err == nil {
					all = append(all, agentSigners...)
				}
			}
			return all, nil
		},
	}, nil
}

// resolveHTTPAuth shells out to `git credential fill`, so any credential
// helper already configured via `git config credential.helper` is honored
// without this tool needing to know how that helper stores secrets.
//
// This is only called if GH_TOKEN/GITHUB_TOKEN are not set.
func (m *authManager) resolveHTTPAuth(scheme, host string) (transport.AuthMethod, error) {
	input := fmt.Sprintf("protocol=%s\nhost=%s\n\n", scheme, host)

	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf(
			"git credential fill failed for host %q: %w (output: %s) — set GH_TOKEN or configure a git credential helper",
			host, err, strings.TrimSpace(out.String()),
		)
	}

	creds := map[string]string{}
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		if eq := strings.Index(line, "="); eq > 0 {
			creds[line[:eq]] = line[eq+1:]
		}
	}

	if creds["username"] == "" {
		return nil, fmt.Errorf(
			"no credentials returned for host %q; set GH_TOKEN environment variable or configure a git credential helper (`git config credential.helper ...`)",
			host,
		)
	}

	return &githttp.BasicAuth{
		Username: creds["username"],
		Password: creds["password"],
	}, nil
}

func sshConfigGet(host, key string) string {
	if val, err := ssh_config.GetStrict(host, key); err == nil && val != "" {
		return val
	}
	return ssh_config.Get(host, key)
}

func expandHome(p, home string) string {
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// ExtractOwnerRepo extracts "owner/repo" from a git remote URL.
// It handles all three URL formats that git accepts:
//   - HTTPS: https://github.com/owner/repo.git
//   - SSH explicit: ssh://git@github.com/owner/repo.git
//   - SSH SCP-like: git@github.com:owner/repo.git
//
// Returns the owner and repository name, with .git suffix stripped.
func ExtractOwnerRepo(remoteURL string) (owner, repo string, err error) {
	_, _, _, path := ParseRemote(remoteURL)
	if path == "" {
		return "", "", fmt.Errorf("no path found in remote URL: %s", remoteURL)
	}

	// Strip leading slash and .git suffix
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, ".git")

	// Split on slash to get owner/repo
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid path format in remote URL %s: expected owner/repo, got %s", remoteURL, path)
	}

	// Handle paths like "owner/repo" or "owner/repo/extra" (use first two parts)
	owner = parts[0]
	repo = parts[1]

	return owner, repo, nil
}

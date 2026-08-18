# prbuilder

A CLI tool to automate version bump PRs across downstream/consumer repositories.

## Overview

When your project releases a new version, `prbuilder` creates PRs in all downstream repositories that consume it. Define version-to-branch mappings and update scripts once, then run a single command to update everything.

**Quick start:**
```bash
# Build
go build -o prbuilder .

# Create PRs for all configured repositories
export GH_TOKEN="ghp_..."
prbuilder create-prs --tag v10.3.2
```

## Basic Configuration

Create a config file (e.g., `.github/pr-consumer-config.yml`) defining version mappings and target repositories.

### Single Repository

For updating one downstream repository:

```yaml
version_strategy: major  # v10.3.2 → "10"

publishing_rules:
  "11": "dev-v2.15"
  "10": "dev-v2.14"
  "9": "dev-v2.13"

target:
  repo: "rancher/rancher"
  update_script_path: "./scripts/bump-rancher.sh"
```

Usage:
```bash
# CI/automation
prbuilder create-prs --tag v10.3.2

# Local testing with existing clone
prbuilder create-prs --tag v10.3.2 --target-dir ~/repos/rancher --dry-run
```

### Multiple Repositories

For updating multiple downstream repositories:

```yaml
version_strategy: major

publishing_rules:
  "11": "dev-v2.15"
  "10": "dev-v2.14"

targets:
  - repo: "rancher/rancher"
    update_script_path: "./scripts/bump-rancher.sh"
  
  - repo: "rancher/charts"
    update_script_path: "./scripts/bump-charts.sh"
```

Usage:
```bash
# Processes all targets automatically
prbuilder create-prs --tag v10.3.2
```

### Update Scripts

Update scripts receive context via environment variables:

```bash
#!/bin/bash
set -e

# Environment variables available:
# PRBUILDER_TAG          = v10.3.2
# PRBUILDER_VERSION      = 10
# PRBUILDER_TARGET_DIR   = /tmp/prbuilder-123
# PRBUILDER_TARGET_REPO  = rancher/rancher
# PRBUILDER_TARGET_BRANCH = dev-v2.14
# PRBUILDER_SOURCE_DIR   = /github/workspace

echo "Updating $PRBUILDER_TARGET_REPO to $PRBUILDER_TAG"

cd "$PRBUILDER_TARGET_DIR"
go get github.com/myorg/mylib@"$PRBUILDER_TAG"
go mod tidy
```

## What Gets Created

When you run `prbuilder create-prs --tag v10.3.2`, here's what gets generated:

### Branch Name
```
bot/dev-v2.14-backup-restore-operator-bump-v10.3.2-1721234567
```

Format: `bot/{target-branch}-{component}-bump-{tag}-{timestamp}`

- `bot/` prefix indicates automation
- Target branch shows where the PR will merge
- Component name identifies what's being updated
- Tag and timestamp ensure uniqueness

### PR Title
```
[dev-v2.14] Bump backup-restore-operator to v10.3.2
```

Format: `[{target-branch}] Bump {component} to {tag}`

### PR Body
```markdown
Automated version bump to `v10.3.2` from upstream release.

This PR updates the dependencies to use the newly released version.

**Component:** backup-restore-operator
**Release tag:** v10.3.2
**Target branch:** dev-v2.14
**Release notes:** https://github.com/rancher/backup-restore-operator/releases/tag/v10.3.2

---
_This PR was automatically created by prbuilder_
```

The component name and release notes link are automatically extracted from your source repository's git remote.

## Advanced Configuration

### Version Strategies

Control how versions are extracted from tags:

```yaml
# Extract major version only
version_strategy: major
# v10.3.2 → "10"

# Extract major.minor version
version_strategy: major.minor
# v10.3.2 → "10.3"
```

### Multiple Target Branches

Create PRs to multiple branches for a single version:

```yaml
publishing_rules:
  "10": ["dev-v2.14", "release-v2.14"]  # Creates 2 PRs
```

When you run `prbuilder create-prs --tag v10.3.2`, it creates separate PRs for both `dev-v2.14` and `release-v2.14`.

### Wildcard Fallback

Use `"*"` to match versions not explicitly mapped:

```yaml
publishing_rules:
  "11": "dev-v2.15"
  "10": "dev-v2.14"
  "*": "main"  # All other versions go to main
```

### Per-Target Overrides

Override global mappings for specific targets:

```yaml
publishing_rules:
  "10": "dev-v2.14"

targets:
  - repo: "rancher/rancher"
    update_script_path: "./scripts/bump.sh"
  
  - repo: "rancher/charts"
    update_script_path: "./scripts/bump.sh"
    publishing_rules:
      "10": "release-v2.14"  # Override: charts uses different branch
```

### Single Script for Multiple Targets

Use `PRBUILDER_TARGET_REPO` to handle multiple targets in one script:

```bash
#!/bin/bash
set -e

case "$PRBUILDER_TARGET_REPO" in
  "rancher/rancher")
    sed -i "s/backup_version=.*/backup_version=$PRBUILDER_TAG/" \
      "$PRBUILDER_TARGET_DIR/config.yaml"
    ;;
  "rancher/charts")
    sed -i "s/version: .*/version: ${PRBUILDER_TAG#v}/" \
      "$PRBUILDER_TARGET_DIR/Chart.yaml"
    ;;
  *)
    echo "Unknown target: $PRBUILDER_TARGET_REPO"
    exit 1
    ;;
esac
```

### CLI Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--tag`, `-t` | `TAG` | - | The release tag (e.g., v10.3.2) |
| `--config`, `-c` | `CONFIG_FILE` | `.github/pr-consumer-config.yml` | Path to config file |
| `--dry-run`, `-n` | `DRY_RUN=true` | false | Show changes without creating PRs |
| `--target-dir`, `-d` | - | - | Path to existing clone (single-target only) |
| `--remote`, `-r` | - | `origin` | Git remote name for push |

### GitHub Actions Integration

```yaml
- name: Create PRs
  env:
    GH_TOKEN: ${{ secrets.PAT_TOKEN }}
    TAG: ${{ github.ref_name }}
  run: prbuilder create-prs
```

When running in GitHub Actions:
- Reads config from environment variables
- Writes PR URLs to `$GITHUB_OUTPUT`
- Uses `github-actions[bot]` as commit author

## Troubleshooting

### "GH_TOKEN or GITHUB_TOKEN environment variable is required"

Set the GitHub token:
```bash
export GH_TOKEN="ghp_..."
```

Generate a token with `repo` scope at https://github.com/settings/tokens

### "No branch mapping found for version X"

Your `publishing_rules` doesn't include the extracted version. Check:
- Your tag format matches `version_strategy` (e.g., `v10.3.2` with `major` → `"10"`)
- You have an entry for that version or a `"*"` wildcard

Example fix:
```yaml
publishing_rules:
  "10": "dev-v2.14"
  "*": "main"  # Fallback for other versions
```

### "Failed to clone repository"

Check:
- Your token has access to the target repository
- Repository name format is correct (`owner/repo`)
- The target branch exists on the remote

### "Branch X does not exist on remote Y"

The target branch in your `publishing_rules` doesn't exist. Verify the branch exists:
```bash
git ls-remote https://github.com/OWNER/REPO refs/heads/BRANCH_NAME
```

### "Update script not found"

Ensure:
- Script path is relative to source repo root
- Script is committed to your repository
- Path starts with `./`

Example:
```yaml
update_script_path: "./scripts/bump.sh"  # ✓ Correct
update_script_path: "scripts/bump.sh"    # ✗ Wrong
```

### No changes detected

If your update script runs but no PR is created, the script made no changes to the target repository. This is expected behavior when:
- The version is already up to date
- The script logic determined no update is needed

To debug:
```bash
# Run with --dry-run and check what the script does
prbuilder create-prs --tag v10.3.2 --dry-run
```

## Contributing

### Building

```bash
# Build for current platform
go build -o prbuilder .

# Build for all platforms
make -C cmd/prbuilder
```

### Testing

```bash
# Run tests
go test ./...

# Run linter
golangci-lint run --timeout=5m
```

### Code Structure

- `cmd/` - CLI commands
- `config/` - Configuration file parsing
- `git/` - Git operations (clone, fetch, commit, push)
- `prbuilder/` - Core logic
  - `branchbuilder.go` - Local operations (clone, branch creation, commits)
  - `publisher.go` - Remote operations (push, PR creation)
  - `prbuilder.go` - Orchestrates the workflow

### Requirements

- Go 1.26 or later
- GitHub Personal Access Token with `repo` scope

All git operations and GitHub API calls are handled via Go libraries ([go-git](https://github.com/go-git/go-git) and [go-github](https://github.com/google/go-github)). No external CLI tools required.

## License

Apache 2.0

# prbuilder

A CLI tool to create pull requests in downstream/consumer repositories when a new version is tagged.

## Overview

`prbuilder` automates the process of creating PRs across multiple repositories when your project releases a new version. It's designed for scenarios where a library/component is consumed by multiple applications that need coordinated version updates.

## Installation

Build from source:

```bash
cd cmd/prbuilder
go build -o prbuilder .
```

Or use the Makefile for multi-platform builds:

```bash
make -C cmd/prbuilder
```

## Usage

`prbuilder` uses the `create-prs` command. The mode (single-target or multi-target) is automatically determined by your config file structure.

### Config-Based Modes

**Single-target mode** - Use `target:` (singular) in your config:
```yaml
target:
  repo: "rancher/rancher"
  update_script_path: "./scripts/bump.sh"
```
- Supports `--target-dir` for working with an existing local clone
- Perfect for local development and testing
- Ideal for repos that only need to update one downstream repository

**Multi-target mode** - Use `targets:` (plural) in your config:
```yaml
targets:
  - repo: "rancher/rancher"
    update_script_path: "./scripts/bump-rancher.sh"
  - repo: "rancher/charts"
    update_script_path: "./scripts/bump-charts.sh"
```
- Automatically clones each target repository
- Processes all targets in sequence
- Used in CI/automated workflows
- `--target-dir` flag is rejected (mode mismatch)

### Examples

**CI/Automation (multi-target):**
```bash
prbuilder create-prs --tag v10.3.2 --config .github/pr-consumer-config.yml
```

**Local testing with existing clone (single-target):**
```bash
prbuilder create-prs \
  --tag v10.3.2 \
  --config .github/pr-consumer-config.yml \
  --target-dir ~/repos/rancher \
  --remote upstream \
  --dry-run
```

**Local testing with auto-clone (single-target):**
```bash
prbuilder create-prs --tag v10.3.2 --config .github/pr-consumer-config.yml --dry-run
```

### Command Line Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--tag`, `-t` | `TAG` | - | The tag that was released (e.g., v10.3.2) |
| `--config`, `-c` | `CONFIG_FILE` | `.github/pr-consumer-config.yml` | Path to config file |
| `--dry-run`, `-n` | `DRY_RUN=true` | false | Dry run mode (show changes but don't create PRs) |
| `--target-dir`, `-d` | - | - | Path to already-cloned target repo (only for single-target configs) |
| `--remote`, `-r` | - | `origin` | Git remote name to use for push |

### Environment Variables

- `GH_TOKEN` or `GITHUB_TOKEN` - GitHub authentication token (required)
- `GITHUB_WORKSPACE` - Source repository path (defaults to current directory)
- `GITHUB_OUTPUT` - GitHub Actions output file (optional, for CI integration)

## Configuration File

The config file defines version-to-branch mappings and target repositories. The config structure determines the operating mode.

### Single-Target Configuration (with `target:` singular)

```yaml
# How to parse version from tags
version_strategy: major

# Global version to branch mapping
publishing_rules:
  "11": "dev-v2.15"
  "10": "dev-v2.14"
  "9": "dev-v2.13"

# Single target repository (note: "target" singular)
target:
  repo: "rancher/rancher"
  update_script_path: "./scripts/bump-rancher.sh"
  # Optional: override global mapping
  publishing_rules:
    "11": "release-v2.15"
    "10": "release-v2.14"
```

Supports `--target-dir` flag for working with existing clones.

### Multi-Target Configuration (with `targets:` plural)

```yaml
# How to parse version from tags
version_strategy: major

# Global version to branch mapping
publishing_rules:
  "11": "dev-v2.15"
  "10": "dev-v2.14"
  "9": "dev-v2.13"

# Multiple target repositories (note: "targets" plural)
targets:
  - repo: "rancher/rancher"
    update_script_path: "./scripts/bump-rancher.sh"
  
  - repo: "rancher/charts"
    # Override global mapping for this target
    publishing_rules:
      "11": "release-v2.15"
      "10": "release-v2.14"
    update_script_path: "./scripts/bump-charts.sh"
```

Processes all targets automatically. Does not support `--target-dir`.

### Configuration Fields

#### `version_strategy`

How to extract version from tags:
- `major`: `v10.3.2` → `10`
- `major.minor`: `v10.3.2` → `10.3`

Default: `major`

#### `publishing_rules`

Global mapping of versions to target branches. Can be overridden per target.

**Supports:**
- **Single branch** (string): `"10": "dev-v2.14"`
- **Multiple branches** (array): `"10": ["dev-v2.14", "release-v2.14"]` - Creates separate PRs for each branch
- **Wildcard fallback** (special key): `"*": "main"` - Matches any version not explicitly mapped

**Examples:**

```yaml
# Single release line - all versions go to main
publishing_rules:
  "*": "main"

# Specific mappings with fallback
publishing_rules:
  "11": "dev-v2.15"
  "10": "dev-v2.14"
  "*": "main"  # All other versions (0.x, 9, 8, etc.) go to main

# Multiple target branches for a version
publishing_rules:
  "10": ["dev-v2.14", "release-v2.14"]  # Creates 2 PRs for v10.x.x releases
  "9": "release-v2.13"
```

#### `target` vs `targets`

**Single-target mode** - Use `target:` (singular):
- `repo` (required): Target repository in `owner/repo` format
- `update_script` (required): Path to update script in source repo (receives environment variables)
- `publishing_rules` (optional): Per-target version mapping (overrides global)
- Supports `--target-dir` flag

**Multi-target mode** - Use `targets:` (plural):
- Array of target repositories with the same fields as above
- Does not support `--target-dir` flag

## Update Scripts

The **update script** lives in your **source repository** and is called with environment variables providing all necessary context.

### Environment Variables Available to Scripts

**See [complete example script](../../actions/create-pr/update-script-example.sh)** demonstrating all features.

All scripts receive these environment variables:

| Variable | Example | Description |
|----------|---------|-------------|
| `PRBUILDER_TAG` | `v10.3.2` | The release tag |
| `PRBUILDER_VERSION` | `10` | Parsed version (based on `version_strategy`) |
| `PRBUILDER_TARGET_DIR` | `/tmp/prbuilder-123` | Path to cloned target repository |
| `PRBUILDER_TARGET_REPO` | `rancher/rancher` | Target repository (owner/repo) |
| `PRBUILDER_TARGET_BRANCH` | `dev-v2.14` | Target branch being updated |
| `PRBUILDER_SOURCE_DIR` | `/github/workspace` | Source repository path |

### Example Update Script

```bash
#!/bin/bash
set -e

# All context is available via environment variables
echo "Updating $PRBUILDER_TARGET_REPO to $PRBUILDER_TAG"

# Update Chart.yaml in target repo
sed -i "s/version: .*/version: ${PRBUILDER_TAG#v}/" \
  "$PRBUILDER_TARGET_DIR/charts/my-chart/Chart.yaml"

# Update go dependency
cd "$PRBUILDER_TARGET_DIR"
go get github.com/myorg/myrepo@"$PRBUILDER_TAG"
go mod tidy

# Optionally call scripts in the target repo if needed
if [ -f "$PRBUILDER_TARGET_DIR/scripts/post-update.sh" ]; then
  "$PRBUILDER_TARGET_DIR/scripts/post-update.sh"
fi
```

### Single Script for Multiple Targets

Since all context is in environment variables, you can use one script for multiple targets:

```bash
#!/bin/bash
set -e

case "$PRBUILDER_TARGET_REPO" in
  "rancher/rancher")
    # Rancher-specific updates
    sed -i "s/backup_version=.*/backup_version=$PRBUILDER_TAG/" \
      "$PRBUILDER_TARGET_DIR/config.yaml"
    ;;
  "rancher/charts")
    # Charts-specific updates
    sed -i "s/version: .*/version: ${PRBUILDER_TAG#v}/" \
      "$PRBUILDER_TARGET_DIR/Chart.yaml"
    ;;
  *)
    echo "Unknown target: $PRBUILDER_TARGET_REPO"
    exit 1
    ;;
esac
```

### Wrapper for Existing Scripts

If you have existing scripts that expect positional arguments:

```bash
#!/bin/bash
# wrapper.sh - adapts prbuilder env vars to existing script
./my-existing-script.sh "$PRBUILDER_TAG" "$PRBUILDER_TARGET_DIR" "$PRBUILDER_TARGET_REPO"
```

## How It Works

For each target repository:

1. Parse version from tag using `version_strategy`
2. Resolve target branch using version mapping
3. Clone target repository on the specified branch
4. Create a new PR branch (`bump-to-{TAG}-{timestamp}`)
5. Run update script from source repo with environment variables
6. Check for changes via git status
7. Commit changes (author: github-actions[bot])
8. Push branch and create PR using `gh` CLI

If any target fails, the tool continues processing other targets.

## Examples

### Multi-Target Mode: CI/Automation

```bash
# Config with "targets:" (plural)
export GH_TOKEN="ghp_..."
prbuilder create-prs --tag v10.3.2
```

### Multi-Target Mode: Dry Run

```bash
# Config with "targets:" (plural)
prbuilder create-prs --tag v10.3.2 --dry-run
```

### Single-Target Mode: Local with Existing Clone

Test your update scripts with a repository you already have cloned:

```bash
# Config must use "target:" (singular)
# 1. Navigate to your cloned repository
cd ~/repos/rancher

# 2. Make sure you're on the right branch
git checkout dev-v2.14

# 3. Run prbuilder pointing to current directory
export GH_TOKEN="ghp_..."
prbuilder create-prs \
  --tag v10.3.2 \
  --config ~/source-repo/.github/pr-consumer-config.yml \
  --target-dir . \
  --dry-run

# 4. Review the changes
git diff

# 5. If changes look good, run without dry-run to create the PR
prbuilder create-prs \
  --tag v10.3.2 \
  --config ~/source-repo/.github/pr-consumer-config.yml \
  --target-dir .
```

### Single-Target Mode: Custom Remote Name

If your repository uses a different remote name:

```bash
# Config must use "target:" (singular)
cd ~/repos/rancher

prbuilder create-prs \
  --tag v10.3.2 \
  --config ~/source/.github/pr-consumer-config.yml \
  --target-dir . \
  --remote upstream
```

### Single-Target Mode: Auto-Clone

Let prbuilder clone the repository for you:

```bash
# Config must use "target:" (singular)
export GH_TOKEN="ghp_..."
prbuilder create-prs --tag v10.3.2 --config .github/pr-consumer-config.yml --dry-run
```

### Single Release Line (All Versions → Same Branch)

For repos where all versions go to the same branch:

```yaml
version_strategy: major
publishing_rules:
  "*": "main"  # All versions (0.x, 1.x, 2.x, etc.) go to main
targets:
  - repo: "myorg/consumer"
    update_script_path: "./scripts/bump.sh"
```

```bash
# Any tag will create a PR to main branch
prbuilder create-prs --tag v0.5.2
prbuilder create-prs --tag v1.0.0
prbuilder create-prs --tag v2.3.1  # All go to main
```

### Multiple Target Branches

Create PRs to multiple branches for a single release:

```yaml
publishing_rules:
  "10": ["dev-v2.14", "release-v2.14"]
targets:
  - repo: "myorg/consumer"
    update_script_path: "./scripts/bump.sh"
```

```bash
# Creates 2 PRs: one to dev-v2.14, one to release-v2.14
prbuilder create-prs --tag v10.3.2
```

### GitHub Actions (CI Mode)

```yaml
- name: Create PRs in All Consumer Repos
  env:
    GH_TOKEN: ${{ secrets.PAT_TOKEN }}
    TAG: ${{ github.ref_name }}
  run: prbuilder create-prs
```

### Checking Your Config Mode

To see which mode your config uses:

```bash
# Single-target config will support --target-dir
# Multi-target config will reject --target-dir with an error message
prbuilder create-prs --tag v10.0.0 --config config.yml --target-dir /tmp/test --dry-run
```

## Exit Codes

- `0` - Success (at least one PR created or no errors occurred)
- `1` - All targets failed or critical error (config parse, missing token)

## Requirements

- Go 1.21 or later
- `gh` CLI tool (for cloning and creating PRs)
- `git` command line tool
- GitHub Personal Access Token with `repo` scope

## Integration with GitHub Actions

When running in GitHub Actions, `prbuilder` automatically:
- Reads configuration from environment variables
- Writes PR URLs to `$GITHUB_OUTPUT` in JSON format
- Uses github-actions[bot] as the commit author

## Troubleshooting

### "GH_TOKEN or GITHUB_TOKEN environment variable is required"

Set the `GH_TOKEN` or `GITHUB_TOKEN` environment variable with a GitHub Personal Access Token.

### "No branch mapping found for version X"

Ensure your `publishing_rules` includes an entry for the extracted version. Check your `version_strategy` setting.

### "Failed to clone repository"

- Verify your PAT has access to the target repository
- Ensure the target branch exists
- Check the repository name format (`owner/repo`)

### "Update script not found"

- Ensure the script path is relative to your source repo root
- Verify the script is committed to your repository
- Check the script path starts with `./`

## License

Apache 2.0

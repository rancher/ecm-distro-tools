# Send PR to Consumer

This GitHub Action automates the process of creating PRs in downstream/consumer repositories when your project releases a new version. It's designed for scenarios where multiple projects from the same team work together and need coordinated version updates.

**Implementation:** This action uses the [`prbuilder create-prs`](/cmd/prbuilder) Go CLI tool. See the [prbuilder documentation](/cmd/prbuilder/README.md) for:
- **Standalone usage** in CI/automation
- **Local testing** with existing clones (single-target mode)

## Use Case

When **Project A** (a library/component) is consumed by **Project B** (the application), releasing a new version of Project A should automatically create a PR in Project B to update the dependency.

For example:
- `rancher-backup` releases version `v10.3.2`
- This action automatically creates a PR in `rancher/rancher` to bump the dependency
- The PR targets the appropriate branch based on version mapping (e.g., version 10.x → branch `dev-v2.14`)

## How It Works

The action relies on two key concepts:

1. **Version-to-Branch Mapping**: Maps your project's versions to specific branches in consumer repos
2. **Update Scripts**: Defines how to perform the bump and any post-update tasks

### Workflow

```
┌─────────────────────────────────────────────────────────────┐
│ Source Repo (e.g., rancher-backup)                          │
│                                                              │
│  1. New tag created (v10.3.2)                               │
│  2. Action triggered                                        │
│  3. Parse tag → extract version (10)                        │
│  4. Read config → find target branches                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ For each target repo:                                        │
│                                                              │
│  5. Clone target repo on appropriate branch                 │
│  6. Run update script (with environment variables)          │
│  7. Commit changes                                          │
│  8. Create PR                                               │
└─────────────────────────────────────────────────────────────┘
```

## Quick Links

- 📋 **[Example Config](config-example.yml)** - Complete configuration example
- 📖 **[Standalone Tool Docs](/cmd/prbuilder/README.md)** - Use prbuilder outside of GitHub Actions

## Setup

### 1. Create a Config File

Create a file in your repository (e.g., `.github/pr-consumer-config.yml`).

**Single-target mode** - Use `target:` (singular) for repos that update one downstream repository:

```yaml
# How to parse version from tags
version_mapping_type: major  # "major" or "major.minor"

# Map your versions to target branches
version_branch_map:
  "11": "dev-v2.15"
  "10": "dev-v2.14"
  "9": "dev-v2.13"

# Single target repository (note: "target" singular)
target:
  repo: "rancher/rancher"
  update_script: "./scripts/bump-rancher.sh"
```

**Multi-target mode** - Use `targets:` (plural) for repos that update multiple downstream repositories:

```yaml
# How to parse version from tags
version_mapping_type: major

# Map your versions to target branches
version_branch_map:
  "11": "dev-v2.15"
  "10": "dev-v2.14"
  "9": "dev-v2.13"

# Multiple target repositories (note: "targets" plural)
targets:
  - repo: "rancher/rancher"
    update_script: "./scripts/bump-rancher.sh"

  - repo: "rancher/charts"
    # Override global mapping for this specific target
    version_branch_map:
      "11": "release-v2.15"
      "10": "release-v2.14"
    update_script: "./scripts/bump-charts.sh"
```

### 2. Create Update Script

The **update script** lives in your **source repo** and performs the actual version bump in the target repository. It receives all necessary context via environment variables (see [Update Script Environment Variables](#update-script-environment-variables) below).

**Your script should:**
- Use the provided environment variables to locate and modify files in the target repository
- Exit with code `0` on success (any changes will be committed and PR created)
- Exit with non-zero code on failure (no PR will be created)
- Be executable and include error handling (`set -e`)

**Common patterns:**

```bash
#!/bin/bash
set -e  # Exit on error

# Update dependency versions
cd "$PRBUILDER_TARGET_DIR"
go get github.com/your-org/your-repo@"$PRBUILDER_TAG"
go mod tidy

# Update configuration files
sed -i "s/VERSION=.*/VERSION=${PRBUILDER_TAG}/" "$PRBUILDER_TARGET_DIR/Makefile"

# Branch-specific logic
case "$PRBUILDER_TARGET_BRANCH" in
  release-*)
    echo "Production config for release branch"
    ;;
  dev-*)
    echo "Development config for dev branch"
    ;;
esac

# Call target repo's scripts if they exist
if [ -f "$PRBUILDER_TARGET_DIR/scripts/post-update.sh" ]; then
  "$PRBUILDER_TARGET_DIR/scripts/post-update.sh" "$PRBUILDER_TAG"
fi
```

### 3. Add to Your Release Workflow

Add this action to your release workflow (`.github/workflows/release.yml`):

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # ... your existing release steps ...

      - name: Create PRs in Consumer Repos
        uses: SUSE/create-pr-action@v1
        with:
          tag: ${{ github.ref_name }}
          github-token: ${{ secrets.PAT_TOKEN }}
          config-file: .github/pr-consumer-config.yml
```

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `tag` | Yes | - | The tag that was released (e.g., `v10.3.2`) |
| `github-token` | Yes | - | GitHub token with permissions to create PRs in target repos. **Note:** Must be a PAT with `repo` scope, not `GITHUB_TOKEN` |
| `config-file` | No | `.github/pr-consumer-config.yml` | Path to config file |
| `dry-run` | No | `false` | If `true`, shows what would happen without creating PRs |

## Outputs

| Output | Description |
|--------|-------------|
| `prs-created` | JSON array of PR URLs that were created |

## Config File Reference

### `version_mapping_type`

How to extract version from tags:
- `major`: `v10.3.2` → `10`
- `major.minor`: `v10.3.2` → `10.3`

### `version_branch_map`

Global mapping of versions to target branches. Can be overridden per target.

**Supports:**
- Single branch (string): `"10": "dev-v2.14"`
- Multiple branches (array): `"10": ["dev-v2.14", "release-v2.14"]` - Creates separate PRs to each branch
- Wildcard fallback: `"*": "main"` - Matches any version not explicitly mapped (useful for single release lines)

```yaml
version_branch_map:
  "11": "dev-v2.15"
  "10": ["dev-v2.14", "release-v2.14"]  # Creates 2 PRs
  "*": "main"  # Fallback for all other versions
```

### `target` vs `targets`

**Single-target mode** - Use `target:` (singular):

```yaml
target:
  repo: "org/repo-name"           # Required: target repository
  update_script: "./path/to/script.sh"  # Required: script (receives env vars)
  version_branch_map:             # Optional: override global mapping
    "10": "custom-branch"
```

**Multi-target mode** - Use `targets:` (plural):

```yaml
targets:
  - repo: "org/repo-name-1"
    update_script: "./path/to/script.sh"
  - repo: "org/repo-name-2"
    update_script: "./path/to/script2.sh"
    version_branch_map:
      "10": "custom-branch"
```

**Note:** You cannot use both `target` and `targets` in the same config file.

## Update Script Environment Variables

All update scripts receive these environment variables:

| Variable | Example | Description |
|----------|---------|-------------|
| `PRBUILDER_TAG` | `v10.3.2` | The release tag |
| `PRBUILDER_VERSION` | `10` | Parsed version based on `version_mapping_type` (e.g., `10` for major, `10.3` for major.minor) |
| `PRBUILDER_TARGET_DIR` | `/tmp/prbuilder-123` | Absolute path to cloned target repository (working directory for modifications) |
| `PRBUILDER_TARGET_REPO` | `rancher/rancher` | Target repository in `owner/repo` format (useful for multi-repo scripts) |
| `PRBUILDER_TARGET_BRANCH` | `dev-v2.14` | Target branch being updated (useful for branch-specific logic) |
| `PRBUILDER_SOURCE_DIR` | `/github/workspace` | Absolute path to source repository (your repo with the update script) |

**Usage tips:**
- Validate required env vars at the start of your script
- Use `$PRBUILDER_TARGET_DIR` as the base path for all file modifications
- Use `$PRBUILDER_TARGET_REPO` to implement repo-specific logic in a shared script
- Use `$PRBUILDER_TARGET_BRANCH` for branch-specific configurations (dev vs release)
- Reference source repo files via `$PRBUILDER_SOURCE_DIR` if needed
- Remove `v` prefix from tag: `VERSION="${PRBUILDER_TAG#v}"` → `10.3.2`

## Authentication

The action requires a GitHub Personal Access Token (PAT) with:
- `repo` scope
- Write access to target repositories

The default `GITHUB_TOKEN` does NOT work for creating PRs in other repositories.

### Creating a PAT

1. Go to GitHub Settings → Developer settings → Personal access tokens
2. Generate new token (classic)
3. Select `repo` scope
4. Add token as a secret in your repository (e.g., `PAT_TOKEN`)

## Examples

### Example 1: Simple Major Version Mapping

Config:
```yaml
version_mapping_type: major
version_branch_map:
  "10": "dev-v2.14"
  "9": "dev-v2.13"

targets:
  - repo: "myorg/consumer"
    update_script: "./scripts/bump.sh"
```

Tag `v10.3.2` → Creates PR in `myorg/consumer` targeting branch `dev-v2.14`

### Example 2: Multiple Targets

Config:
```yaml
version_mapping_type: major
version_branch_map:
  "10": "main"

targets:
  - repo: "myorg/app1"
    update_script: "./scripts/bump-app1.sh"

  - repo: "myorg/app2"
    update_script: "./scripts/bump-app2.sh"
    post_update_script: "./scripts/validate.sh"

  - repo: "myorg/app3"
    version_branch_map:
      "10": "develop"  # Override for this target
    update_script: "./scripts/bump-app3.sh"
```

Tag `v10.1.0` → Creates 3 PRs (one in each target repo)

### Example 3: Dry Run

```yaml
- name: Test PR Creation
  uses: rancher/ecm-distro-tools/actions/create-pr@<commit-sha>
  with:
    tag: v10.3.2
    github-token: ${{ secrets.PAT_TOKEN }}
    dry-run: true
```

Shows what would happen without actually creating PRs.

## Standalone Usage

This action uses the `prbuilder create-prs` command. The operating mode is determined by your config file:

- **Multi-target mode** - Config uses `targets:` (plural) - Processes all targets automatically
- **Single-target mode** - Config uses `target:` (singular) - Can use `--target-dir` for local testing

See the [prbuilder documentation](/cmd/prbuilder/README.md) for standalone usage, including local testing with already-cloned repositories.

## Troubleshooting

### "GH_TOKEN or GITHUB_TOKEN environment variable is required"

The action requires a GitHub Personal Access Token. Make sure you've passed it via the `github-token` input.

### "No branch mapping found for version X"

Make sure your `version_branch_map` includes an entry for the version extracted from your tag.

If using `version_mapping_type: major` and tag `v10.3.2`, ensure you have a mapping for `"10"`.

### "Failed to clone repository"

Check that:
1. Your PAT has access to the target repository
2. The target branch exists in the target repository
3. The repository name is correct (format: `owner/repo`)

### "Update script not found"

Ensure:
1. The script path in config is relative to your source repo root
2. The script file is committed to your repository
3. The script path starts with `./`

### "No changes detected"

The update script didn't modify any files in the target repository. Check:
1. The script is running correctly (add debug output)
2. File paths in the script are correct
3. The script has necessary permissions

## Contributing

Issues and pull requests are welcome at [github.com/SUSE/create-pr-action](https://github.com/SUSE/create-pr-action).

## License

MIT

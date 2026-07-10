#!/bin/bash
# Example update script for prbuilder
#
# This script demonstrates how to use the environment variables provided by prbuilder.
# Copy this file to your source repository and customize it for your needs.
#
# Location: This script should be in your SOURCE repository
# Config reference: update_script: "./scripts/update-example.sh"

set -e  # Exit on error
set -u  # Exit on undefined variable

# =============================================================================
# ENVIRONMENT VARIABLES PROVIDED BY PRBUILDER
# =============================================================================
#
# PRBUILDER_TAG          - The release tag (e.g., "v10.3.2")
# PRBUILDER_VERSION      - Parsed version based on version_mapping_type
#                          e.g., "10" for major, "10.3" for major.minor
# PRBUILDER_TARGET_DIR   - Absolute path to the cloned target repository
#                          e.g., "/tmp/prbuilder-123/rancher-rancher"
# PRBUILDER_TARGET_REPO  - Target repository in "owner/repo" format
#                          e.g., "rancher/rancher"
# PRBUILDER_TARGET_BRANCH - Target branch being updated
#                           e.g., "dev-v2.14"
# PRBUILDER_SOURCE_DIR   - Absolute path to the source repository
#                          e.g., "/github/workspace"

# =============================================================================
# VALIDATION
# =============================================================================

# Validate that all required environment variables are set
if [ -z "${PRBUILDER_TAG:-}" ] || \
   [ -z "${PRBUILDER_VERSION:-}" ] || \
   [ -z "${PRBUILDER_TARGET_DIR:-}" ] || \
   [ -z "${PRBUILDER_TARGET_REPO:-}" ] || \
   [ -z "${PRBUILDER_TARGET_BRANCH:-}" ] || \
   [ -z "${PRBUILDER_SOURCE_DIR:-}" ]; then
    echo "ERROR: Required PRBUILDER_* environment variables not set"
    echo "This script must be called by prbuilder"
    exit 1
fi

# =============================================================================
# EXAMPLE 1: Basic file updates
# =============================================================================

echo "Updating $PRBUILDER_TARGET_REPO to $PRBUILDER_TAG (branch: $PRBUILDER_TARGET_BRANCH)"

# Update a version file
if [ -f "$PRBUILDER_TARGET_DIR/version.txt" ]; then
    echo "Updating version.txt..."
    echo "$PRBUILDER_TAG" > "$PRBUILDER_TARGET_DIR/version.txt"
fi

# Update Chart.yaml (Helm charts)
if [ -f "$PRBUILDER_TARGET_DIR/Chart.yaml" ]; then
    echo "Updating Chart.yaml..."
    # Remove 'v' prefix for semantic version
    VERSION_NO_V="${PRBUILDER_TAG#v}"
    sed -i.bak "s/^version: .*/version: $VERSION_NO_V/" "$PRBUILDER_TARGET_DIR/Chart.yaml"
    rm -f "$PRBUILDER_TARGET_DIR/Chart.yaml.bak"  # Remove backup file
fi

# =============================================================================
# EXAMPLE 2: Update Go module dependencies
# =============================================================================

# Change to target directory for go commands
cd "$PRBUILDER_TARGET_DIR"

# Update go.mod if it exists
if [ -f "go.mod" ]; then
    echo "Updating Go dependencies..."

    # Example: Update a specific module to the new tag
    # go get github.com/your-org/your-module@"$PRBUILDER_TAG"
    # go mod tidy

    echo "  (go.mod updates would go here)"
fi

# =============================================================================
# EXAMPLE 3: Handle different target repositories differently
# =============================================================================

# You can use a single script for multiple targets by checking PRBUILDER_TARGET_REPO
case "$PRBUILDER_TARGET_REPO" in
    "rancher/rancher")
        echo "Applying rancher-specific updates..."
        # Update rancher-specific files
        # sed -i "s/BACKUP_VERSION=.*/BACKUP_VERSION=$PRBUILDER_TAG/" \
        #   "$PRBUILDER_TARGET_DIR/pkg/settings/settings.go"
        ;;

    "rancher/charts")
        echo "Applying charts-specific updates..."
        # Update chart dependencies
        # sed -i "s/version: .*/version: ${PRBUILDER_TAG#v}/" \
        #   "$PRBUILDER_TARGET_DIR/charts/rancher-backup/Chart.yaml"
        ;;

    "rancher/rke2-charts")
        echo "Applying rke2-charts-specific updates..."
        # Update RKE2 chart references
        ;;

    *)
        echo "WARNING: No specific handling for $PRBUILDER_TARGET_REPO"
        echo "Applying generic updates..."
        ;;
esac

# =============================================================================
# EXAMPLE 4: Call scripts from the target repository
# =============================================================================

# Sometimes the target repository has its own scripts that need to run
# after the version update (e.g., code generation, validation)

if [ -f "$PRBUILDER_TARGET_DIR/scripts/post-update.sh" ]; then
    echo "Running target repository's post-update script..."
    chmod +x "$PRBUILDER_TARGET_DIR/scripts/post-update.sh"

    # Pass relevant info to the target script if needed
    "$PRBUILDER_TARGET_DIR/scripts/post-update.sh" "$PRBUILDER_TAG"
fi

# =============================================================================
# EXAMPLE 5: Use source repository files/scripts
# =============================================================================

# You can reference files from your source repository
if [ -f "$PRBUILDER_SOURCE_DIR/scripts/helper.sh" ]; then
    echo "Using helper script from source repository..."
    # source "$PRBUILDER_SOURCE_DIR/scripts/helper.sh"
    # run_helper_function "$PRBUILDER_TAG"
fi

# =============================================================================
# EXAMPLE 6: Branch-specific logic
# =============================================================================

# Different branches might need different handling
case "$PRBUILDER_TARGET_BRANCH" in
    dev-*)
        echo "Development branch: enabling debug features"
        # Enable debug/development features
        ;;
    release-*)
        echo "Release branch: production configuration"
        # Use production settings
        ;;
    main|master)
        echo "Main branch: standard configuration"
        ;;
esac

# =============================================================================
# EXAMPLE 7: Version-specific logic
# =============================================================================

# Use the parsed version for logic
MAJOR_VERSION="${PRBUILDER_VERSION%%.*}"  # Extract major version number

if [ "$MAJOR_VERSION" -ge 10 ]; then
    echo "Version 10+: using new API format"
    # Use new configuration format
else
    echo "Version <10: using legacy API format"
    # Use old configuration format
fi

# =============================================================================
# EXAMPLE 8: Generate or update multiple files
# =============================================================================

# Create/update a metadata file with build information
cat > "$PRBUILDER_TARGET_DIR/build-metadata.json" <<EOF
{
  "source_version": "$PRBUILDER_TAG",
  "parsed_version": "$PRBUILDER_VERSION",
  "target_branch": "$PRBUILDER_TARGET_BRANCH",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

# =============================================================================
# VALIDATION & TESTING
# =============================================================================

# Optionally run tests or validation before committing
if [ -f "$PRBUILDER_TARGET_DIR/Makefile" ]; then
    echo "Running validation..."
    # make -C "$PRBUILDER_TARGET_DIR" validate || true
fi

# =============================================================================
# SUMMARY
# =============================================================================

echo ""
echo "✓ Update completed successfully"
echo "  Tag:           $PRBUILDER_TAG"
echo "  Version:       $PRBUILDER_VERSION"
echo "  Target:        $PRBUILDER_TARGET_REPO"
echo "  Branch:        $PRBUILDER_TARGET_BRANCH"
echo ""

# Exit successfully - prbuilder will commit and create the PR
exit 0

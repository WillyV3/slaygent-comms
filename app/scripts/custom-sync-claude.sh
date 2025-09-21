#!/bin/bash
# Sync custom registry content to CLAUDE.md files
# Constitution: No fallbacks, fail fast and clearly

set -e  # Exit on any error

# Accept custom content as first parameter
CUSTOM_CONTENT="$1"
if [[ -z "$CUSTOM_CONTENT" ]]; then
    echo "ERROR: Custom content must be provided as first parameter"
    echo "Usage: $0 '<custom_registry_content>'"
    exit 1
fi

REGISTRY_PATH="$HOME/.slaygent/registry.json"
MARKER_START="<!-- SLAYGENT-REGISTRY-START -->"
MARKER_END="<!-- SLAYGENT-REGISTRY-END -->"

# Check if registry exists
if [[ ! -f "$REGISTRY_PATH" ]]; then
    echo "ERROR: Registry not found at $REGISTRY_PATH"
    exit 1
fi

# Find all CLAUDE.md and AGENTS.md files using fd
if ! command -v fd >/dev/null 2>&1; then
    echo "fd command not found. Installing..."

    # Detect OS and install fd
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if command -v brew >/dev/null 2>&1; then
            brew install fd
        else
            echo "ERROR: Homebrew not found on macOS. Install from: https://brew.sh"
            exit 1
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command -v apt >/dev/null 2>&1; then
            sudo apt update && sudo apt install -y fd-find
            # Create symlink if fdfind exists but fd doesn't
            if command -v fdfind >/dev/null 2>&1 && ! command -v fd >/dev/null 2>&1; then
                mkdir -p "$HOME/.local/bin"
                ln -sf "$(which fdfind)" "$HOME/.local/bin/fd"
                export PATH="$HOME/.local/bin:$PATH"
            fi
        elif command -v dnf >/dev/null 2>&1; then
            sudo dnf install -y fd-find
        elif command -v yum >/dev/null 2>&1; then
            sudo yum install -y fd-find
        elif command -v pacman >/dev/null 2>&1; then
            sudo pacman -S --noconfirm fd
        elif command -v zypper >/dev/null 2>&1; then
            sudo zypper install -y fd
        else
            echo "ERROR: No supported package manager found (apt, dnf, yum, pacman, zypper)"
            exit 1
        fi
    else
        echo "ERROR: Unsupported operating system: $OSTYPE"
        exit 1
    fi

    # Verify installation
    if ! command -v fd >/dev/null 2>&1; then
        echo "ERROR: fd installation failed"
        exit 1
    fi

    echo "✓ fd installed successfully"
fi

# Find all CLAUDE.md and AGENTS.md files from home directory
CLAUDE_FILES=$(fd -t f "^(CLAUDE|AGENTS)\.md$" ~ 2>/dev/null)

if [[ -z "$CLAUDE_FILES" ]]; then
    echo "No CLAUDE.md or AGENTS.md files found"
    exit 0
fi

# Count files and show them
FILE_COUNT=$(echo "$CLAUDE_FILES" | wc -l)
echo "Found $FILE_COUNT CLAUDE.md/AGENTS.md files:"
echo "$CLAUDE_FILES"
echo ""
echo "Are you sure you want to update all of them? (y/N): "
read -r CONFIRM

if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
    echo "Sync cancelled"
    exit 0
fi

echo "Starting sync..."

# Process each file
while IFS= read -r CLAUDE_FILE; do
    if [[ -z "$CLAUDE_FILE" ]]; then
        continue
    fi

    echo "SYNC: $CLAUDE_FILE"

    # Create backup
    cp "$CLAUDE_FILE" "$CLAUDE_FILE.backup"

    # Check if markers exist
    if grep -q "$MARKER_START" "$CLAUDE_FILE"; then
        echo "  → Updating existing registry section"
        # Replace content between markers using the custom content with markers
        # Create a temporary file with the custom content wrapped in markers
        TEMP_FILE=$(mktemp)
        echo "$MARKER_START" > "$TEMP_FILE"
        echo "$CUSTOM_CONTENT" >> "$TEMP_FILE"
        echo "$MARKER_END" >> "$TEMP_FILE"

        # Use sed to replace the content between markers
        sed -i.tmp "/$MARKER_START/,/$MARKER_END/{
            /$MARKER_START/r $TEMP_FILE
            /$MARKER_START/,/$MARKER_END/d
        }" "$CLAUDE_FILE"

        rm -f "$CLAUDE_FILE.tmp" "$TEMP_FILE"
    else
        echo "  → Adding new registry section"
        # Append the custom content with markers
        echo "" >> "$CLAUDE_FILE"
        echo "$MARKER_START" >> "$CLAUDE_FILE"
        echo "$CUSTOM_CONTENT" >> "$CLAUDE_FILE"
        echo "$MARKER_END" >> "$CLAUDE_FILE"
    fi

    echo "✓ Synced registry reference to $CLAUDE_FILE"
done <<< "$CLAUDE_FILES"

echo ""
echo "Registry sync complete!"
echo "All files have been updated with registry reference: $REGISTRY_PATH"
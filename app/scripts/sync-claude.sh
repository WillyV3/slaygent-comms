#!/bin/bash
# Sync registry reference to CLAUDE.md files
# Constitution: No fallbacks, fail fast and clearly

set -e  # Exit on any error

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
    echo "ERROR: fd command not found. Install with: brew install fd"
    exit 1
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
        # Replace content between markers using sed
        sed -i.tmp "/$MARKER_START/,/$MARKER_END/c\\
$MARKER_START\\
# Inter-Agent Communication\\
@$REGISTRY_PATH\\
\\
To send messages to other coding agents, use: \`msg <agent_name> \"<message>\"\`\\
Example: \`msg backend-dev \"Please update the API endpoint\"\`\\
\\
IMPORTANT: When responding to messages, always use the --from flag:\\
\`msg --from <your_agent_name> <target_agent> \"<response>\"\`\\
This ensures proper conversation logging and tracking.\\
\\
<!-- Registry automatically synced by slaygent-manager -->\\
$MARKER_END" "$CLAUDE_FILE"
        rm -f "$CLAUDE_FILE.tmp"
    else
        echo "  → Adding new registry section"
        # Append new section
        cat >> "$CLAUDE_FILE" << EOF

$MARKER_START
# Inter-Agent Communication
@$REGISTRY_PATH

To send messages to other coding agents, use: \`msg <agent_name> "<message>"\`
Example: \`msg backend-dev "Please update the API endpoint"\`

IMPORTANT: When responding to messages, always use the --from flag:
\`msg --from <your_agent_name> <target_agent> "<response>"\`
This ensures proper conversation logging and tracking.

<!-- Registry automatically synced by slaygent-manager -->
$MARKER_END
EOF
    fi

    echo "✓ Synced registry reference to $CLAUDE_FILE"
done <<< "$CLAUDE_FILES"

echo ""
echo "Registry sync complete!"
echo "All files have been updated with registry reference: $REGISTRY_PATH"
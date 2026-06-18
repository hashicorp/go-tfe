#!/usr/bin/env bash
# Copyright IBM Corp. 2018, 2026
# SPDX-License-Identifier: MPL-2.0

set -e

HOOK_PATH=.git/hooks/pre-commit

mkdir -p v2/openapi
mkdir -p v2/internal/api

echo "This script will copy the internal-beta API spec from ../atlas and build the SDK."
echo "The results should NOT be committed to the public repo and should be used for internal development only."

read -r -p "Do you want to continue? [y/N] " response
case "$response" in
    [yY][eE][sS]|[yY]) 
        cp ../atlas/openapi/bundled/hcpt_v2_internal_beta.json v2/openapi/spec.json
        ;;
    *)
        echo "Canceled."
        exit 1
        ;;
esac

cat > "$HOOK_PATH" <<'EOF'
#!/usr/bin/env bash

set -e

if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "Commit blocked: 'make api_internal' used"
    echo "----------------------------------------"
    echo "If you want to commit changes, delete this hook using"
    echo ""
    echo "rm .git/hooks/pre-commit"
    echo ""
    echo "and try again. Do not push internal-beta API changes to"
    echo "the public repository."
    echo ""
    echo "Otherwise, use 'git reset --hard' to discard all changes."
    exit 1
fi
EOF

chmod +x "$HOOK_PATH"

echo "Installed pre-commit hook at $HOOK_PATH"

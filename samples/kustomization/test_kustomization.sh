#!/bin/bash
# cSpell: words toplevel appstage
set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

GIT_ROOT=$(git rev-parse --show-toplevel)

"$GIT_ROOT/karmafun" build appstage-00-bootstrap

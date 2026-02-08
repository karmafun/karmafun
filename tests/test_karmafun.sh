#!/bin/bash

# DEPENDENCIES
# kustomize
# yq

#set -uexo pipefail
set -e pipefail

trap "find . -type d -name 'applications' -exec rm -rf {} +" EXIT

# cSpell: disable
SOPS_AGE_KEY=$(cat - <<EOF
# created: 2023-01-19T19:41:45Z
# public key: age166k86d56ejs2ydvaxv2x3vl3wajny6l52dlkncf2k58vztnlecjs0g5jqq
AGE-SECRET-KEY-15RKTPQCCLWM7EHQ8JEP0TQLUWJAECVP7332M3ZP0RL9R7JT7MZ6SY79V8Q
EOF
)
SOPS_RECIPIENT="age166k86d56ejs2ydvaxv2x3vl3wajny6l52dlkncf2k58vztnlecjs0g5jqq"
# cSpell: enable
export SOPS_AGE_KEY
export SOPS_RECIPIENT

find . -mindepth 1 -maxdepth 1 -type d | while read -r d; do
    echo "Running Test in $d..."
    cd "$d"
    rm -rf applications
    cp -r original applications
    echo "  > Performing kustomizations..."
    kustomize fn run --enable-exec --fn-path functions applications
    if [ -d expected ]; then
        for f in applications/*; do
            b=$(basename "$f")
            echo "  > Checking $b..."
            diff <(yq eval -P "expected/$b") <(yq eval -P "applications/$b")
        done
    else
        echo "  > No expected result. Skipping check"
    fi
    cd ..
done
echo "Done ok 🎉"

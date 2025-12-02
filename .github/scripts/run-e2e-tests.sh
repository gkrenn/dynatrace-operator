#!/usr/bin/env bash

set -eu -o pipefail

echo "Preparing test tenant secrets..."

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "Executing prepare-e2e-secrets.sh from '$__dir'"
bash ${__dir}/prepare-e2e-secrets.sh

echo hello from run-e2e-tests.sh
ls -lah
pwd

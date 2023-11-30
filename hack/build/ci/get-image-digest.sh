#!/bin/bash

readonly online="${1}"

if [ -z "$2" ]
then
  echo "Usage: $0 <fetch_from_registry>"
  exit 1
fi

digest=$(skopeo inspect docker-daemon:"${IMAGE}" --format "{{.Digest}}")
echo "${DIGEST_KEY}=${digest}">> "$GITHUB_OUTPUT"

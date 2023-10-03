#!/bin/bash

readonly PATH_TO_HELM_CHART="${1}"
readonly REGISTRY_URL="${2}"

# Run the command and capture stdout and stderr separately
output=$(helm push "${PATH_TO_HELM_CHART}" oci://registry.hub.docker.com/gabrielkrenn390 2>&1)
exit_status=$?

if [ $exit_status -eq 0 ]; then
  # Command succeeded, extract the digest
  digest=$(echo "$output" | awk '/Digest:/ {print $2}')
  # write to digest to github actions output 
  echo "digest=$digest"
else
  # Command failed, print the error message and exit with the error code
  echo "Command failed with exit status $exit_status. Error: $output"
  exit $exit_status
fi

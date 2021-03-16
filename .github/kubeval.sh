#!/bin/bash

set -euxo pipefail

KUBEVAL_VERSION="0.15.0"
SCHEMA_LOCATION="https://raw.githubusercontent.com/instrumenta/kubernetes-json-schema/master/"

KUBEVAL_PATH="kubeval"

if ! command -v kubeval &> /dev/null; then
  # install kubeval
  curl --silent --show-error --fail --location --output /tmp/kubeval.tar.gz https://github.com/instrumenta/kubeval/releases/download/"${KUBEVAL_VERSION}"/kubeval-linux-amd64.tar.gz
  tar -xf /tmp/kubeval.tar.gz kubeval
  KUBEVAL_PATH="./kubeval"
fi

# validate charts
helm template helm/beacon/ | $KUBEVAL_PATH --strict --ignore-missing-schemas --kubernetes-version "${KUBERNETES_VERSION#v}" --schema-location "${SCHEMA_LOCATION}"
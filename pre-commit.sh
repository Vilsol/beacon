#!/usr/bin/env bash

set -euxo pipefail

for CHART_DIR in ./helm/*; do
  helm dependency update "${CHART_DIR}"
done

ct lint --all --config .github/ct.yaml

helm-docs

KUBERNETES_VERSION=1.20.4 ./.github/kubeval.sh

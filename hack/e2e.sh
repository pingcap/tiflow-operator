#!/usr/bin/env bash

# Copyright 2022 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

#
# E2E entrypoint script.
#


set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

# check bash version
BASH_MAJOR_VERSION=$(echo "$BASH_VERSION" | cut -d '.' -f 1)
# we need bash version >= 4
if [ $BASH_MAJOR_VERSION -lt 4 ]
then
  echo "error: e2e.sh could not work with bash version earlier than 4 for now, please upgrade your bash"
  exit 1
fi

function usage() {
    cat <<'EOF'
This script is entrypoint to run e2e tests.

Usage: hack/e2e.sh [-h] -- [extra test args]

    -h      show this message and exit

Environments:

    PROVIDER              Kubernetes provider, e.g. kind, gke, eks, defaults: kind
    DOCKER_REPO           docker image repo
    IMAGE_TAG             image tag
    CLUSTER               the name of e2e cluster, defaults: tidb-operator

EOF

}

while getopts "h?" opt; do
    case "$opt" in
    h|\?)
        usage
        exit 0
        ;;
    esac
done

if [ "${1:-}" == "--" ]; then
    shift
fi

PROVIDER=${PROVIDER:-kind}
DOCKER_REPO=${DOCKER_REPO:-localhost:5000/pingcap}
IMAGE_TAG=${IMAGE_TAG:-latest}
CLUSTER=${CLUSTER:-tiflow-operator}

echo "starting e2e test at $(date -Iseconds -u)"
echo "PROVIDER: $PROVIDER"
echo "DOCKER_REPO: $DOCKER_REPO"
echo "IMAGE_TAG: $IMAGE_TAG"
echo "CLUSTER: $CLUSTER"


create_cluster() {
  $KIND_BIN create cluster --name "${CLUSTER}"
}

install_operator() {
  # Can't seem to figure out how to leverage the stamp variables here. So for
  # now I've added a defined make variable which can be used for substitution
  # in //config/default/BUILD.bazel.
  K8S_CLUSTER="${CLUSTER}" \
    DEV_REGISTRY="${DOCKER_REPO}" \
    make deploy
}

wait_for_ready() {
  echo "Waiting for deployment to be available..."
  kubectl wait \
    --for=condition=Available \
    --timeout=2m \
    -n tiflow-operator-system \
    deploy/tiflow-operator-controller-manager
}

case "${1:-}" in
  run)
  hack::ensure_kind
  hack::ensure_kubectl
  create_cluster
  install_operator
  wait_for_ready;;
  clean)
    $KIND_BIN delete cluster --name "${CLUSTER}";;
  *) echo "Unknown command. Usage $0 <run|clean>" 1>&2; exit 1;;
esac

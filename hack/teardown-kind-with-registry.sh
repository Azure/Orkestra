#!/bin/sh

set -o errexit

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-orkestra}"
reg_name='kind-registry'

# Delete registry container if it exists
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" == 'true' ]; then
  cid="$(docker inspect -f '{{.ID}}' "${reg_name}")"
  echo "> Stopping and deleting Kind Registry container ..."
  docker stop $cid >/dev/null
  docker rm $cid >/dev/null
else
  echo "Kind Registry doesn't exist, moving on ..."
fi

kind delete cluster --name=$KIND_CLUSTER_NAME 2>&1

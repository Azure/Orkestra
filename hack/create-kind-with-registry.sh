#!/bin/sh
#
# Adapted from:
# https://kind.sigs.k8s.io/docs/user/local-registry/

set -o errexit

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-orkestra}"
reg_name='kind-registry'
reg_port='5000'

# Create registry container unless it already exists
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  echo "> Creating Kind Registry container ..."
  docker run \
    -d --restart=always -p "127.0.0.1:${reg_port}:5000" --name "${reg_name}" \
    registry:2
else
  echo "> Kind Registry container already exists, moving on ..."
fi

# Create kind cluster with the local registry enabled in containerd unless it already exists
if kind get clusters | grep -q ^"${KIND_CLUSTER_NAME}"$ ; then
  echo "> Kind cluster already exists, moving on ..."
else
  kind create cluster --name "${KIND_CLUSTER_NAME}" --config=.kind-cluster.yaml
fi

# connect the registry to the cluster network
# (the network may already be connected)
docker network connect "kind" "${reg_name}" || true

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

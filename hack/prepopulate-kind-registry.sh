#!/bin/sh

set -o errexit

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-orkestra}"
reg_name='kind-registry'

# Check if kind registry container exists.
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  echo "> Kind Registry container ${reg_name} doesn't exist ..."
  echo "> Stopping prepopulating kind registry ..."
  exit 0
fi

images_to_push=(
    "azureorkestra/orkestra:latest"
    "chartmuseum/chartmuseum:v0.12.0"
    "fluxcd/helm-controller:v0.9.0"
    "fluxcd/source-controller:v0.10.0"
    "quay.io/argoproj/argocli:v3.0.2"
    "quay.io/argoproj/workflow-controller:v3.0.2"
)

# Pull, tag, and push all images to local registry.
# Docker will pull images from remote only if one is not available locally or is not upto date.
# NOTE: We are pushing these images to local registry instead of loading to kind because
#       these images either have tag "latest" or have the image pull policy "Always". In both of
#       these cases, kind will pull new image even if one is available. So, we want kind to first
#       try pulling these images from local registry, and if one is not available, pull from
#       remote repository. 
for image in ${images_to_push[@]}; do
    echo ""
    echo "> 🔨  Pull, tag, and push ${image}"
    docker pull ${image}
    docker tag ${image} localhost:5000/${image}
    docker push localhost:5000/${image}
done

images_to_load=(
  "docker.io/istio/examples-bookinfo-details-v1:1.16.2"
  "docker.io/istio/examples-bookinfo-productpage-v1:1.16.2"
  "docker.io/istio/examples-bookinfo-ratings-v1:1.16.2"
  "docker.io/istio/examples-bookinfo-reviews-v1:1.16.2"
  "docker.io/datawire/aes:1.12.1"
  "prom/statsd-exporter:v0.8.1"
  "redis:5.0.1"
)

# Pull image if not available locally or is not upto date and then load to kind.
for image in ${images_to_load[@]}; do
    echo ""
    echo "> 🔨  Pull, load to kind ${image}"
    docker pull ${image}
    kind load docker-image ${image} --name ${KIND_CLUSTER_NAME}
done

#!/bin/sh
set -o errexit

reg_name='kind-registry'
reg_port='5000'

# connect the registry to the cluster network
docker network connect "kind" "${reg_name}" || true

# tell https://tilt.dev to use the registry
# https://docs.tilt.dev/choosing_clusters.html#discovering-the-registry
for node in $(./kind get nodes --name apicurio-cluster); do
  kubectl annotate node "${node}" "kind.x-k8s.io/registry=localhost:${reg_port}";
done
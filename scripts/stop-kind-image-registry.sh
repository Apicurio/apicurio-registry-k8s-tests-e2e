#!/bin/sh
set -o errexit

reg_name='kind-registry'

running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" == 'true' ]; then
  containerID="$(docker inspect -f '{{.Id}}' "${reg_name}" )"
  docker stop "${containerID}"
  docker rm "${containerID}"
fi
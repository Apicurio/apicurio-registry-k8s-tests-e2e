#!/bin/bash
set -e -a

make pull-apicurio-registry build-apicurio-registry

./scripts/install_kind.sh

make run-apicurio-ci

set +e +a 
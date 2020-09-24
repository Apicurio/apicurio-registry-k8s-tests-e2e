#!/bin/bash
set -e -a

./scripts/install_kind.sh

make run-operator-ci

set +e +a
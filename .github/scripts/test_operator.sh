#!/bin/bash
set -e -a

./scripts/setup-deps.sh

make pull-operator-repo

make run-operator-ci

set +e +a
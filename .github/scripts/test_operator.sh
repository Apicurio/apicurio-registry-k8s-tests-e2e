#!/bin/bash
set -e -a

./scripts/setup-deps.sh

make run-operator-ci

set +e +a
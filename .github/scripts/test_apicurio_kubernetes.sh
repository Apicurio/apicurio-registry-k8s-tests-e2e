#!/bin/bash
set -e -a

make pull-apicurio-registry build-apicurio-registry

./scripts/install_kind.sh

if [ "$E2E_APICURIO_TESTS_PROFILE" == "upgrade" ]
then
    E2E_APICURIO_TESTS_PROFILE=smoke
    make run-upgrade-ci
else
    make run-apicurio-ci
fi

set +e +a 
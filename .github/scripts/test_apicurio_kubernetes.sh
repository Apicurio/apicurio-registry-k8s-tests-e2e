#!/bin/bash
set -e -a

make pull-apicurio-registry build-apicurio-registry

./scripts/install_kind.sh

if [ "$E2E_APICURIO_TESTS_PROFILE" == "upgrade" ]
then
    E2E_APICURIO_TESTS_PROFILE=smoke
    make run-upgrade-ci
else
    if [ "$E2E_APICURIO_TESTS_PROFILE" == "clustered" ]
    then
        E2E_APICURIO_TESTS_PROFILE=acceptance
        KIND_CLUSTER_CONFIG=kind-config-big-cluster.yaml
        make run-apicurio-base-ci
        make run-clustered-tests
    else
        make run-apicurio-ci
    fi
fi

set +e +a 
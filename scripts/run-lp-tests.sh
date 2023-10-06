E2E_OLM_USE_DEFAULT_CATALOG_SOURCE=redhat-operators \
    E2E_SUITE_PROJECT_DIR=$(echo $PWD) \
    E2E_OLM_CATALOG_SOURCE_NAMESPACE=openshift-marketplace \
    E2E_OLM_CLUSTER_WIDE_OPERATORS_NAMESPACE=openshift-operators \
    E2E_OLM_PACKAGE_MANIFEST_NAME=service-registry-operator \
    go run github.com/onsi/ginkgo/ginkgo -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
    --junit-report=xunit-report.xml \
    --cover --trace --race --progress -v ./testsuite/olm -- -only-test-operator -disable-clustered-tests -enable-olm-advanced-tests -install-strimzi-olm
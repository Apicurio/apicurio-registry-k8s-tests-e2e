E2E_OLM_USE_DEFAULT_CATALOG_SOURCE=redhat-operators \
 OLM_CATALOG_SOURCE_NAMESPACE=openshift-marketplace \
 OLM_CLUSTER_WIDE_OPERATORS_NAMESPACE=openshift-operators \
 OLM_PACKAGE_MANIFEST_NAME=service-registry-operator make run-lp-tests
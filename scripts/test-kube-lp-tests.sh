
make kind-start kind-setup-olm

E2E_OLM_USE_DEFAULT_CATALOG_SOURCE=operatorhubio-catalog \
 OLM_PACKAGE_MANIFEST_NAME=apicurio-registry make run-operator-tests
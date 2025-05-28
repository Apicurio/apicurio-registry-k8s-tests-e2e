KIND_CLUSTER_NAME ?= "apicurio-cluster"
KIND_CLUSTER_CONFIG ?= kind-config.yaml

ifeq (1, $(shell command -v kind | wc -l))
KIND_CMD = kind
else
ifeq (1, $(shell command -v ./kind | wc -l))
KIND_CMD = ./kind
else
$(error "No kind binary found")
endif
endif

GINKGO_CMD = go run github.com/onsi/ginkgo/ginkgo

export E2E_SUITE_PROJECT_DIR=$(shell pwd)
OPERATOR_PROJECT_DIR ?= $(E2E_SUITE_PROJECT_DIR)/apicurio-registry-operator

# apicurio-registry variables
E2E_APICURIO_PROJECT_DIR?=$(E2E_SUITE_PROJECT_DIR)/apicurio-registry
# export E2E_APICURIO_TESTS_PROFILE=all

# operator bundle variables, operator repo should always have to be pulled, in order to access install.yaml file
BUNDLE_PATH ?= $(OPERATOR_PROJECT_DIR)/dist/install.yaml
export E2E_OPERATOR_BUNDLE_PATH = $(BUNDLE_PATH)

ifeq ($(OPERATOR_VERSION),)
$(error OPERATOR_VERSION is required)
endif

export E2E_OPERATOR_VERSION = $(OPERATOR_VERSION)

ifeq ($(OPERATOR_IMAGE_REPOSITORY),)
$(error OPERATOR_IMAGE_REPOSITORY is required)
endif

ifeq ($(OPERATOR_IMAGE),)
$(error OPERATOR_IMAGE is required)
endif

CI_BUILD_OPERATOR_IMAGE = localhost:5000/apicurio-registry-operator:latest-ci

ifeq ($(PACKAGE_VERSION),)
$(error PACKAGE_VERSION is required)
endif

export E2E_OLM_CSV = apicurio-registry-operator.v$(PACKAGE_VERSION)

ifeq ($(BUNDLE_IMAGE),)
$(error BUNDLE_IMAGE is required)
endif

CI_BUILD_BUNDLE_IMAGE = localhost:5000/apicurio-registry-operator-bundle:latest-ci

ifeq ($(CATALOG_IMAGE),)
$(error CATALOG_IMAGE is required)
endif

CI_BUILD_CATALOG_IMAGE = localhost:5000/apicurio-registry-operator-catalog:latest-ci

ifeq ($(CI_BUILD),true)
export E2E_OLM_CATALOG_SOURCE_IMAGE = $(CI_BUILD_CATALOG_IMAGE)
else
export E2E_OLM_CATALOG_SOURCE_IMAGE = $(CATALOG_IMAGE)
endif

# olm variables
OLM_PACKAGE_MANIFEST_NAME?=apicurio-registry-operator
export E2E_OLM_PACKAGE_MANIFEST_NAME=$(OLM_PACKAGE_MANIFEST_NAME)
# OLM Channel ommited, default channel will be used
# Temporarily uncommenting the following two lines, since the default channel does not seem to be detected?
OLM_CHANNEL?=2.x
export E2E_OLM_CHANNEL=$(OLM_CHANNEL)

OLM_CATALOG_SOURCE_NAMESPACE?=olm
export E2E_OLM_CATALOG_SOURCE_NAMESPACE=$(OLM_CATALOG_SOURCE_NAMESPACE)
OLM_CLUSTER_WIDE_OPERATORS_NAMESPACE?=operators
export E2E_OLM_CLUSTER_WIDE_OPERATORS_NAMESPACE=$(OLM_CLUSTER_WIDE_OPERATORS_NAMESPACE)

# upgrade test variables - not used
export E2E_OLM_UPGRADE_CHANNEL=alpha
export E2E_OLM_UPGRADE_OLD_CSV=apicurio-registry.v0.0.4-v1.3.2.final
export E2E_OLM_UPGRADE_NEW_CSV=apicurio-registry.v0.0.5-dev
export E2E_OLM_UPGRADE_OLD_CATALOG=operatorhubio-catalog
export E2E_OLM_UPGRADE_OLD_CATALOG_NAMESPACE=olm
#E2E_OLM_CATALOG_SOURCE_IMAGE is used as new catalog

# kafka storage variables
STRIMZI_BUNDLE_PATH ?= https://github.com/strimzi/strimzi-kafka-operator/releases/download/0.23.0/strimzi-cluster-operator-0.23.0.yaml
export E2E_STRIMZI_BUNDLE_PATH = $(STRIMZI_BUNDLE_PATH)

# CI
run-operator-ci: kind-start kind-setup-olm setup-operator-deps run-operator-tests

run-apicurio-base-ci: kind-start setup-apicurio-deps

run-apicurio-ci: run-apicurio-base-ci run-apicurio-tests

run-upgrade-ci: kind-start kind-setup-olm kind-catalog-source-img run-upgrade-tests

run-operator-simple: kind-start run-operator-tests/only-bundle


kind-catalog-source-img:
ifeq ($(CI_BUILD),true)
	# We need to build the bundle and catalog image which reference the CI images
	cd $(OPERATOR_PROJECT_DIR); make ADD_LATEST_TAG=false OPERATOR_IMAGE=$(CI_BUILD_OPERATOR_IMAGE) BUNDLE_IMAGE=$(CI_BUILD_BUNDLE_IMAGE) bundle bundle-build bundle-push
	cd $(OPERATOR_PROJECT_DIR); make ADD_LATEST_TAG=false OPERATOR_IMAGE=$(CI_BUILD_OPERATOR_IMAGE) BUNDLE_IMAGE=$(CI_BUILD_BUNDLE_IMAGE) CATALOG_IMAGE=$(CI_BUILD_CATALOG_IMAGE) catalog-build catalog-push
	docker push $(CI_BUILD_CATALOG_IMAGE)
endif


debug:
	echo "$(OPERATOR_PROJECT_DIR)"
	echo "$(OPERATOR_IMAGE_REPOSITORY)"
	echo "$(BUNDLE_PATH)"
	echo "$(E2E_OPERATOR_BUNDLE_PATH)"


kind-load-operator-images:
	echo "$(OPERATOR_PROJECT_DIR)"
	echo "$(OPERATOR_IMAGE_REPOSITORY)"
	echo "$(BUNDLE_PATH)"
	echo "$(E2E_OPERATOR_BUNDLE_PATH)"
	docker tag $(OPERATOR_IMAGE) $(CI_BUILD_OPERATOR_IMAGE)
	docker push $(CI_BUILD_OPERATOR_IMAGE)
	sed -i "s#$(OPERATOR_IMAGE_REPOSITORY)/apicurio-registry-operator.*#$(CI_BUILD_OPERATOR_IMAGE)#" $(E2E_OPERATOR_BUNDLE_PATH)


setup-operator-deps: $(if $(CI_BUILD), kind-load-operator-images) kind-catalog-source-img

APICURIO_IMAGES_TAG?=latest-snapshot

kind-load-apicurio-images:
	docker tag apicurio/apicurio-registry-mem:$(APICURIO_IMAGES_TAG) localhost:5000/apicurio-registry-mem:latest-ci
	docker push localhost:5000/apicurio-registry-mem:latest-ci
	sed -i "s#quay.io/apicurio/apicurio-registry-mem.*#localhost:5000/apicurio-registry-mem:latest-ci#" $(E2E_OPERATOR_BUNDLE_PATH)

	docker tag apicurio/apicurio-registry-kafkasql:$(APICURIO_IMAGES_TAG) localhost:5000/apicurio-registry-kafkasql:latest-ci
	docker push localhost:5000/apicurio-registry-kafkasql:latest-ci
	sed -i "s#quay.io/apicurio/apicurio-registry-kafkasql.*#localhost:5000/apicurio-registry-kafkasql:latest-ci#" $(E2E_OPERATOR_BUNDLE_PATH)

	docker tag apicurio/apicurio-registry-sql:$(APICURIO_IMAGES_TAG) localhost:5000/apicurio-registry-sql:latest-ci
	docker push localhost:5000/apicurio-registry-sql:latest-ci
	sed -i "s#quay.io/apicurio/apicurio-registry-sql.*#localhost:5000/apicurio-registry-sql:latest-ci#" $(E2E_OPERATOR_BUNDLE_PATH)

default-replace-apicurio-images:
	sed -i "s#apicurio/apicurio-registry-mem.*#apicurio/apicurio-registry-mem:$(APICURIO_IMAGES_TAG)#" $(E2E_OPERATOR_BUNDLE_PATH)
	sed -i "s#apicurio/apicurio-registry-kafkasql.*#apicurio/apicurio-registry-kafkasql:$(APICURIO_IMAGES_TAG)#" $(E2E_OPERATOR_BUNDLE_PATH)
	sed -i "s#apicurio/apicurio-registry-sql.*#apicurio/apicurio-registry-sql:$(APICURIO_IMAGES_TAG)#" $(E2E_OPERATOR_BUNDLE_PATH)

ifeq ($(CI_BUILD),true)
APICURIO_TARGETS = kind-load-apicurio-images
else
APICURIO_TARGETS = default-replace-apicurio-images
endif

setup-apicurio-deps: $(APICURIO_TARGETS)
	#setup kafka connect converters distro
	cp $(E2E_APICURIO_PROJECT_DIR)/distro/connect-converter/target/apicurio-kafka-connect-converter-*.tar.gz scripts/converters/converter-distro.tar.gz

kind-delete:
	${KIND_CMD} delete cluster --name ${KIND_CLUSTER_NAME}
	./scripts/stop-kind-image-registry.sh

kind-start:
ifeq (1, $(shell ${KIND_CMD} get clusters | grep ${KIND_CLUSTER_NAME} | wc -l))
	@echo "Cluster already exists" 
else
	@echo "Creating Cluster"
	./scripts/start-kind-image-registry.sh
	# create a cluster with the local registry enabled in containerd
	${KIND_CMD} create cluster --name ${KIND_CLUSTER_NAME} --image=kindest/node:v1.25.16 --config=./scripts/${KIND_CLUSTER_CONFIG}
	./scripts/setup-kind-image-registry.sh
	# setup ingress
	# using nginx ingress version v0.46.0
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.3.1/deploy/static/provider/kind/deploy.yaml
	kubectl patch deployment ingress-nginx-controller -n ingress-nginx --type=json -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--enable-ssl-passthrough"}]'
endif

kind-setup-olm:
	./scripts/setup-olm.sh ; if [ $$? -ne 0 ] ; then ./scripts/setup-olm.sh ; fi

# we run olm tests only for operator testsuite
run-operator-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--junit-report=xunit-report.xml \
		--cover --trace --race --progress -v ./testsuite/bundle ./testsuite/olm -- -only-test-operator -disable-clustered-tests

run-lp-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--junit-report=xunit-report.xml \
		--cover --trace --race --progress -v ./testsuite/olm -- -only-test-operator -disable-clustered-tests -enable-olm-advanced-tests

run-operator-tests/only-bundle:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v ./testsuite/bundle -- -only-test-operator -disable-clustered-tests

# for apicurio-registry tests we mostly focus on registry functionality so there is no need to run olm tests as well
run-apicurio-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v ./testsuite/bundle -- -disable-clustered-tests

run-apicurio-tests-with-clustered-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v ./testsuite/bundle

run-upgrade-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v ./testsuite/upgrade

run-security-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="security" ./testsuite/bundle -- -only-test-operator

run-keycloak-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="keycloak" ./testsuite/bundle -- -only-test-operator

run-migration-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="migration" ./testsuite/bundle

run-clustered-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="clustered" ./testsuite/bundle

run-converters-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="converters" ./testsuite/bundle

run-backupandrestore-test:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="backup" ./testsuite/bundle

run-sql-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="sql" ./testsuite/bundle -- -only-test-operator -disable-clustered-tests

run-kafkasql-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="kafkasql" ./testsuite/bundle

run-olm-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v ./testsuite/olm -- -only-test-operator

example-run-sql-and-kafkasql-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="sql|kafkasql" -dryRun

example-run-sql-with-olm-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="olm.*sql" -dryRun

example-run-sql-with-olm-and-upgrade-tests:
	$(GINKGO_CMD) -r --randomize-all --randomize-suites --fail-on-pending --keep-going \
		--cover --trace --race --progress -v --focus="olm.*sql|upgrade" -dryRun

clean-tests-logs:
	rm -rf tests-logs

# repo dependencies utilities
APICURIO_REGISTRY_REPO?=https://github.com/Apicurio/apicurio-registry.git
APICURIO_REGISTRY_BRANCH?=master

pull-apicurio-registry:
ifeq (,$(wildcard ./apicurio-registry))
	git clone -b $(APICURIO_REGISTRY_BRANCH) $(APICURIO_REGISTRY_REPO)
else
	cd apicurio-registry; git pull
endif

build-apicurio-registry:
	# important parts from here are the connect-converters and the tenant-manager-client
	# cd apicurio-registry; mvn package -Pmultitenancy -DskipTests --no-transfer-progress -Dmaven.javadoc.skip=true
	cd apicurio-registry; mvn install -am -Pprod -pl distro/connect-converter -DskipTests -Dmaven.javadoc.skip=true --no-transfer-progress
	cd apicurio-registry; mvn install -am -Pprod -Pmultitenancy -pl 'multitenancy/tenant-manager-client' -DskipTests -Dmaven.javadoc.skip=true --no-transfer-progress

OPERATOR_REPO?=https://github.com/Apicurio/apicurio-registry-operator.git
OPERATOR_BRANCH?=master

pull-operator-repo:
ifeq (,$(wildcard ./apicurio-registry-operator))
	git clone -b $(OPERATOR_BRANCH) $(OPERATOR_REPO)
else
	cd apicurio-registry-operator; git pull
endif
	cd apicurio-registry-operator; make dist
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

# apicurio-registry variables
E2E_APICURIO_PROJECT_DIR?=$(E2E_SUITE_PROJECT_DIR)/apicurio-registry
# export E2E_APICURIO_TESTS_PROFILE=all

# operator bundle variables, operator repo should always have to be pulled, in order to access install.yaml file
BUNDLE_URL?=$(E2E_SUITE_PROJECT_DIR)/apicurio-registry-operator/dist/default-install.yaml
export E2E_OPERATOR_BUNDLE_PATH=$(BUNDLE_URL)

OPERATOR_IMAGE?=quay.io/apicurio/apicurio-registry-operator:1.0.0-dev

# olm variables
export E2E_OLM_PACKAGE_MANIFEST_NAME=apicurio-registry-operator
export E2E_OLM_CHANNEL=apicurio-registry-2.x
OPERATOR_METADATA_IMAGE?=quay.io/apicurio/apicurio-registry-operator-bundle:1.0.0-dev
ifeq ($(CI_BUILD),true)
OPERATOR_METADATA_IMAGE=localhost:5000/apicurio-registry-operator-bundle:latest-ci
endif
CATALOG_SOURCE_IMAGE=localhost:5000/apicurio-registry-operator-index:1.0.0-dev
export E2E_OLM_CATALOG_SOURCE_IMAGE=$(CATALOG_SOURCE_IMAGE)
export E2E_OLM_CATALOG_SOURCE_NAMESPACE=olm
export E2E_OLM_CLUSTER_WIDE_OPERATORS_NAMESPACE=operators

# upgrade test variables
export E2E_OLM_UPGRADE_CHANNEL=alpha
export E2E_OLM_UPGRADE_OLD_CSV=apicurio-registry.v0.0.4-v1.3.2.final
export E2E_OLM_UPGRADE_NEW_CSV=apicurio-registry.v0.0.5-dev
export E2E_OLM_UPGRADE_OLD_CATALOG=operatorhubio-catalog
export E2E_OLM_UPGRADE_OLD_CATALOG_NAMESPACE=olm
#E2E_OLM_CATALOG_SOURCE_IMAGE is used as new catalog

# kafka storage variables
STRIMZI_BUNDLE_URL?=https://github.com/strimzi/strimzi-kafka-operator/releases/download/0.22.1/strimzi-cluster-operator-0.22.1.yaml
export E2E_STRIMZI_BUNDLE_PATH=$(STRIMZI_BUNDLE_URL)

# CI
run-operator-ci: create-summary-file kind-start kind-setup-olm pull-operator-repo setup-operator-deps run-operator-tests

run-apicurio-base-ci: create-summary-file kind-start pull-operator-repo setup-apicurio-deps

run-apicurio-ci: run-apicurio-base-ci run-apicurio-tests

run-upgrade-ci: create-summary-file kind-start kind-setup-olm pull-operator-repo kind-catalog-source-img run-upgrade-tests

CI_MESSAGE_HEADER?=Tests executed
SUMMARY_FILE=$(E2E_SUITE_PROJECT_DIR)/tests-logs/TESTS_SUMMARY.json
export E2E_SUMMARY_FILE=$(SUMMARY_FILE)
create-summary-file:
	rm $(SUMMARY_FILE) || true
	mkdir -p $(E2E_SUITE_PROJECT_DIR)/tests-logs
	cat $(E2E_SUITE_PROJECT_DIR)/scripts/CI_MESSAGE.json | sed -e 's/TEMPLATE/$(CI_MESSAGE_HEADER)/' > $(SUMMARY_FILE)

send-ci-message:
	./scripts/send-ci-message.sh $(SUMMARY_FILE)

# note there is no need to push CATALOG_SOURCE_IMAGE to docker hub
create-catalog-source-image:
ifeq ($(CI_BUILD),true)
	cd apicurio-registry-operator; make BUNDLE_IMAGE=$(OPERATOR_METADATA_IMAGE) OPERATOR_IMAGE=$(OPERATOR_IMAGE) bundle-build bundle-push
endif
	opm index add --bundles $(OPERATOR_METADATA_IMAGE) --tag $(CATALOG_SOURCE_IMAGE) --skip-tls --permissive -c docker

kind-catalog-source-img: create-catalog-source-image
	docker push $(CATALOG_SOURCE_IMAGE)

kind-load-operator-images:
	docker tag $(OPERATOR_IMAGE) localhost:5000/apicurio-registry-operator:latest-ci
	docker push localhost:5000/apicurio-registry-operator:latest-ci
	sed -i "s#quay.io/apicurio/apicurio-registry-operator.*#localhost:5000/apicurio-registry-operator:latest-ci#" $(E2E_OPERATOR_BUNDLE_PATH)

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
	${KIND_CMD} create cluster --name ${KIND_CLUSTER_NAME} --image=kindest/node:v1.19.0 --config=./scripts/${KIND_CLUSTER_CONFIG}
	./scripts/setup-kind-image-registry.sh
	# setup ingress
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/kind/deploy.yaml
	kubectl patch deployment ingress-nginx-controller -n ingress-nginx --type=json -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--enable-ssl-passthrough"}]'
endif

kind-setup-olm:
	./scripts/setup-olm.sh ; if [ $$? -ne 0 ] ; then ./scripts/setup-olm.sh ; fi

# we run olm tests only for operator testsuite
run-operator-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v ./testsuite/bundle ./testsuite/olm -- -only-test-operator -disable-clustered-tests

# for apicurio-registry tests we mostly focus on registry functionality so there is no need to run olm tests as well
run-apicurio-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v ./testsuite/bundle -- -disable-clustered-tests

run-apicurio-tests-with-clustered-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v ./testsuite/bundle

run-upgrade-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v ./testsuite/upgrade

run-security-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="security" ./testsuite/bundle -- -only-test-operator

run-keycloak-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="keycloak" ./testsuite/bundle -- -only-test-operator

run-migration-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="migration" ./testsuite/bundle

run-clustered-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="clustered" ./testsuite/bundle

run-converters-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="converters" ./testsuite/bundle

run-backupandrestore-test:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="backup" ./testsuite/bundle

run-sql-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="sql" ./testsuite/bundle -- -only-test-operator -disable-clustered-tests

run-kafkasql-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="kafkasql" ./testsuite/bundle

run-olm-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v ./testsuite/olm -- -only-test-operator

example-run-sql-and-kafkasql-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="sql|kafkasql" -dryRun

example-run-sql-with-olm-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="olm.*sql" -dryRun

example-run-sql-with-olm-and-upgrade-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
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
KIND_CLUSTER_NAME ?= "apicurio-cluster"

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
# export E2E_APICURIO_TESTS_PROFILE=cluster

# operator bundle variables, operator repo should always have to be pulled, in order to access install.yaml file
BUNDLE_URL?=$(E2E_SUITE_PROJECT_DIR)/apicurio-registry-operator/docs/resources/install.yaml
export E2E_OPERATOR_BUNDLE_PATH=$(BUNDLE_URL)

# olm variables
OPERATOR_METADATA_IMAGE?=docker.io/apicurio/apicurio-registry-operator-metadata:latest-dev
CATALOG_SOURCE_IMAGE=docker.io/apicurio/apicurio-registry-operator-catalog-source:latest-dev
export E2E_OLM_CATALOG_SOURCE_IMAGE=$(CATALOG_SOURCE_IMAGE)

# kafka streams variables
STRIMZI_BUNDLE_URL?=https://github.com/strimzi/strimzi-kafka-operator/releases/download/0.18.0/strimzi-cluster-operator-0.18.0.yaml
export E2E_STRIMZI_BUNDLE_PATH=$(STRIMZI_BUNDLE_URL)

# CI
# run-operator-ci: kind-start kind-catalog-source-img run-operator-tests
# FIXME ignoring olm for now
run-operator-ci: kind-start pull-operator-repo setup-operator-deps run-operator-tests

run-apicurio-ci: kind-start pull-operator-repo setup-apicurio-deps run-apicurio-tests

# note there is no need to push CATALOG_SOURCE_IMAGE to docker hub
create-catalog-source-image:
	docker build -t $(CATALOG_SOURCE_IMAGE) --build-arg MANIFESTS_IMAGE=$(OPERATOR_METADATA_IMAGE) ./olm-catalog-source

# FIXME ignoring olm for now
kind-catalog-source-img: create-catalog-source-image
	${KIND_CMD} load docker-image $(CATALOG_SOURCE_IMAGE) --name $(KIND_CLUSTER_NAME) -v 5

OPERATOR_IMAGE?=docker.io/apicurio/apicurio-registry-operator:latest-dev

kind-load-operator-images:
	docker tag $(OPERATOR_IMAGE) localhost:5000/apicurio-registry-operator:latest-ci
	docker push localhost:5000/apicurio-registry-operator:latest-ci
	sed -i "s#docker.io/apicurio/apicurio-registry-operator.*#localhost:5000/apicurio-registry-operator:latest-ci#" $(E2E_OPERATOR_BUNDLE_PATH)

setup-operator-deps: $(if $(CI_BUILD), kind-load-operator-images)

APICURIO_IMAGES_TAG?=latest-snapshot

kind-load-apicurio-images:
	docker tag apicurio/apicurio-registry-mem:$(APICURIO_IMAGES_TAG) localhost:5000/apicurio-registry-mem:latest-ci
	docker push localhost:5000/apicurio-registry-mem:latest-ci
	sed -i "s#docker.io/apicurio/apicurio-registry-mem.*\"#localhost:5000/apicurio-registry-mem:latest-ci\"#" $(E2E_OPERATOR_BUNDLE_PATH)

	docker tag apicurio/apicurio-registry-streams:$(APICURIO_IMAGES_TAG) localhost:5000/apicurio-registry-streams:latest-ci
	docker push localhost:5000/apicurio-registry-streams:latest-ci
	sed -i "s#docker.io/apicurio/apicurio-registry-streams.*\"#localhost:5000/apicurio-registry-streams:latest-ci\"#" $(E2E_OPERATOR_BUNDLE_PATH)

	docker tag apicurio/apicurio-registry-jpa:$(APICURIO_IMAGES_TAG) localhost:5000/apicurio-registry-jpa:latest-ci
	docker push localhost:5000/apicurio-registry-jpa:latest-ci
	sed -i "s#docker.io/apicurio/apicurio-registry-jpa.*\"#localhost:5000/apicurio-registry-jpa:latest-ci\"#" $(E2E_OPERATOR_BUNDLE_PATH)

	docker tag apicurio/apicurio-registry-infinispan:$(APICURIO_IMAGES_TAG) localhost:5000/apicurio-registry-infinispan:latest-ci
	docker push localhost:5000/apicurio-registry-infinispan:latest-ci
	sed -i "s#docker.io/apicurio/apicurio-registry-infinispan.*\"#localhost:5000/apicurio-registry-infinispan:latest-ci\"#" $(E2E_OPERATOR_BUNDLE_PATH)

default-replace-apicurio-images:
	sed -i "s#apicurio/apicurio-registry-mem.*\"#apicurio/apicurio-registry-mem:$(APICURIO_IMAGES_TAG)\"#" $(E2E_OPERATOR_BUNDLE_PATH)
	sed -i "s#apicurio/apicurio-registry-streams.*\"#apicurio/apicurio-registry-streams:$(APICURIO_IMAGES_TAG)\"#" $(E2E_OPERATOR_BUNDLE_PATH)
	sed -i "s#apicurio/apicurio-registry-jpa.*\"#apicurio/apicurio-registry-jpa:$(APICURIO_IMAGES_TAG)\"#" $(E2E_OPERATOR_BUNDLE_PATH)
	sed -i "s#apicurio/apicurio-registry-infinispan.*\"#apicurio/apicurio-registry-infinispan:$(APICURIO_IMAGES_TAG)\"#" $(E2E_OPERATOR_BUNDLE_PATH)

ifeq ($(CI_BUILD),true)
APICURIO_TARGETS = kind-load-apicurio-images
else
APICURIO_TARGETS = default-replace-apicurio-images
endif

setup-apicurio-deps: $(APICURIO_TARGETS)
	#setup kafka connect converters distro
	cp $(E2E_APICURIO_PROJECT_DIR)/distro/connect-converter/target/apicurio-kafka-connect-converter-*-converter.tar.gz scripts/converters/converter-distro.tar.gz

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
	${KIND_CMD} create cluster --name ${KIND_CLUSTER_NAME} --image=kindest/node:v1.17.5 --config=./scripts/kind-config.yaml
	./scripts/setup-kind-image-registry.sh
	# setup ingress
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/kind/deploy.yaml
	kubectl patch deployment ingress-nginx-controller -n ingress-nginx --type=json -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--enable-ssl-passthrough"}]'
	# FIXME ignoring olm for now
	# ./scripts/setup-olm.sh ; if [ $$? -ne 0 ] ; then ./scripts/setup-olm.sh ; fi
endif

run-operator-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v ./testsuite/bundle -- -only-test-operator
	# FIXME ignoring olm for now
	# ./testsuite/olm 

run-apicurio-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v ./testsuite/bundle

run-converters-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="converters" ./testsuite/bundle

run-backupandrestore-test:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="backup" ./testsuite/bundle

run-jpa-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="jpa" ./testsuite/bundle

example-run-jpa-and-streams-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="jpa|streams" -dryRun

example-run-jpa-with-olm-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="olm.*jpa" -dryRun

example-run-jpa-with-olm-and-upgrade-tests:
	$(GINKGO_CMD) -r --randomizeAllSpecs --randomizeSuites --failOnPending -keepGoing \
		--cover --trace --race --progress -v --focus="olm.*jpa|upgrade" -dryRun

clean-tests-logs:
	rm -rf tests-logs

# repo dependencies utilities
pull-apicurio-registry:
ifeq (,$(wildcard ./apicurio-registry))
	git clone https://github.com/Apicurio/apicurio-registry.git
else
	cd apicurio-registry; git pull
endif

build-apicurio-registry:
	cd apicurio-registry; mvn package -DskipTests -pl '!tests' --no-transfer-progress

pull-operator-repo:
ifeq (,$(wildcard ./apicurio-registry-operator))
	git clone https://github.com/Apicurio/apicurio-registry-operator.git
else
	cd apicurio-registry-operator; git pull
endif
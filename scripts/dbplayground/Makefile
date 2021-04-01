
# IMAGE_NAME?=quay.io/rh_integration/service-registry-dbplayground:pg10
IMAGE_NAME?=quay.io/rh_integration/service-registry-dbplayground:pg12

build:
	docker build -t $(IMAGE_NAME) .

push:
	docker push $(IMAGE_NAME)
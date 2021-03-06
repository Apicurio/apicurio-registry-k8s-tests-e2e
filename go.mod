module github.com/Apicurio/apicurio-registry-k8s-tests-e2e

go 1.14

require (
	github.com/Apicurio/apicurio-registry-operator v0.0.0-20210427095249-dccf34b4843a
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v0.0.0-20210317213936-dcbf045ae1b8
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/api v0.5.3
	github.com/operator-framework/operator-lifecycle-manager v0.17.0
	github.com/segmentio/kafka-go v0.3.10
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
	sigs.k8s.io/controller-runtime v0.8.0

)

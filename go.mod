module github.com/Apicurio/apicurio-registry-k8s-tests-e2e

go 1.16

require (
	github.com/Apicurio/apicurio-registry-operator v1.0.1-0.20210702070317-8fcad4efd108
	github.com/onsi/ginkgo v1.16.5-0.20210926212817-d0c597ffc7d0
	github.com/onsi/gomega v1.16.0
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

module github.com/Apicurio/apicurio-registry-k8s-tests-e2e

go 1.13

require (
	github.com/Apicurio/apicurio-registry-operator v0.0.0-20200716121633-a0066804b59c
	github.com/confluentinc/confluent-kafka-go v1.4.2
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200321030439-57b580e57e88
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible

	// sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/controller-runtime v0.5.2
)

// replace github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200326155132-2a6cd50aedd0 // release-4.5
// github.com/operator-framework/api v0.3.11 // indirect
replace (
	//controller-runtime v0.5.2 requires k8s 0.17.x
	k8s.io/api => k8s.io/api v0.17.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.9
	k8s.io/client-go => k8s.io/client-go v0.17.9
)

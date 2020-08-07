module github.com/famartinrh/apicurio-registry-k8s-tests-e2e

go 1.13

require (
	github.com/Apicurio/apicurio-registry-operator v0.0.0-20200716121633-a0066804b59c
	github.com/fatih/color v1.9.0 // indirect
	github.com/gobuffalo/flect v0.2.1 // indirect
	github.com/kisielk/errcheck v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200321030439-57b580e57e88
	// github.com/operator-framework/api v0.3.11 // indirect
	github.com/spf13/cobra v1.0.0 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/sys v0.0.0-20200806060901-a37d78b92225 // indirect
	golang.org/x/tools v0.0.0-20200806022845-90696ccdc692 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/imdario/mergo.v0 v0.3.9 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/cluster-bootstrap v0.18.6 // indirect
	k8s.io/utils v0.0.0-20200731180307-f00132d28269 // indirect

	// sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/controller-runtime v0.5.2

	sigs.k8s.io/testing_frameworks v0.1.2 // indirect
)

// replace github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200326155132-2a6cd50aedd0 // release-4.5
// github.com/operator-framework/api v0.3.11 // indirect
replace (
	//controller-runtime v0.5.2 requires k8s 0.17.x
	k8s.io/api => k8s.io/api v0.17.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.9
	k8s.io/client-go => k8s.io/client-go v0.17.9
)

> ⚠️ Warning
> 
> **This repository is in maintenance-only mode and will be archived in the near future.**

<br/><br/>

![latest registry results](https://github.com/Apicurio/apicurio-registry-k8s-tests-e2e/workflows/Apicurio%20Registry%20Tests/badge.svg?branch=master)
![Latest operator results](https://github.com/Apicurio/apicurio-registry-k8s-tests-e2e/workflows/Apicurio%20Registry%20Operator%20Tests/badge.svg?branch=master)

# Apicurio Registry Kubernetes E2E testsuite

This repository contains a golang testsuite to verify apicurio-registry-operator and apicurio-registry. 
This testsuite is used to verify apicurio-registry-operator functionality in Kubernetes/Openshift, while at the same time is able to
execute [apicurio-registry functional tests](https://github.com/Apicurio/apicurio-registry/tree/master/integration-tests/testsuite) (written in Java) to verify the functionality of apicurio-registry while deployed in Kubernetes/Openshift.

## How the testsuite works?

This testsuite can be executed with different parameters that will result in a combination of different testcases being executed. Our testcases are mainly differentiated by 
the type of deployment used to deploy apicurio-registry-operator.

We differentiate two types of apicurio-registry-operator deployment:
- OLM deployment, this part of the testsuite can be found [here](testsuite/olm)
- bundle deployment, this means deploying the operator by just appliying the operator yaml manifests directly to Kubernetes/Openshift. this part of the testsuite can be found [here](testsuite/bundle)

We are using Ginkgo as the framework to develop this testsuite, and actually the tests for each one of deployment types are invoked from separated Ginkgo testsuites. i.e: [OLM testsuite](testsuite/olm/singlenamespace/suite_test.go)

Our Ginkgo testsuites usually follow this procedure:
- deploy apicurio-registry-operator using any deployment type, usually this creates a test namespace `apicurio-registry-e2e`
- run multiple testcases that test the functionality of the operator, i.e: deploy apicurio-registry using sql storage
    - if functional testing is enabled, the Java Integration tests are invoked to verify that the deployed apicurio-registry instance works correctly
    - if no functional testing is enabled, a set of simple http requests are made to the apicurio-registry instance in order to verify it's properly deployed and alive.
- deprovision the deployed apicurio-registry instance (provisioning and deprovisioning is done for each testcase)
- collect logs from all the pods deployed in the test namespace, collect events and related kubernetes resources and logs ...
- remove the apicurio-registry-operator deployment from the Kubernetes/Openshift cluster and clean up any other resources

Regardless of the deployment type used we have some basic testcases that are common to this two deployment types, i.e: deploy apicurio-registry with each one of the available storages(i.e: kafkasql and sql)

## How to start using the testsuite?

The easiest way to get an idea of how to run the testsuite is by checking our [Github Actions Workflows](.github/workflows)

But here is an example of how to run the operator tests.

**Note**
You need a Kubernetes or Openshift cluster. This document will cover how to run the testsuite in Kubernetes.
You need to have the tool [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/) installed. There is an utility script to do that in `/scripts/install_kind.sh`

And then run:

`make run-operator-ci`

This will create a specific Kind cluster with OLM available, then it will create a catalog-source image from the [apicurio-registry-operator-metadata image](https://hub.docker.com/r/apicurio/apicurio-registry-operator-metadata/tags) in order to successfully test operator OLM installation, and finally it will execute the tests under `/testsuite` folder.


# Apicurio Registry Kubernetes E2E testsuite

This repository contains a golang testsuite to verify apicurio-registry-operator and apicurio-registry

## How to start using the testsuite?

First of all you need [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/) installed. There is an utility script to do that in `/scripts/install_kind.sh`

And then run:

`make run-operator-ci` or `make run-ci`

This will create a specific Kind cluster with OLM available, then it will create a catalog-source image from the [apicurio-registry-operator-metadata image](https://hub.docker.com/r/apicurio/apicurio-registry-operator-metadata/tags) in order to successfully test operator OLM installation, and finally it will execute the tests under `/testsuite` folder.

Currently there are tests for operator bundle installation and OLM installation (upgrade tests will come in the future as well as other procedures that need testing). For bundle and OLM installations, all different usecases of ApicurioRegistry deployments are executed (for now only JPA storage is being tested, there are plans to include the rest of the storage variants as well as specific configurations for some storage variants)

The differences between `run-operator-ci` and `run-ci` are on the execution of the Apicurio Registry "functional" tests to verify the registry APIs, Kafka SerDes libraries,... For that `run-ci` executes, using maven, the tests located in `tests` folder on [Apicurio Registry repository](https://github.com/Apicurio/apicurio-registry/tree/master/tests)



package utils

import (
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	. "github.com/onsi/gomega"
)

var log = logf.Log.WithName("kubernetes-utils")

func IsOCP(config *rest.Config) (bool, error) {
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}

	_, err = client.ServerResourcesForGroupVersion("route.openshift.io/v1")

	if err != nil && errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

//CreateTestNamespace creates one namespace with the given name
func CreateTestNamespace(clientset *kubernetes.Clientset, namespace string) {
	log.Info("Creating namespace", "name", namespace)
	_, err := clientset.CoreV1().Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
	Expect(err).ToNot(HaveOccurred())
}

//DeleteTestNamespace removes one namespace and waits until it's deleted
func DeleteTestNamespace(clientset *kubernetes.Clientset, namespace string) {
	ns, err := clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
	if ns != nil {
		log.Info("Removing namespace", "name", namespace)
		err = clientset.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		timeout := 60 * time.Second
		log.Info("Waiting for namespace to be removed", "timeout", timeout)
		err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
			od, err := clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			if od != nil {
				return false, nil
			}
			return true, nil
		})
		Expect(err).ToNot(HaveOccurred())
	}
}

func WaitForOperatorDeploymentReady(clientset *kubernetes.Clientset, namespace string) {
	timeout := 120 * time.Second
	log.Info("Waiting for operator to be deployed", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(namespace).Get(utils.OperatorDeploymentName, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		if od != nil {
			if od.Status.AvailableReplicas > int32(0) {
				return true, nil
			}
		}
		return false, nil
	})
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())
}

func WaitForOperatorDeploymentRemoved(clientset *kubernetes.Clientset, namespace string) {
	timeout := 60 * time.Second
	log.Info("Waiting for operator to be removed", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(namespace).Get(utils.OperatorDeploymentName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		if od != nil {
			return false, nil
		}
		return true, nil
	})
	Expect(err).ToNot(HaveOccurred())
}

func isOperatorDeployed(clientset *kubernetes.Clientset, namespace string) (bool, error) {
	od, err := clientset.AppsV1().Deployments(namespace).Get(utils.OperatorDeploymentName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if od != nil {
		return true, nil
	}
	return false, nil
}

func WaitForDeploymentReady(clientset *kubernetes.Clientset, timeout time.Duration, namespace string, deploymentName string, expectedReplicas int) {
	if expectedReplicas == 0 {
		expectedReplicas = 1
	}
	// timeout := 120 * time.Second
	log.Info("Waiting for deployment "+deploymentName+" to be ready ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		if od != nil {
			if od.Status.AvailableReplicas == int32(expectedReplicas) {
				return true, nil
			}
		}
		return false, nil
	})
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())
}

func WaitForObjectDeleted(name string, apiCall func() (interface{}, error)) {
	timeout := 30 * time.Second
	log.Info("Waiting for "+name+" to be removed ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := apiCall()
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	Expect(err).ToNot(HaveOccurred())
}

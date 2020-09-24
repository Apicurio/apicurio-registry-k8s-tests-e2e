package utils

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/gomega"
)

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
		err := wait.Poll(APIPollInterval, timeout, func() (bool, error) {
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

func WaitForOperatorDeploymentReady(clientset *kubernetes.Clientset) {
	timeout := 120 * time.Second
	log.Info("Waiting for operator to be deployed", "timeout", timeout)
	err := wait.Poll(APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(OperatorNamespace).Get(OperatorDeploymentName, metav1.GetOptions{})
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
	ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())
}

func WaitForOperatorDeploymentRemoved(clientset *kubernetes.Clientset) {
	timeout := 60 * time.Second
	log.Info("Waiting for operator to be removed", "timeout", timeout)
	err := wait.Poll(APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(OperatorNamespace).Get(OperatorDeploymentName, metav1.GetOptions{})
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

func isOperatorDeployed(clientset *kubernetes.Clientset) (bool, error) {
	od, err := clientset.AppsV1().Deployments(OperatorNamespace).Get(OperatorDeploymentName, metav1.GetOptions{})
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

func WaitForDeploymentReady(clientset *kubernetes.Clientset, timeout time.Duration, deploymentName string, expectedReplicas int) {
	if expectedReplicas == 0 {
		expectedReplicas = 1
	}
	// timeout := 120 * time.Second
	log.Info("Waiting for deployment "+deploymentName+" to be ready ", "timeout", timeout)
	err := wait.Poll(APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(OperatorNamespace).Get(deploymentName, metav1.GetOptions{})
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
	ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())
}

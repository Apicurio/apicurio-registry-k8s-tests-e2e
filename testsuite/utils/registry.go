package utils

import (
	"context"
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubetypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

func WaitForRegistryReady(K8sClient client.Client, clientset *kubernetes.Clientset, registryName string, storage string) {

	// var registryDeploymentName string = "apicurio-registry-" + storage

	timeout := 15 * time.Second
	log.Info("Waiting for registry CR", "timeout", timeout)
	err := wait.Poll(APIPollInterval, timeout, func() (bool, error) {
		existing := apicurio.ApicurioRegistry{}
		err := K8sClient.Get(context.TODO(),
			kubetypes.NamespacedName{Name: registryName, Namespace: OperatorNamespace},
			&existing)

		if err != nil {
			if errors.IsNotFound(err) {
				//continue waiting
				return false, nil
			}
			return false, err
		}
		//TODO operator is not updating status
		// if existing.Status.DeploymentName != "" {
		// 	registryDeploymentName = existing.Status.DeploymentName
		// 	return true, nil
		// }
		return true, nil
	})
	ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	timeout = 180 * time.Second
	log.Info("Waiting for registry deployment to be ready", "timeout", timeout)
	err = wait.Poll(APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"app": "apicurio-registry-" + storage})

		deployments, err := clientset.AppsV1().Deployments(OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
		// registryDeployment, err := clientset.AppsV1().Deployments(OperatorNamespace).Get(context.TODO(), registryDeploymentName, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		if len(deployments.Items) != 0 {
			registryDeployment := deployments.Items[0]
			if registryDeployment.Status.AvailableReplicas > int32(0) {
				return true, nil
			}
		}
		return false, nil
	})
	ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

}

func DeleteRegistryAndWait(K8sClient client.Client, clientset *kubernetes.Clientset, registryName string, storage string) {

	obj := &apicurio.ApicurioRegistry{}
	err := K8sClient.Get(context.TODO(), kubetypes.NamespacedName{Name: registryName, Namespace: OperatorNamespace}, obj)
	if err != nil && !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
	log.Info("Removing registry CR")
	err = K8sClient.Delete(context.TODO(), obj)
	Expect(err).ToNot(HaveOccurred())

	timeout := 15 * time.Second
	log.Info("Waiting for registry CR to be removed", "timeout", timeout)
	err = wait.Poll(APIPollInterval, timeout, func() (bool, error) {
		existing := apicurio.ApicurioRegistry{}
		err := K8sClient.Get(context.TODO(),
			kubetypes.NamespacedName{Name: registryName, Namespace: OperatorNamespace},
			&existing)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	Expect(err).ToNot(HaveOccurred())

	ExecuteCmdOrDie(true, "kubectl", "get", "apicurioregistry", "-n", OperatorNamespace)

	//TODO operator bug, deployment is not removed
	// timeout = 30 * time.Second
	// log.Info("Waiting for registry deployment to be removed", "timeout", timeout)
	// err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
	// 	labelsSet := labels.Set(map[string]string{"app": "apicurio-registry-" + ctx.Storage})

	// 	deployments, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	// 	// registryDeployment, err := clientset.AppsV1().Deployments(OperatorNamespace).Get(context.TODO(), registryDeploymentName, metav1.GetOptions{})
	// 	if err != nil {
	// 		if errors.IsNotFound(err) {
	// 			return true, nil
	// 		}
	// 		return false, err
	// 	}
	// 	if len(deployments.Items) == 0 {
	// 		return true, nil
	// 	}
	// 	return false, nil
	// })
	// utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	// Expect(err).ToNot(HaveOccurred())

}

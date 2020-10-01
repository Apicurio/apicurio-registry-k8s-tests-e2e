package apicurio

import (
	"context"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubetypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var log = logf.Log.WithName("apicurio")

//CreateRegistryAndWait common logic to create one ApicurioRegistry and wait for the deployment to be ready
func CreateRegistryAndWait(suiteCtx *suite.SuiteContext, ctx *types.TestContext, registry *apicurio.ApicurioRegistry) {

	if !suiteCtx.IsOpenshift {
		// this is just a workaround to work with Kind nginx ingress
		registry.Spec.Deployment.Host = "localhost"
		ctx.RegistryHost = "localhost"
		ctx.RegistryPort = "80"
	}

	err := suiteCtx.K8sClient.Create(context.TODO(), registry)
	Expect(err).ToNot(HaveOccurred())

	waitForRegistryReady(suiteCtx, registry.Name)

	if suiteCtx.IsOpenshift {
		labelsSet := labels.Set(map[string]string{"app": registry.Name})
		routes, err := suiteCtx.OcpRouteClient.Routes(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(routes.Items)).To(BeIdenticalTo(1))
		Expect(len(routes.Items[0].Status.Ingress)).ToNot(BeIdenticalTo(0))
		route := routes.Items[0]

		ctx.RegistryHost = route.Status.Ingress[0].Host
		ctx.RegistryPort = "80"
	}

}

func waitForRegistryReady(suiteCtx *suite.SuiteContext, registryName string) {

	// var registryDeploymentName string = registryName

	timeout := 15 * time.Second
	log.Info("Waiting for registry CR", "timeout", timeout)
	apicurioRegistry := apicurio.ApicurioRegistry{}
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		err := suiteCtx.K8sClient.Get(context.TODO(),
			kubetypes.NamespacedName{Name: registryName, Namespace: utils.OperatorNamespace},
			&apicurioRegistry)

		if err != nil {
			if errors.IsNotFound(err) {
				//continue waiting
				return false, nil
			}
			return false, err
		}
		//TODO operator is not updating status
		// if apicurioRegistry.Status.DeploymentName != "" {
		// 	registryDeploymentName = apicurioRegistry.Status.DeploymentName
		// 	return true, nil
		// }
		return true, nil
	})
	kubernetescli.Execute("get", "apicurioregistry", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	var registryReplicas int32 = 1
	if apicurioRegistry.Status.ReplicaCount != 0 {
		registryReplicas = apicurioRegistry.Status.ReplicaCount
	}

	timeout = 180 * time.Second
	log.Info("Waiting for registry deployment to be ready", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"app": registryName})

		if suiteCtx.IsOpenshift {
			deployments, err := suiteCtx.OcpAppsClient.DeploymentConfigs(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			}
			if len(deployments.Items) != 0 {
				registryDeployment := deployments.Items[0]
				if registryDeployment.Status.AvailableReplicas == registryReplicas {
					return true, nil
				}
			}
		} else {
			deployments, err := suiteCtx.Clientset.AppsV1().Deployments(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			}
			if len(deployments.Items) != 0 {
				registryDeployment := deployments.Items[0]
				if registryDeployment.Status.AvailableReplicas == registryReplicas {
					return true, nil
				}
			}
		}
		return false, nil
	})
	kubernetescli.GetPods(utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())
	log.Info("Registry deployment is ready")
}

//DeleteRegistryAndWait removes one ApicurioRegistry deployment and ensures it's deleted waiting
func DeleteRegistryAndWait(suiteCtx *suite.SuiteContext, registryName string) {

	obj := &apicurio.ApicurioRegistry{}
	err := suiteCtx.K8sClient.Get(context.TODO(), kubetypes.NamespacedName{Name: registryName, Namespace: utils.OperatorNamespace}, obj)
	if err != nil && !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
	log.Info("Removing registry CR")
	err = suiteCtx.K8sClient.Delete(context.TODO(), obj)
	Expect(err).ToNot(HaveOccurred())

	timeout := 15 * time.Second
	log.Info("Waiting for registry CR to be removed", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		existing := apicurio.ApicurioRegistry{}
		err := suiteCtx.K8sClient.Get(context.TODO(),
			kubetypes.NamespacedName{Name: registryName, Namespace: utils.OperatorNamespace},
			&existing)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	kubernetescli.Execute("get", "apicurioregistry", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	err = waitRegistryDeploymentDeleted(suiteCtx, registryName)
	//operator bug should be fixed
	// if err != nil {
	// 	log.Info("Verify operator, possible bug, registry deployment is not removed after deleteing ApirucioRegistry CR, manually removing it")
	// 	labelsSet := labels.Set(map[string]string{"app": registryName})
	// 	deployments, err := clientset.AppsV1().Deployments(OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	// 	Expect(err).ToNot(HaveOccurred())
	// 	Expect(len(deployments.Items)).To(Equal(1))
	// 	deployment := deployments.Items[0]
	// 	err = clientset.AppsV1().Deployments(OperatorNamespace).Delete(deployment.Name, &metav1.DeleteOptions{})
	// 	Expect(err).ToNot(HaveOccurred())

	// 	err = waitRegistryDeploymentDeleted(clientset, registryName)
	// 	Expect(err).ToNot(HaveOccurred())
	// }

}

func waitRegistryDeploymentDeleted(suiteCtx *suite.SuiteContext, registryName string) error {
	timeout := 30 * time.Second
	log.Info("Waiting for registry deployment to be removed", "timeout", timeout)
	return wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"app": registryName})

		if suiteCtx.IsOpenshift {
			deployments, err := suiteCtx.OcpAppsClient.DeploymentConfigs(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
			if err != nil {
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			if len(deployments.Items) == 0 {
				return true, nil
			}
		} else {
			deployments, err := suiteCtx.Clientset.AppsV1().Deployments(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
			if err != nil {
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			if len(deployments.Items) == 0 {
				return true, nil
			}
		}
		return false, nil
	})
}

//ExistsRegistry verifies if the ApicurioRegistry CR named registryName exists
func ExistsRegistry(suiteCtx *suite.SuiteContext, registryName string) bool {
	obj := &apicurio.ApicurioRegistry{}
	err := suiteCtx.K8sClient.Get(context.TODO(), kubetypes.NamespacedName{Name: registryName, Namespace: utils.OperatorNamespace}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		Expect(err).ToNot(HaveOccurred())
	}
	return obj.Name == registryName
}

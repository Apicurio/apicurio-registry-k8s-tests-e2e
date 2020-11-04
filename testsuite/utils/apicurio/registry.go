package apicurio

import (
	"context"
	"strconv"
	"strings"
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
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var log = logf.Log.WithName("apicurio")

//CreateRegistryAndWait common logic to create one ApicurioRegistry and wait for the deployment to be ready
func CreateRegistryAndWait(suiteCtx *types.SuiteContext, ctx *types.TestContext, registry *apicurio.ApicurioRegistry) {

	ctx.RegistryName = registry.Name

	if !suiteCtx.IsOpenshift {
		registry.Spec.Deployment.Host = registry.Name + ".127.0.0.1.nip.io"
		ctx.RegistryHost = registry.Name + ".127.0.0.1.nip.io"
		ctx.RegistryPort = "80"
	}

	//TODO review
	//should be the caller who decides the namespace to create the registry or should it be based on the test context?
	if registry.Namespace == "" {
		registry.Namespace = ctx.RegistryNamespace
	}

	err := suiteCtx.K8sClient.Create(context.TODO(), registry)
	Expect(err).ToNot(HaveOccurred())

	var registryReplicas int32 = 1
	if registry.Spec.Deployment.Replicas > 0 {
		registryReplicas = int32(registry.Spec.Deployment.Replicas)
	}

	WaitForRegistryReady(suiteCtx, registry.Namespace, registry.Name, registryReplicas)

	labelsSet := labels.Set(map[string]string{"app": registry.Name})

	if suiteCtx.IsOpenshift {
		kubernetescli.Execute("get", "route", "-n", ctx.RegistryNamespace)

		//TODO make this timeout configurable
		timeout := 90 * time.Second
		log.Info("Waiting for registry route to be ready", "timeout", timeout)
		err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
			routes, err := suiteCtx.OcpRouteClient.Routes(ctx.RegistryNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			}
			if len(routes.Items) != 0 && len(routes.Items[0].Status.Ingress) != 0 {
				//TODO fix this, workaround because operator first fills the host with a non existent url and later on it updates the host with a valid one
				//I don't know how this happens, if it's the operator updating twice, or if it's ocp changing the route host...
				return strings.HasSuffix(routes.Items[0].Status.Ingress[0].Host, ".com"), nil
			}
			return false, nil
		})
		kubernetescli.Execute("get", "route", "-n", ctx.RegistryNamespace)
		Expect(err).ToNot(HaveOccurred())
		routes, err := suiteCtx.OcpRouteClient.Routes(ctx.RegistryNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(routes.Items)).To(BeIdenticalTo(1))
		Expect(len(routes.Items[0].Status.Ingress)).ToNot(BeIdenticalTo(0))
		route := routes.Items[0]

		ctx.RegistryHost = route.Status.Ingress[0].Host
		ctx.RegistryPort = "80"
	}

	//TODO fix this, operator usability problem, service name should be consistent
	svcs, err := suiteCtx.Clientset.CoreV1().Services(ctx.RegistryNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	Expect(err).ToNot(HaveOccurred())
	Expect(len(svcs.Items)).To(BeIdenticalTo(1))
	reg := svcs.Items[0]

	ctx.RegistryInternalHost = reg.Name + "." + reg.Namespace
	ctx.RegistryInternalPort = strconv.Itoa(int(reg.Spec.Ports[0].Port))

}

func WaitForRegistryReady(suiteCtx *types.SuiteContext, namespace string, registryName string, registryReplicas int32) {

	// var registryDeploymentName string = registryName

	timeout := 15 * time.Second
	log.Info("Waiting for registry CR", "timeout", timeout)
	apicurioRegistry := apicurio.ApicurioRegistry{}
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		err := suiteCtx.K8sClient.Get(context.TODO(),
			kubetypes.NamespacedName{Name: registryName, Namespace: namespace},
			&apicurioRegistry)

		if err != nil {
			if errors.IsNotFound(err) {
				//continue waiting
				return false, nil
			}
			return false, err
		}
		// TODO operator is not updating status
		// if apicurioRegistry.Status.DeploymentName != "" {
		// 	// registryDeploymentName = apicurioRegistry.Status.DeploymentName
		// 	return true, nil
		// }
		return true, nil
	})
	kubernetescli.Execute("get", "apicurioregistry", "-n", namespace)
	Expect(err).ToNot(HaveOccurred())

	timeout = 180 * time.Second
	if registryReplicas > 1 {
		timeout = 300 * time.Second
	}
	log.Info("Waiting for registry deployment to be ready", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"app": registryName})

		if suiteCtx.IsOpenshift {
			deployments, err := suiteCtx.OcpAppsClient.DeploymentConfigs(namespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
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
			deployments, err := suiteCtx.Clientset.AppsV1().Deployments(namespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
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
	kubernetescli.GetPods(namespace)
	kubernetescli.Execute("get", "apicurioregistry", "-o", "yaml", "-n", namespace)
	Expect(err).ToNot(HaveOccurred())
	log.Info("Registry deployment is ready")
}

//DeleteRegistryAndWait removes one ApicurioRegistry deployment and ensures it's deleted waiting
func DeleteRegistryAndWait(suiteCtx *types.SuiteContext, namespace string, registryName string) {

	obj := &apicurio.ApicurioRegistry{}
	err := suiteCtx.K8sClient.Get(context.TODO(), kubetypes.NamespacedName{Name: registryName, Namespace: namespace}, obj)
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
			kubetypes.NamespacedName{Name: registryName, Namespace: namespace},
			&existing)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	kubernetescli.Execute("get", "apicurioregistry", "-n", namespace)
	Expect(err).ToNot(HaveOccurred())

	err = waitRegistryDeploymentDeleted(suiteCtx, namespace, registryName)

}

func waitRegistryDeploymentDeleted(suiteCtx *types.SuiteContext, namespace string, registryName string) error {
	timeout := 30 * time.Second
	log.Info("Waiting for registry deployment to be removed", "timeout", timeout)
	return wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"app": registryName})

		if suiteCtx.IsOpenshift {
			deployments, err := suiteCtx.OcpAppsClient.DeploymentConfigs(namespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
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
			deployments, err := suiteCtx.Clientset.AppsV1().Deployments(namespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
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
func ExistsRegistry(suiteCtx *types.SuiteContext, namespace string, registryName string) bool {
	obj := &apicurio.ApicurioRegistry{}
	err := suiteCtx.K8sClient.Get(context.TODO(), kubetypes.NamespacedName{Name: registryName, Namespace: namespace}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		Expect(err).ToNot(HaveOccurred())
	}
	return obj.Name == registryName
}

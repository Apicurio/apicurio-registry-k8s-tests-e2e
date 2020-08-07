package jpa

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/famartinrh/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	suite "github.com/famartinrh/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	types "github.com/famartinrh/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	"k8s.io/apimachinery/pkg/api/errors"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubetypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var log = logf.Log.WithName("postgresql")

const postgresqlName string = "postgresql"

var registryName string

//DeployJpaRegistry deploys a posgresql database and deploys an ApicurioRegistry CR using that database
func DeployJpaRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {
	log.Info("Deploying postgresql database")

	utils.ExecuteCmdOrDie(true, "kubectl", "apply", "-f", utils.SuiteProjectDirValue+"/kubefiles/postgres-deployment.yaml", "-n", utils.OperatorNamespace)

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	timeout := 120 * time.Second
	log.Info("Waiting for postgresql database to be ready ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get(postgresqlName, metav1.GetOptions{})
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
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	svc, err := clientset.CoreV1().Services(utils.OperatorNamespace).Get(postgresqlName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	dataSourceURL := "jdbc:postgresql://" + svc.Spec.ClusterIP + ":5432/"

	log.Info("Deploying apicurio registry")

	registryName = "apicurio-registry-" + ctx.Storage
	registry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: utils.OperatorNamespace,
			Name:      registryName,
		},
		Spec: apicurio.ApicurioRegistrySpec{
			Deployment: apicurio.ApicurioRegistrySpecDeployment{
				//TODO detect if cluster is kind and do this workaround only in that case
				Host: "localhost",
			},
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				Persistence: utils.StorageJpa,
				DataSource: apicurio.ApicurioRegistrySpecConfigurationDataSource{
					Url:      dataSourceURL,
					UserName: "apicurio-registry",
					Password: "password",
				},
			},
		},
	}

	err = suiteCtx.K8sClient.Create(context.TODO(), &registry)
	Expect(err).ToNot(HaveOccurred())

	// var registryDeploymentName string = "apicurio-registry-" + StorageJpa

	timeout = 15 * time.Second
	log.Info("Waiting for registry CR", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		existing := apicurio.ApicurioRegistry{}
		err := suiteCtx.K8sClient.Get(context.TODO(),
			kubetypes.NamespacedName{Name: registryName, Namespace: utils.OperatorNamespace},
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
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	timeout = 180 * time.Second
	log.Info("Waiting for registry deployment to be ready", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"app": "apicurio-registry-" + utils.StorageJpa})

		deployments, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
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
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	ctx.RegistryHost = "localhost"
	ctx.RegistryPort = "80"
}

//RemoveJpaRegistry uninstalls registry CR and postgresql database
func RemoveJpaRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	obj := &apicurio.ApicurioRegistry{}
	err := suiteCtx.K8sClient.Get(context.TODO(), kubetypes.NamespacedName{Name: registryName, Namespace: utils.OperatorNamespace}, obj)
	if err != nil && !kubeerrors.IsNotFound(err) {
		log.Info("Removing registry CR")
		err = suiteCtx.K8sClient.Delete(context.TODO(), obj)
		Expect(err).ToNot(HaveOccurred())
	}
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

	//TODO operator bug, deployment is not removed
	// timeout = 30 * time.Second
	// log.Info("Waiting for registry deployment to be removed", "timeout", timeout)
	// err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
	// 	labelsSet := labels.Set(map[string]string{"app": "apicurio-registry-" + utils.StorageJpa})

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

	log.Info("Removing postgresql database")

	utils.ExecuteCmdOrDie(true, "kubectl", "delete", "-f", utils.SuiteProjectDirValue+"/kubefiles/postgres-deployment.yaml", "-n", utils.OperatorNamespace)

	timeout = 30 * time.Second
	log.Info("Waiting for postgresql database to be removed ", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get(postgresqlName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

}

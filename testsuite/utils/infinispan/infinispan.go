package infinispan

import (
	"context"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var log = logf.Log.WithName("infinispan")

var registryName string

//DeployInfinispanRegistry deploys an ApicurioRegistry CR using infinispan as storage
func DeployInfinispanRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

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
				LogLevel:    "DEBUG",
				Persistence: utils.StorageInfinispan,
				Infinispan: apicurio.ApicurioRegistrySpecConfigurationInfinispan{
					ClusterName: "registry-application",
				},
			},
		},
	}

	err := suiteCtx.K8sClient.Create(context.TODO(), &registry)
	Expect(err).ToNot(HaveOccurred())

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	utils.WaitForRegistryReady(suiteCtx.K8sClient, clientset, registryName)

	ctx.RegistryHost = "localhost"
	ctx.RegistryPort = "80"
}

//RemoveInfinispanRegistry uninstalls registry CR
func RemoveInfinispanRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	utils.DeleteRegistryAndWait(suiteCtx.K8sClient, clientset, registryName)

}

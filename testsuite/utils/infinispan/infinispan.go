package infinispan

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	apicurioutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var log = logf.Log.WithName("infinispan")

var registryName string

//DeployInfinispanRegistry deploys an ApicurioRegistry CR using infinispan as storage
func DeployInfinispanRegistry(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	log.Info("Deploying apicurio registry")

	replicas := 1
	if ctx.Replicas > 0 {
		replicas = ctx.Replicas
	}

	registryName = "apicurio-registry-" + ctx.Storage
	registry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Name: registryName,
		},
		Spec: apicurio.ApicurioRegistrySpec{
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				LogLevel:    "DEBUG",
				Persistence: utils.StorageInfinispan,
				Infinispan: apicurio.ApicurioRegistrySpecConfigurationInfinispan{
					ClusterName: "registry-application",
				},
			},
			Deployment: apicurio.ApicurioRegistrySpecDeployment{
				Replicas: int32(replicas),
			},
		},
	}

	apicurioutils.CreateRegistryAndWait(suiteCtx, ctx, &registry)

}

//RemoveInfinispanRegistry uninstalls registry CR
func RemoveInfinispanRegistry(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	apicurioutils.DeleteRegistryAndWait(suiteCtx, ctx.RegistryNamespace, registryName)

}

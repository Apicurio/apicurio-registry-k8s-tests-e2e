package infinispan

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	apicurioutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio"
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
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				LogLevel:    "DEBUG",
				Persistence: utils.StorageInfinispan,
				Infinispan: apicurio.ApicurioRegistrySpecConfigurationInfinispan{
					ClusterName: "registry-application",
				},
			},
		},
	}

	apicurioutils.CreateRegistryAndWait(suiteCtx, ctx, &registry)

}

//RemoveInfinispanRegistry uninstalls registry CR
func RemoveInfinispanRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	apicurioutils.DeleteRegistryAndWait(suiteCtx, registryName)

}

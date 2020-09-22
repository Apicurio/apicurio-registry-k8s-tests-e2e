package testcase

import (
	"errors"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/functional"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/infinispan"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/jpa"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/streams"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("testcase")

//ExecuteTestCase common logic to test operator deploying an instance of ApicurioRegistry with one of it's storage variants
func ExecuteTestCase(suiteCtx *suite.SuiteContext, testContext *types.TestContext) {
	ExecuteTestOnStorage(suiteCtx, testContext, func() {
		if !suiteCtx.OnlyTestOperator {
			functional.ExecuteRegistryFunctionalTests(testContext)
		} else {
			functional.BasicRegistryAPITest(testContext)
		}
	})
}

//ExecuteTestOnStorage extensible logic to test apicurio registry functionality deployed with one of it's storage variants
func ExecuteTestOnStorage(suiteCtx *suite.SuiteContext, testContext *types.TestContext, testFunction func()) {
	if testContext.ID == "" {
		testContext.ID = testContext.Storage
	}

	defer cleanRegistryDeployment(suiteCtx, testContext)

	deployRegistryStorage(suiteCtx, testContext)
	log.Info("-----------------------------------------------------------")
	testFunction()
}

func deployRegistryStorage(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {
	if ctx.Storage == utils.StorageJpa {
		jpa.DeployJpaRegistry(suiteCtx, ctx)
	} else if ctx.Storage == utils.StorageStreams {
		streams.DeployStreamsRegistry(suiteCtx, ctx)
	} else if ctx.Storage == utils.StorageInfinispan {
		infinispan.DeployInfinispanRegistry(suiteCtx, ctx)
	} else {
		Expect(errors.New("Storage not implemented")).ToNot(HaveOccurred())
	}
}

//clean namespace, only thing that can be left is registry operator
func cleanRegistryDeployment(suiteCtx *suite.SuiteContext, ctx *types.TestContext) error {

	log.Info("-----------------------------------------------------------")

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	testDescription := CurrentGinkgoTestDescription()
	utils.SaveTestPodsLogs(clientset, suiteCtx.SuiteID, testDescription.TestText)

	if ctx.Storage == utils.StorageJpa {
		jpa.RemoveJpaRegistry(suiteCtx, ctx)
	} else if ctx.Storage == utils.StorageStreams {
		streams.RemoveStreamsRegistry(suiteCtx, ctx)
	} else if ctx.Storage == utils.StorageInfinispan {
		infinispan.RemoveInfinispanRegistry(suiteCtx, ctx)
	} else {
		return errors.New("Storage not implemented")
	}

	return nil
}

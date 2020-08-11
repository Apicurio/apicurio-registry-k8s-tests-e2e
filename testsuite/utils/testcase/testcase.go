package testcase

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/functional"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/jpa"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

//ExecuteTestCase common logic to test operator deploying an instance of ApicurioRegistry with one of it's storage variants
func ExecuteTestCase(suiteCtx *suite.SuiteContext, testContext *types.TestContext) {
	if testContext.ID == "" {
		testContext.ID = testContext.Storage
	}

	defer cleanRegistryDeployment(suiteCtx, testContext)
	runRegistryStorageTest(suiteCtx, testContext)
}

func runRegistryStorageTest(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	if ctx.Storage == utils.StorageJpa {
		jpa.DeployJpaRegistry(suiteCtx, ctx)
	} else {
		Expect(errors.New("Storage not implemented")).ToNot(HaveOccurred())
	}

	if !suiteCtx.OnlyTestOperator {
		functional.ExecuteRegistryFunctionalTests(ctx)
	}

}

//clean namespace, only thing that can be left is registry operator
func cleanRegistryDeployment(suiteCtx *suite.SuiteContext, ctx *types.TestContext) error {

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	testDescription := CurrentGinkgoTestDescription()
	utils.SaveTestPodsLogs(clientset, suiteCtx.SuiteID, testDescription.TestText)

	if ctx.Storage == utils.StorageJpa {
		jpa.RemoveJpaRegistry(suiteCtx, ctx)
	} else {
		return errors.New("Storage not implemented")
	}

	return nil
}

package deploy

import (
	"errors"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kafkasql"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/logs"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/sql"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func DeployRegistryStorage(suiteCtx *types.SuiteContext, ctx *types.TestContext) {
	if ctx.Storage == utils.StorageSql {
		sql.DeploySqlRegistry(suiteCtx, ctx)
	} else if ctx.Storage == utils.StorageKafkaSql {
		kafkasql.DeployKafkaSqlRegistry(suiteCtx, ctx)
	} else {
		Expect(errors.New("Storage not implemented")).ToNot(HaveOccurred())
	}
}

func RemoveRegistryDeployment(suiteCtx *types.SuiteContext, ctx *types.TestContext) error {

	beforeRemoveRegistryDeployment(suiteCtx, ctx)

	if ctx.Storage == utils.StorageSql {
		sql.RemoveJpaRegistry(suiteCtx, ctx)
	} else if ctx.Storage == utils.StorageKafkaSql {
		kafkasql.RemoveKafkaSqlRegistry(suiteCtx, ctx)
	} else {
		return errors.New("Storage not implemented")
	}

	return nil
}

func beforeRemoveRegistryDeployment(suiteCtx *types.SuiteContext, ctx *types.TestContext) {
	testDescription := CurrentGinkgoTestDescription()

	testName := ""
	for _, comp := range testDescription.ComponentTexts {
		testName += (comp + "-")
	}
	testName = testName[0 : len(testName)-1]

	if ctx.ID != "" {
		testName += ("-" + ctx.ID)
	}

	logs.SaveTestPodsLogs(suiteCtx.Clientset, suiteCtx.SuiteID, ctx.RegistryNamespace, testName)
	ctx.ExecuteCleanups()
}

package migration

import (
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio/deploy"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/functional"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

//DataMigrationTestcase runs migration profile in Functional testsuite, this profile tests the import/export functionalities of the Apicurio Registry API
func DataMigrationTestcase(suiteCtx *types.SuiteContext, testContext *types.TestContext) {

	sourceCtx := &types.TestContext{
		ID:                "source-" + testContext.ID,
		Storage:           testContext.Storage,
		RegistryName:      "source-registry",
		RegistryNamespace: testContext.RegistryNamespace,
		Size:              types.SmallSize,
	}

	testContext.RegisterCleanup(func() {
		deploy.RemoveRegistryDeployment(suiteCtx, sourceCtx)
	})
	deploy.DeployRegistryStorage(suiteCtx, sourceCtx)

	destCtx := &types.TestContext{
		ID:                "dest-" + testContext.ID,
		Storage:           testContext.Storage,
		RegistryName:      "dest-registry",
		RegistryNamespace: testContext.RegistryNamespace,
		Size:              types.SmallSize,
	}

	testContext.RegisterCleanup(func() {
		//so strimzi operator is removed by last cleanup
		destCtx.SkipInfraRemoval = true
		deploy.RemoveRegistryDeployment(suiteCtx, destCtx)
	})
	deploy.DeployRegistryStorage(suiteCtx, destCtx)

	migrationTestsCtx := &types.TestContext{
		ID:                     testContext.ID,
		Storage:                testContext.Storage,
		FunctionalTestsProfile: "migration",
		FunctionalTestsExtraEnv: []string{
			"SOURCE_REGISTRY_HOST=" + sourceCtx.RegistryHost,
			"SOURCE_REGISTRY_PORT=" + sourceCtx.RegistryPort,
			"DEST_REGISTRY_HOST=" + destCtx.RegistryHost,
			"DEST_REGISTRY_PORT=" + destCtx.RegistryPort,
		},
	}

	functional.ExecuteRegistryFunctionalTests(suiteCtx, migrationTestsCtx)
}

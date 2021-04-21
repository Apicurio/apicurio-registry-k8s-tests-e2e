package security

import (
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio/deploy"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/functional"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/keycloak"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/logs"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("security")

func Testcase(suiteCtx *types.SuiteContext, namespace string) {

	ctx := &types.TestContext{RegistryNamespace: namespace}
	keycloakURL := keycloak.DeployKeycloak(suiteCtx, ctx)
	defer keycloak.RemoveKeycloak(suiteCtx, ctx)

	storages := []types.TestContext{
		{ID: utils.StorageSql, Storage: utils.StorageSql, RegistryNamespace: namespace, Auth: true, KeycloakURL: keycloakURL, Size: types.SmallSize},
		{ID: utils.StorageKafkaSql, Storage: utils.StorageKafkaSql, RegistryNamespace: namespace, Auth: true, KeycloakURL: keycloakURL, Size: types.SmallSize},
	}

	for _, s := range storages {
		runStorageSecurityTest(suiteCtx, &s)
	}

}

func runStorageSecurityTest(suiteCtx *types.SuiteContext, s *types.TestContext) {
	logs.PrintSeparator()
	log.Info("Running security test", "storage", s.Storage)

	defer func() {
		logs.PrintSeparator()
		logs.SaveLogs(suiteCtx, s)
		deploy.RemoveRegistryDeployment(suiteCtx, s)
	}()

	deploy.DeployRegistryStorage(suiteCtx, s)

	runSecurityTest(suiteCtx, s)
}

func runSecurityTest(suiteCtx *types.SuiteContext, testContext *types.TestContext) {

	// info hardcoded in kubefiles/keycloak/*.yaml
	adminUser := "registry-admin"
	adminPwd := "changeme"
	functional.BasicRegistryAPITestWithAuthentication(testContext, adminUser, adminPwd)

	devUser := "registry-developer"
	devPwd := "changeme"
	functional.BasicRegistryAPITestWithAuthentication(testContext, devUser, devPwd)

	roUser := "registry-user"
	roPwd := "changeme"
	functional.BasicRegistryAPITestWithAuthentication(testContext, roUser, roPwd)

	//TODO use the basic test for the operator and the java tests for the registry

	// authTestsCtx := &types.TestContext{
	// 	Storage:                 testContext.Storage,
	// 	FunctionalTestsProfile:  "auth",
	// 	FunctionalTestsExtraEnv: []string{
	// 		//TODO create java tests and set env vars
	// 		// "SOURCE_REGISTRY_HOST=" + sourceCtx.RegistryHost,
	// 		// "SOURCE_REGISTRY_PORT=" + sourceCtx.RegistryPort,
	// 		// "DEST_REGISTRY_HOST=" + destCtx.RegistryHost,
	// 		// "DEST_REGISTRY_PORT=" + destCtx.RegistryPort,
	// 	},
	// }
	// functional.ExecuteRegistryFunctionalTests(suiteCtx, authTestsCtx)

}

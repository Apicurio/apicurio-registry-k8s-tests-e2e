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

	// if suiteCtx.OnlyTestOperator {
	runSecurityTest(suiteCtx, s)
	// } else {
	// 	runFunctionalSecurityTests(suiteCtx, s)
	// }

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

}

//TODO fix me , java tests do client credentials flow, k8s testsuite is prepared for user/password flow
// func runFunctionalSecurityTests(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

// 	// info hardcoded in kubefiles/keycloak/*.yaml
// 	adminUser := "registry-admin"
// 	adminPwd := "changeme"

// 	devUser := "registry-developer"
// 	devPwd := "changeme"

// 	roUser := "registry-user"
// 	roPwd := "changeme"

// 	authServerUrl := ctx.RegistryResource.Spec.Configuration.Security.Keycloak.Url
// 	realm := ctx.RegistryResource.Spec.Configuration.Security.Keycloak.Realm

// 	authTestsCtx := &types.TestContext{
// 		Storage:                ctx.Storage,
// 		FunctionalTestsProfile: "auth",
// 		FunctionalTestsExtraEnv: []string{
// 			"AUTH_SERVER_URL", authServerUrl,
// 			"AUTH_REALM", realm,

// 			"AUTH_ADMIN_CLIENT_ID", adminUser,

// 			//TODO create java tests and set env vars
// 			// "SOURCE_REGISTRY_HOST=" + sourceCtx.RegistryHost,
// 			// "SOURCE_REGISTRY_PORT=" + sourceCtx.RegistryPort,
// 			// "DEST_REGISTRY_HOST=" + destCtx.RegistryHost,
// 			// "DEST_REGISTRY_PORT=" + destCtx.RegistryPort,
// 		},
// 	}
// 	functional.ExecuteRegistryFunctionalTests(suiteCtx, authTestsCtx)

// }

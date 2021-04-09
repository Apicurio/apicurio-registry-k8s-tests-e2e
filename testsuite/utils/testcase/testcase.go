package testcase

import (
	"strconv"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio/deploy"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/converters"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/functional"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/logs"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/migration"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("testcase")

//CommonTestCases declares a common set of ginkgo testcases that olm and operator bundle testsuites share
func CommonTestCases(suiteCtx *types.SuiteContext, namespace string) {
	var _ = DescribeTable("registry deployment",
		func(testContext *types.TestContext) {
			executeTestCase(suiteCtx, testContext)
		},

		Entry("sql", &types.TestContext{Storage: utils.StorageSql, RegistryNamespace: namespace}),
		Entry("kafkasql", &types.TestContext{Storage: utils.StorageKafkaSql, RegistryNamespace: namespace}),
	)

}

//BundleOnlyTestCases contains test cases that will be only executed for operator bundle installation
func BundleOnlyTestCases(suiteCtx *types.SuiteContext, namespace string) {

	if suiteCtx.DisableClusteredTests {
		log.Info("Ignoring clustered registry tests")
	} else {
		var _ = DescribeTable("clustered registry",
			func(testContext *types.TestContext) {
				executeTestCase(suiteCtx, testContext)
			},

			Entry("sql", &types.TestContext{Storage: utils.StorageSql, Replicas: 3}),
			Entry("kafkasql", &types.TestContext{Storage: utils.StorageKafkaSql, Replicas: 3, RegistryNamespace: namespace}),
		)
	}

	if suiteCtx.OnlyTestOperator {
		var _ = DescribeTable("security",
			func(testContext *types.TestContext) {
				executeTestCase(suiteCtx, testContext)
			},

			Entry("scram", &types.TestContext{Storage: utils.StorageKafkaSql, Security: "scram", RegistryNamespace: namespace}),
			Entry("tls", &types.TestContext{Storage: utils.StorageKafkaSql, Security: "tls", RegistryNamespace: namespace}),
		)
	} else {
		var _ = DescribeTable("kafka connect converters",
			func(testContext *types.TestContext) {
				executeTestOnStorage(suiteCtx, testContext, func() {
					converters.ConvertersTestCase(suiteCtx, testContext)
				})
			},

			Entry("sql", &types.TestContext{Storage: utils.StorageSql}),
		)

		var _ = DescribeTable("data migration",
			func(testContext *types.TestContext) {
				defer SaveLogsAndExecuteTestCleanups(suiteCtx, testContext)
				migration.DataMigrationTestcase(suiteCtx, testContext)
			},

			Entry("sql", &types.TestContext{Storage: utils.StorageSql, ID: utils.StorageSql, RegistryNamespace: utils.OperatorNamespace}),
			Entry("kafkasql", &types.TestContext{Storage: utils.StorageKafkaSql, ID: utils.StorageKafkaSql, RegistryNamespace: utils.OperatorNamespace}),
		)

		// var _ = It("backup and restore", func() {
		// 	ctx := &types.TestContext{}
		// 	ctx.RegistryNamespace = utils.OperatorNamespace
		// 	defer SaveLogsAndExecuteTestCleanups(suiteCtx, ctx)
		// 	sql.ExecuteBackupAndRestoreTestCase(suiteCtx, ctx)
		// })
	}

}

func MultinamespacedTestCase(suiteCtx *types.SuiteContext) {
	var _ = It("multinamespaced olm test", func() {

		var baseNamespace string = "test-multinamespace-"

		var contexts []*types.TestContext = []*types.TestContext{}
		for i := 1; i <= 2; i++ {
			ctx := &types.TestContext{
				ID:                baseNamespace + strconv.Itoa(i),
				RegistryNamespace: baseNamespace + strconv.Itoa(i),
				Storage:           utils.StorageSql,
			}
			contexts = append(contexts, ctx)

			kubernetesutils.CreateTestNamespace(suiteCtx.Clientset, ctx.RegistryNamespace)
		}

		cleanup := func() {
			for i := range contexts {
				defer kubernetesutils.DeleteTestNamespace(suiteCtx.Clientset, contexts[i].RegistryNamespace)
				contexts[i].RegisterCleanup(func() {
					deploy.RemoveRegistryDeployment(suiteCtx, contexts[i])
				})
				SaveLogsAndExecuteTestCleanups(suiteCtx, contexts[i])
			}
		}

		defer cleanup()

		for i := range contexts {
			printSeparator()
			deploy.DeployRegistryStorage(suiteCtx, contexts[i])
			printSeparator()
			functional.BasicRegistryAPITest(contexts[i])
		}

	})
}

//ExecuteTestCase common logic to test operator deploying an instance of ApicurioRegistry with one of it's storage variants
func executeTestCase(suiteCtx *types.SuiteContext, testContext *types.TestContext) {
	executeTestOnStorage(suiteCtx, testContext, func() {
		if !suiteCtx.OnlyTestOperator {
			functional.ExecuteRegistryFunctionalTests(suiteCtx, testContext)
		} else {
			functional.BasicRegistryAPITest(testContext)
		}
	})
}

//ExecuteTestOnStorage extensible logic to test apicurio registry functionality deployed with one of it's storage variants
func executeTestOnStorage(suiteCtx *types.SuiteContext, testContext *types.TestContext, testFunction func()) {
	// if testContext.ID == "" {
	// 	testContext.ID = testContext.Storage
	// }

	//implement here support for multiple namespaces
	if testContext.RegistryNamespace == "" {
		testContext.RegistryNamespace = utils.OperatorNamespace
	}

	testContext.RegisterCleanup(func() {
		deploy.RemoveRegistryDeployment(suiteCtx, testContext)
	})

	defer SaveLogsAndExecuteTestCleanups(suiteCtx, testContext)

	deploy.DeployRegistryStorage(suiteCtx, testContext)
	printSeparator()
	testFunction()
}

func SaveLogsAndExecuteTestCleanups(suiteCtx *types.SuiteContext, ctx *types.TestContext) {
	printSeparator()
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

func printSeparator() {
	log.Info("-----------------------------------------------------------")
}

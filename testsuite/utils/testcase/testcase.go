package testcase

import (
	"strconv"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	. "github.com/onsi/ginkgo"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio/deploy"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/converters"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/functional"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kafkasql"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/logs"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/security"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("testcase")

//CommonTestCases declares a common set of ginkgo testcases that olm and operator bundle testsuites share
func CommonTestCases(suiteCtx *types.SuiteContext, namespace string) {
	var size types.DeploymentSize = types.NormalSize
	if !suiteCtx.IsOpenshift {
		size = types.SmallSize
	}
	var _ = DescribeTable("registry deployment",
		func(testContext *types.TestContext) {
			executeTestCase(suiteCtx, testContext)
		},

		Entry("storage-sql", &types.TestContext{Storage: utils.StorageSql, RegistryNamespace: namespace, Size: size}),
		Entry("storage-kafkasq", &types.TestContext{Storage: utils.StorageKafkaSql, RegistryNamespace: namespace, Size: size}),
	)

	if suiteCtx.OnlyTestOperator {
		if suiteCtx.DisableAuthTests {
			log.Info("Ignoring Keycloak Authentication tests")
		} else {
			//TODO FIXME run on k8s as well
			var _ = It("keycloak", func() {
				if suiteCtx.IsOpenshift {
					security.Testcase(suiteCtx, namespace)
				}
			})
		}
	}

}

//BundleOnlyTestCases contains test cases that will be only executed for operator bundle installation
func AdvancedTestCases(suiteCtx *types.SuiteContext, namespace string) {

	if suiteCtx.DisableClusteredTests {
		log.Info("Ignoring clustered registry tests")
	} else {
		var _ = DescribeTable("clustered registry",
			func(testContext *types.TestContext) {
				if !suiteCtx.IsOpenshift && testContext.Storage == utils.StorageKafkaSql {
					//FIXME skipping this out temporarily because of weird ImagePullBackOff errors when running the second testcase
					//TODO investigate some more
					return
				}
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

			Entry("scram", &types.TestContext{Storage: utils.StorageKafkaSql, KafkaSecurity: types.Scram, RegistryNamespace: namespace}),
			Entry("tls", &types.TestContext{Storage: utils.StorageKafkaSql, KafkaSecurity: types.Tls, RegistryNamespace: namespace}),
		)
	} else {
		if suiteCtx.DisableConvertersTests {
			log.Info("Ignoring converters tests")
		} else {
			var _ = DescribeTable("kafka connect converters",
				func(testContext *types.TestContext) {
					executeTestOnStorage(suiteCtx, testContext, func() {
						converters.ConvertersTestCase(suiteCtx, testContext)
					})
				},

				Entry("sql", &types.TestContext{Storage: utils.StorageSql}),
			)
		}

		//migration tests are executed and managed by integration tests testsuite, not longer supported to run from k8s testsuite
		// var _ = DescribeTable("data migration",
		// 	func(testContext *types.TestContext) {
		// 		defer SaveLogsAndExecuteTestCleanups(suiteCtx, testContext)
		// 		migration.DataMigrationTestcase(suiteCtx, testContext)
		// 	},

		// 	Entry("sql", &types.TestContext{Storage: utils.StorageSql, ID: utils.StorageSql, RegistryNamespace: utils.OperatorNamespace}),
		// 	Entry("kafkasql", &types.TestContext{Storage: utils.StorageKafkaSql, ID: utils.StorageKafkaSql, RegistryNamespace: utils.OperatorNamespace}),
		// )

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
			logs.PrintSeparator()
			deploy.DeployRegistryStorage(suiteCtx, contexts[i])
			logs.PrintSeparator()
			functional.BasicRegistryAPITest(contexts[i])
		}

	})
}

//ExecuteTestCase common logic to test operator deploying an instance of ApicurioRegistry with one of it's storage variants
func executeTestCase(suiteCtx *types.SuiteContext, testContext *types.TestContext) {
	executeTestOnStorage(suiteCtx, testContext, func() {
		if !suiteCtx.OnlyTestOperator {
			//shared kafka deployment in k8s
			sharedKafkaCluster := kafkasql.DeploySharedKafkaIfNeeded(suiteCtx, testContext)
			if sharedKafkaCluster != nil {
				defer kafkasql.RemoveSharedKafkaIfNeeded(suiteCtx, testContext, sharedKafkaCluster)
				testContext.FunctionalTestsSharedKafkaCluster = sharedKafkaCluster
			}
			functional.ExecuteRegistryFunctionalTests(suiteCtx, testContext)
		} else {
			functional.BasicRegistryAPITest(testContext)
		}
	})
}

//ExecuteTestOnStorage extensible logic to test apicurio registry functionality deployed with one of it's storage variants
func executeTestOnStorage(suiteCtx *types.SuiteContext, testContext *types.TestContext, testFunction func()) {
	//implement here support for multiple namespaces
	if testContext.RegistryNamespace == "" {
		testContext.RegistryNamespace = utils.OperatorNamespace
	}

	testContext.RegisterCleanup(func() {
		deploy.RemoveRegistryDeployment(suiteCtx, testContext)
	})

	defer SaveLogsAndExecuteTestCleanups(suiteCtx, testContext)

	deploy.DeployRegistryStorage(suiteCtx, testContext)
	logs.PrintSeparator()
	testFunction()
}

func SaveLogsAndExecuteTestCleanups(suiteCtx *types.SuiteContext, ctx *types.TestContext) {
	logs.PrintSeparator()

	logs.SaveLogs(suiteCtx, ctx)

	ctx.ExecuteCleanups()
}

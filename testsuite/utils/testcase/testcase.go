package testcase

import (
	"errors"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/converters"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/functional"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/infinispan"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/jpa"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/logs"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/streams"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("testcase")

//CommonTestCases declares a common set of ginkgo testcases that olm and operator bundle testsuites share
func CommonTestCases(suiteCtx *types.SuiteContext) {
	var _ = DescribeTable("registry deployment",
		func(testContext *types.TestContext) {
			executeTestCase(suiteCtx, testContext)
		},

		//TODO revert all jpa comments when operator supports SQL persistence
		// Entry("jpa", &types.TestContext{Storage: utils.StorageJpa}),
		Entry("streams", &types.TestContext{Storage: utils.StorageStreams}),
		Entry("infinispan", &types.TestContext{Storage: utils.StorageInfinispan}),
	)

}

//BundleOnlyTestCases contains test cases that will be only executed for operator bundle installation
func BundleOnlyTestCases(suiteCtx *types.SuiteContext) {

	if suiteCtx.DisableClusteredTests {
		log.Info("Ignoring clustered registry tests")
	} else {
		var _ = DescribeTable("clustered registry",
			func(testContext *types.TestContext) {
				executeTestCase(suiteCtx, testContext)
			},

			// Entry("jpa", &types.TestContext{Storage: utils.StorageJpa, Replicas: 3}),
			Entry("streams", &types.TestContext{Storage: utils.StorageStreams, Replicas: 3}),
			Entry("infinispan", &types.TestContext{Storage: utils.StorageInfinispan, Replicas: 3}),
		)
	}

	if !suiteCtx.OnlyTestOperator {
		var _ = DescribeTable("kafka connect converters",
			func(testContext *types.TestContext) {
				executeConvertersTestCase(suiteCtx, testContext)
			},

			// Entry("postgres", &types.TestContext{Storage: utils.StorageJpa}),
			// Entry("streams", &types.TestContext{Storage: utils.StorageStreams}),
			Entry("infinispan", &types.TestContext{Storage: utils.StorageInfinispan}),
		)
	}

	// var _ = It("backup and restore", func() {
	// 	ctx := &types.TestContext{}
	// 	ctx.RegistryNamespace = utils.OperatorNamespace
	// 	defer SaveLogsAndExecuteTestCleanups(suiteCtx, ctx)
	// 	jpa.ExecuteBackupAndRestoreTestCase(suiteCtx, ctx)
	// })

	var _ = DescribeTable("streams security",
		func(testContext *types.TestContext) {
			executeTestCase(suiteCtx, testContext)
		},

		Entry("scram", &types.TestContext{Storage: utils.StorageStreams, Security: "scram"}),
		Entry("tls", &types.TestContext{Storage: utils.StorageStreams, Security: "tls"}),
	)

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

func executeConvertersTestCase(suiteCtx *types.SuiteContext, testContext *types.TestContext) {
	executeTestOnStorage(suiteCtx, testContext, func() {
		converters.ConvertersTestCase(suiteCtx, testContext)
	})
}

//ExecuteTestOnStorage extensible logic to test apicurio registry functionality deployed with one of it's storage variants
func executeTestOnStorage(suiteCtx *types.SuiteContext, testContext *types.TestContext, testFunction func()) {
	if testContext.ID == "" {
		testContext.ID = testContext.Storage
	}

	//implement here support for multiple namespaces
	if testContext.RegistryNamespace == "" {
		testContext.RegistryNamespace = utils.OperatorNamespace
	}

	defer CleanRegistryDeployment(suiteCtx, testContext)

	DeployRegistryStorage(suiteCtx, testContext)
	log.Info("-----------------------------------------------------------")
	testFunction()
}

func DeployRegistryStorage(suiteCtx *types.SuiteContext, ctx *types.TestContext) {
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
func CleanRegistryDeployment(suiteCtx *types.SuiteContext, ctx *types.TestContext) error {

	SaveLogsAndExecuteTestCleanups(suiteCtx, ctx)

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

func SaveLogsAndExecuteTestCleanups(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	log.Info("-----------------------------------------------------------")

	testDescription := CurrentGinkgoTestDescription()
	logs.SaveTestPodsLogs(suiteCtx.Clientset, suiteCtx.SuiteID, ctx.RegistryNamespace, testDescription)

	log.Info("Executing cleanups")

	ctx.ExecuteCleanups()
}

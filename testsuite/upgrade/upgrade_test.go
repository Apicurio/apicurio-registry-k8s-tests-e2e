package olm

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	apicurioutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio"
	apicurioclient "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio/client"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio/deploy"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/functional"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/olm"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/testcase"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var artifactData string = "{\"type\":\"record\",\"name\":\"price\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"symbol\",\"type\":\"string\"},{\"name\":\"price\",\"type\":\"string\"}]}"

var _ = DescribeTable("olm-upgrade",
	func(ctx *types.TestContext) {
		defer testcase.SaveLogsAndExecuteTestCleanups(suiteCtx, ctx)
		executeUpgradeTest(suiteCtx, ctx)
	},

	// Entry("sql", &types.TestContext{Storage: utils.StorageSql}),
	// Entry("streams", &types.TestContext{Storage: utils.StorageStreams}),
)

func executeUpgradeTest(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	//inputs

	channel := utils.OLMUpgradeChannel

	startingCatalogSource := utils.OLMUpgradeOldCatalog
	startingCatalogSourceNamespace := utils.OLMUpgradeOldCatalogNamespace
	startingCSV := utils.OLMUpgradeOldCSV

	//upgrade catalog source from utils.OLMCatalogSourceImage
	upgradeCSV := utils.OLMUpgradeNewCSV

	//test actions

	//create test namespace
	const operatorNamespace string = utils.OperatorNamespace
	ctx.RegistryNamespace = operatorNamespace

	ctx.RegisterCleanup(func() {
		kubernetesutils.DeleteTestNamespace(suiteCtx.Clientset, operatorNamespace)
	})
	kubernetesutils.CreateTestNamespace(suiteCtx.Clientset, operatorNamespace)

	//create operator group
	const operatorGroupName string = "apicurio-registry-operator-group"

	olm.CreateOperatorGroup(suiteCtx, operatorNamespace, operatorGroupName)
	ctx.RegisterCleanup(func() {
		olm.DeleteOperatorGroup(suiteCtx, operatorNamespace, operatorGroupName)
	})

	//TODO verify if starting catalog source needs to be deployed

	//create starting subscription
	const operatorSubscriptionName string = "registry-upgrade-sub"

	sub := olm.CreateSubscription(suiteCtx, &olm.CreateSubscriptionRequest{
		SubscriptionName:      operatorSubscriptionName,
		SubscriptionNamespace: operatorNamespace,

		Package:                utils.OLMApicurioPackageManifestName,
		CatalogSourceName:      startingCatalogSource,
		CatalogSourceNamespace: startingCatalogSourceNamespace,

		ChannelName: channel,
		ChannelCSV:  startingCSV,
	})
	kubernetesutils.WaitForOperatorDeploymentReady(suiteCtx.Clientset, sub.Namespace)

	ctx.RegisterCleanup(func() {
		olm.DeleteSubscription(suiteCtx, sub, true)
	})

	//deploy registry and run smoke tests
	kubernetescli.GetPods(operatorNamespace)

	deploy.DeployRegistryStorage(suiteCtx, ctx)

	// functional.ExecuteRegistryFunctionalTests(suiteCtx, ctx)

	functional.BasicRegistryAPITest(ctx)

	//create artifacts on the registry
	log.Info("Creating test artifacts")
	registryClient := apicurioclient.NewApicurioRegistryApiClient(ctx.RegistryHost, ctx.RegistryPort, http.DefaultClient)

	for i := 1; i <= 50; i++ {
		err := registryClient.CreateArtifact("upgrd-"+strconv.Itoa(i), apicurioclient.Avro, artifactData)
		Expect(err).ToNot(HaveOccurred())
		time.Sleep(1 * time.Second)
	}

	artifacts, err := registryClient.ListArtifacts()
	Expect(err).ToNot(HaveOccurred())
	Expect(len(artifacts)).To(BeIdenticalTo(50))
	log.Info(strconv.Itoa(len(artifacts)) + "test artifacts created")

	//deploy new catalog source
	const catalogSourceName string = "registry-upgrade-catalog"
	// utils.OLMCatalogSourceImage //catalog image via this env var

	olm.CreateCatalogSource(suiteCtx, operatorNamespace, catalogSourceName)
	ctx.RegisterCleanup(func() {
		olm.DeleteCatalogSource(suiteCtx, operatorNamespace, catalogSourceName)
	})

	//update subscription to point new catalog, do not change csv nor channel, new catalog will trigger that update
	log.Info("Updating operator subscription", "catalog", catalogSourceName)
	oldsub, err := suiteCtx.OLMClient.OperatorsV1alpha1().Subscriptions(operatorNamespace).Get(context.TODO(), sub.Name, v1.GetOptions{})
	oldsub.Spec.CatalogSource = catalogSourceName
	oldsub.Spec.CatalogSourceNamespace = operatorNamespace
	updatedsub, err := suiteCtx.OLMClient.OperatorsV1alpha1().Subscriptions(operatorNamespace).Update(context.TODO(), oldsub, v1.UpdateOptions{})
	if err != nil {
		kubernetescli.GetPods(operatorNamespace)
	}
	Expect(err).ToNot(HaveOccurred())

	//wait for subscription to point to new CSV
	timeout := 120 * time.Second
	log.Info("Waiting for subscription to be updated", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		updatedsub, err = suiteCtx.OLMClient.OperatorsV1alpha1().Subscriptions(sub.Namespace).Get(context.TODO(), sub.Name, v1.GetOptions{})
		if err != nil {
			return false, err
		}
		if updatedsub.Status.CurrentCSV != "" && updatedsub.Status.CurrentCSV == upgradeCSV {
			return true, nil
		}
		return false, nil
	})
	kubernetescli.GetPods(sub.Namespace)
	Expect(err).ToNot(HaveOccurred())

	ctx.RegisterCleanup(func() {
		updatedsub.Spec.StartingCSV = upgradeCSV
		olm.DeleteSubscription(suiteCtx, updatedsub, true)
	})

	//wait for new csv to be created
	timeout = 160 * time.Second
	log.Info("Waiting for new csv to be created and ready", "timeout", timeout)
	lastPhase := ""
	err = wait.Poll(utils.MediumPollInterval, timeout, func() (bool, error) {
		newcsv, err := suiteCtx.OLMClient.OperatorsV1alpha1().ClusterServiceVersions(sub.Namespace).Get(context.TODO(), upgradeCSV, v1.GetOptions{})
		if err != nil {
			if kubeerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if newcsv.Status.Phase == v1alpha1.CSVPhaseFailed {
			return false, errors.New("CSV failed")
		}
		if newcsv.Status.Phase == v1alpha1.CSVPhaseSucceeded {
			log.Info("CSV Succeeded")
			return true, nil
		}
		if lastPhase == "" || lastPhase != string(newcsv.Status.Phase) {
			lastPhase = string(newcsv.Status.Phase)
			log.Info("CSV Phase " + string(newcsv.Status.Phase))
		}
		return false, nil
	})
	kubernetescli.Execute("get", "csv", upgradeCSV, "-n", sub.Namespace, "-o", "wide")
	kubernetescli.Execute("get", "installplan", "-n", sub.Namespace)
	Expect(err).ToNot(HaveOccurred())

	// kubernetescli.Execute("get", "apicurioregistry", "-o", "yaml")

	//wait for deployments
	kubernetesutils.WaitForOperatorDeploymentReady(suiteCtx.Clientset, sub.Namespace)
	apicurioutils.WaitForRegistryReady(suiteCtx, ctx.RegistryNamespace, ctx.RegistryName, int32(ctx.Replicas))

	//verify artifacts after upgrade
	functional.BasicRegistryAPITest(ctx)

	log.Info("Verifiying test artifacts")
	artifacts, err = registryClient.ListArtifacts()
	Expect(err).ToNot(HaveOccurred())
	Expect(len(artifacts)).To(BeIdenticalTo(50))

}

package olm

import (
	"errors"
	"time"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/logs"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/olm"
	testcase "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/testcase"

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	packagev1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
)

var _ = Describe("olm installation", func() {

	testcase.CommonTestCases(suiteCtx)

})

func logPodsAll(operatorNamespace string) {
	kubernetescli.Execute("get", "pod", "-n", operatorNamespace, "-o", "yaml")
}

type OLMInstallationInfo struct {
	CatalogSource *operatorsv1alpha1.CatalogSource
	OperatorGroup *operatorsv1.OperatorGroup
	Subscription  *operatorsv1alpha1.Subscription
}

func installOperatorOLM() *OLMInstallationInfo {

	if utils.OLMCatalogSourceImage == "" {
		Expect(errors.New("Env var " + utils.OLMCatalogSourceImageEnvVar + " is required")).ToNot(HaveOccurred())
	}

	var catalogSourceNamespace string = utils.OLMCatalogSourceNamespace
	err := kubernetesutils.CreateNamespace(suiteCtx.Clientset, catalogSourceNamespace)
	if !kubeerrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	log.Info("Using catalog source namespace " + catalogSourceNamespace)

	const operatorSubscriptionName string = "apicurio-registry-sub"
	const operatorGroupName string = "apicurio-registry-operator-group"
	const catalogSourceName string = "apicurio-registry-catalog"
	const operatorNamespace string = utils.OperatorNamespace

	kubernetesutils.CreateTestNamespace(suiteCtx.Clientset, operatorNamespace)

	//catalog-source
	catalog := olm.CreateCatalogSource(suiteCtx, catalogSourceNamespace, catalogSourceName)

	//operator-group
	operatorGroup := olm.CreateOperatorGroup(suiteCtx, operatorNamespace, operatorGroupName)

	//subscription

	//TODO make this timeout configurable
	timeout := 540 * time.Second
	log.Info("Waiting for package manifest to be available", "timeout", timeout)
	var packageManifest *packagev1.PackageManifest = nil
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"catalog": catalogSourceName})
		pkgsList, err := suiteCtx.PackageClient.OperatorsV1().PackageManifests(catalogSourceNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
		if err != nil && !kubeerrors.IsNotFound(err) {
			return false, err
		}
		if pkgsList != nil {
			pkg := findApicurioPackageManifest(pkgsList)
			if pkg != nil {
				packageManifest = pkg
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		logPodsAll(operatorNamespace)
		kubernetescli.Execute("get", "packagemanifest")
	}
	Expect(err).ToNot(HaveOccurred())

	// labelsSet := labels.Set(map[string]string{"catalog": catalogSourceName})
	// pkgsList, err := suiteCtx.PackageClient.OperatorsV1().PackageManifests(catalogSourceNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	// Expect(err).ToNot(HaveOccurred())
	// var packageManifest *packagev1.PackageManifest = findApicurioPackageManifest(pkgsList)
	Expect(packageManifest).ToNot(BeNil())
	var channelName string = packageManifest.Status.DefaultChannel
	var channelCSV string
	for _, channel := range packageManifest.Status.Channels {
		if channel.Name == channelName {
			channelCSV = channel.CurrentCSV
		}
	}
	Expect(channelCSV).NotTo(BeNil())

	sub := olm.CreateSubscription(suiteCtx, &olm.CreateSubscriptionRequest{
		SubscriptionName:       operatorSubscriptionName,
		SubscriptionNamespace:  operatorNamespace,
		CatalogSourceName:      catalogSourceName,
		CatalogSourceNamespace: catalogSourceNamespace,
		ChannelCSV:             channelCSV,
		ChannelName:            channelName,
	})

	return &OLMInstallationInfo{
		CatalogSource: catalog,
		OperatorGroup: operatorGroup,
		Subscription:  sub,
	}

}

func uninstallOperatorOLM(olminfo *OLMInstallationInfo) {

	logs.SaveOperatorLogs(suiteCtx.Clientset, suiteCtx.SuiteID, olminfo.Subscription.Namespace)

	log.Info("Uninstalling operator")

	olm.DeleteSubscription(suiteCtx, olminfo.Subscription)

	olm.DeleteOperatorGroup(suiteCtx, olminfo.OperatorGroup.Namespace, olminfo.OperatorGroup.Name)

	olm.DeleteCatalogSource(suiteCtx, olminfo.CatalogSource.Namespace, olminfo.CatalogSource.Name)

	//TODO verify, this namespace may change in the future
	kubernetesutils.DeleteTestNamespace(suiteCtx.Clientset, olminfo.Subscription.Namespace)

}

func findApicurioPackageManifest(pkgsList *packagev1.PackageManifestList) *packagev1.PackageManifest {
	for _, pkg := range pkgsList.Items {
		if pkg.Name == utils.OLMApicurioPackageManifestName {
			return &pkg
		}
	}
	return nil
}

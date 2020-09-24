package olm

import (
	"time"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	testcase "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/testcase"

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	packagev1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
)

var _ = Describe("olm installation", func() {

	testcase.CommonTestCases(suiteCtx)

})

const operatorSubscriptionName string = "apicurio-registry-sub"
const operatorGroupName string = "apicurio-registry-operator-group"
const catalogSourceName string = "apicurio-registry-catalog"
const catalogSourceNamespace string = utils.OperatorNamespace

var operatorCSV string

func installOperatorOLM() {
	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	utils.CreateTestNamespace(clientset, utils.OperatorNamespace)

	//catalog-source
	_, err := suiteCtx.OLMClient.OperatorsV1alpha1().CatalogSources(catalogSourceNamespace).Create(&operatorsv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalogSourceName,
			Namespace: catalogSourceNamespace,
		},
		Spec: operatorsv1alpha1.CatalogSourceSpec{
			DisplayName: "Apicurio Registry Operator Catalog Source",
			Image:       utils.OLMCatalogSourceImage,
			Publisher:   "apicurio-registry-qe",
			SourceType:  operatorsv1alpha1.SourceTypeGrpc,
		},
	})
	Expect(err).ToNot(HaveOccurred())

	timeout := 120 * time.Second
	log.Info("Waiting for catalog source to be ready", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		catalogSource, err := suiteCtx.OLMClient.OperatorsV1alpha1().CatalogSources(catalogSourceNamespace).Get(catalogSourceName, metav1.GetOptions{})
		if err != nil && !kubeerrors.IsNotFound(err) {
			return false, err
		}
		if catalogSource != nil {
			if catalogSource.Status.GRPCConnectionState.LastObservedState == "READY" {
				return true, nil
			}
		}
		return false, nil
	})
	Expect(err).ToNot(HaveOccurred())

	//operator-group
	log.Info("Creating operator group")
	_, err = suiteCtx.OLMClient.OperatorsV1().OperatorGroups(utils.OperatorNamespace).Create(&operatorsv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorGroupName,
			Namespace: utils.OperatorNamespace,
		},
		Spec: operatorsv1.OperatorGroupSpec{
			TargetNamespaces: []string{utils.OperatorNamespace},
		},
	})
	Expect(err).ToNot(HaveOccurred())

	//subscription
	timeout = 30 * time.Second
	log.Info("Waiting for package manifest to be available", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"catalog": catalogSourceName})
		pkgsList, err := suiteCtx.PackageClient.OperatorsV1().PackageManifests(catalogSourceNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
		if err != nil && !kubeerrors.IsNotFound(err) {
			return false, err
		}
		if len(pkgsList.Items) == 1 {
			return true, nil
		}
		return false, nil
	})
	Expect(err).ToNot(HaveOccurred())

	labelsSet := labels.Set(map[string]string{"catalog": catalogSourceName})
	pkgsList, err := suiteCtx.PackageClient.OperatorsV1().PackageManifests(catalogSourceNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	Expect(err).ToNot(HaveOccurred())
	Expect(len(pkgsList.Items)).To(BeIdenticalTo(1))
	var packageManifest packagev1.PackageManifest = pkgsList.Items[0]
	var packageName string = packageManifest.Name
	var channelName string = packageManifest.Status.DefaultChannel
	var channelCSV string
	for _, channel := range packageManifest.Status.Channels {
		if channel.Name == channelName {
			channelCSV = channel.CurrentCSV
		}
	}
	Expect(channelCSV).NotTo(BeNil())
	operatorCSV = channelCSV

	log.Info("Creating operator subscription", "package", packageName, "channel", channelName, "csv", channelCSV)
	_, err = suiteCtx.OLMClient.OperatorsV1alpha1().Subscriptions(utils.OperatorNamespace).Create(&operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorSubscriptionName,
			Namespace: utils.OperatorNamespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			Package:                packageName,
			CatalogSource:          catalogSourceName,
			CatalogSourceNamespace: catalogSourceNamespace,
			StartingCSV:            channelCSV,
			Channel:                channelName,
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
		},
	})
	Expect(err).ToNot(HaveOccurred())

	utils.WaitForOperatorDeploymentReady(clientset)

}

func uninstallOperatorOLM() {

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	utils.SaveOperatorLogs(clientset, suiteCtx.SuiteID)

	log.Info("Uninstalling operator")

	err := suiteCtx.OLMClient.OperatorsV1alpha1().Subscriptions(utils.OperatorNamespace).Delete(operatorSubscriptionName, &metav1.DeleteOptions{})
	if err != nil && !kubeerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	if operatorCSV != "" {
		err = suiteCtx.OLMClient.OperatorsV1alpha1().ClusterServiceVersions(utils.OperatorNamespace).Delete(operatorCSV, &metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		utils.WaitForOperatorDeploymentRemoved(clientset)
	}

	err = suiteCtx.OLMClient.OperatorsV1().OperatorGroups(utils.OperatorNamespace).Delete(operatorGroupName, &metav1.DeleteOptions{})
	if err != nil && !kubeerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	err = suiteCtx.OLMClient.OperatorsV1alpha1().CatalogSources(catalogSourceNamespace).Delete(catalogSourceName, &metav1.DeleteOptions{})
	if err != nil && !kubeerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	utils.DeleteTestNamespace(clientset, utils.OperatorNamespace)

}

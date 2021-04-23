package olm

import (
	"context"
	"errors"
	"time"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	. "github.com/onsi/gomega"
	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/logs"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

var log = logf.Log.WithName("olm-testsuite")

type CreateSubscriptionRequest struct {
	SubscriptionNamespace string
	SubscriptionName      string

	Package                string
	CatalogSourceName      string
	CatalogSourceNamespace string

	ChannelName string
	ChannelCSV  string
}

func CreateCatalogSource(suiteCtx *types.SuiteContext, catalogSourceNamespace string, catalogSourceName string) *operatorsv1alpha1.CatalogSource {
	log.Info("Creating catalog source " + catalogSourceName)
	catalog, err := suiteCtx.OLMClient.OperatorsV1alpha1().CatalogSources(catalogSourceNamespace).Create(context.TODO(), &operatorsv1alpha1.CatalogSource{
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
	}, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	timeout := 300 * time.Second
	log.Info("Waiting for catalog source", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := suiteCtx.OLMClient.OperatorsV1alpha1().CatalogSources(catalogSourceNamespace).Get(context.TODO(), catalogSourceName, metav1.GetOptions{})
		if err != nil {
			if kubeerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	kubernetescli.GetPods(catalogSourceNamespace)
	if err != nil {
		kubernetescli.Execute("get", "catalogsource", catalogSourceName, "-n", catalogSourceNamespace, "-o", "yaml")
	}
	Expect(err).ToNot(HaveOccurred())

	return catalog
}

func DeleteCatalogSource(suiteCtx *types.SuiteContext, catalogSourceNamespace string, catalogSourceName string) {
	log.Info("Removing catalog source " + catalogSourceName)
	err := suiteCtx.OLMClient.OperatorsV1alpha1().CatalogSources(catalogSourceNamespace).Delete(context.TODO(), catalogSourceName, metav1.DeleteOptions{})
	if err != nil && !kubeerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}

func CreateOperatorGroup(suiteCtx *types.SuiteContext, operatorNamespace string, operatorGroupName string) *operatorsv1.OperatorGroup {
	log.Info("Creating operator group")
	group, err := suiteCtx.OLMClient.OperatorsV1().OperatorGroups(operatorNamespace).Create(context.TODO(), &operatorsv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorGroupName,
			Namespace: operatorNamespace,
		},
		Spec: operatorsv1.OperatorGroupSpec{
			TargetNamespaces: []string{operatorNamespace},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		logPodsAll(operatorNamespace)
	}
	Expect(err).ToNot(HaveOccurred())

	return group
}

func DeleteOperatorGroup(suiteCtx *types.SuiteContext, operatorNamespace string, operatorGroupName string) {
	err := suiteCtx.OLMClient.OperatorsV1().OperatorGroups(operatorNamespace).Delete(context.TODO(), operatorGroupName, metav1.DeleteOptions{})
	if err != nil && !kubeerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}

func CreateSubscription(suiteCtx *types.SuiteContext, req *CreateSubscriptionRequest) *operatorsv1alpha1.Subscription {
	log.Info("Creating operator subscription", "package", req.Package, "channel", req.ChannelName, "csv", req.ChannelCSV)
	sub, err := suiteCtx.OLMClient.OperatorsV1alpha1().Subscriptions(req.SubscriptionNamespace).Create(context.TODO(), &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.SubscriptionName,
			Namespace: req.SubscriptionNamespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			Package:                req.Package,
			CatalogSource:          req.CatalogSourceName,
			CatalogSourceNamespace: req.CatalogSourceNamespace,
			StartingCSV:            req.ChannelCSV,
			Channel:                req.ChannelName,
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		logPodsAll(req.SubscriptionNamespace)
	}
	Expect(err).ToNot(HaveOccurred())

	return sub
}

func DeleteSubscription(suiteCtx *types.SuiteContext, sub *operatorsv1alpha1.Subscription, defaultWait bool) {
	log.Info("Going to delete subscription " + sub.Name)
	err := suiteCtx.OLMClient.OperatorsV1alpha1().Subscriptions(sub.Namespace).Delete(context.TODO(), sub.Name, metav1.DeleteOptions{})
	if err != nil && !kubeerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	if sub.Spec.StartingCSV != "" {
		log.Info("Going to delete CSV " + sub.Spec.StartingCSV)
		err = suiteCtx.OLMClient.OperatorsV1alpha1().ClusterServiceVersions(sub.Namespace).Delete(context.TODO(), sub.Spec.StartingCSV, metav1.DeleteOptions{})
		if err != nil {
			if kubeerrors.IsNotFound(err) {
				//don't wait for operator
				return
			}
			Expect(err).ToNot(HaveOccurred())
		}
		if defaultWait {
			kubernetesutils.WaitForOperatorDeploymentRemoved(suiteCtx.Clientset, sub.Namespace)
		}
	}
}

func logPodsAll(operatorNamespace string) {
	kubernetescli.Execute("get", "pod", "-n", operatorNamespace, "-o", "yaml")
}

type OLMInstallationInfo struct {
	CatalogSource *operatorsv1alpha1.CatalogSource
	OperatorGroup *operatorsv1.OperatorGroup
	Subscription  *operatorsv1alpha1.Subscription
	clusterwide   bool
}

func InstallOperatorOLM(suiteCtx *types.SuiteContext, operatorNamespace string, clusterwide bool) *OLMInstallationInfo {

	if utils.OLMCatalogSourceImage == "" {
		Expect(errors.New("OLM catalog source image env var is required")).ToNot(HaveOccurred())
	}

	const operatorSubscriptionName string = "apicurio-registry-sub"
	const operatorGroupName string = "apicurio-registry-operator-group"
	const catalogSourceName string = "apicurio-registry-catalog"

	if !clusterwide {
		kubernetesutils.CreateTestNamespace(suiteCtx.Clientset, operatorNamespace)
	}

	var catalogSourceNamespace string = utils.OLMCatalogSourceNamespace
	err := kubernetesutils.CreateNamespace(suiteCtx.Clientset, catalogSourceNamespace)
	if !kubeerrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	log.Info("Using catalog source namespace " + catalogSourceNamespace)

	//catalog-source
	catalog := CreateCatalogSource(suiteCtx, catalogSourceNamespace, catalogSourceName)

	var operatorGroup *operatorsv1.OperatorGroup = nil
	if !clusterwide {
		//operator-group
		operatorGroup = CreateOperatorGroup(suiteCtx, operatorNamespace, operatorGroupName)
	}

	//subscription

	//TODO make this timeout configurable
	timeout := 540 * time.Second
	log.Info("Waiting for package manifest to be available", "timeout", timeout)
	var packageManifest *v1.PackageManifest = nil
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"catalog": catalogSourceName})
		pkgsList, err := suiteCtx.PackageClient.OperatorsV1().PackageManifests(catalogSourceNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
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

	Expect(packageManifest).ToNot(BeNil())

	var channelName string = packageManifest.Status.DefaultChannel
	if utils.OLMApicurioChannelName != "" {
		channelName = utils.OLMApicurioChannelName
	}

	var channelCSV string
	for _, channel := range packageManifest.Status.Channels {
		if channel.Name == channelName {
			channelCSV = channel.CurrentCSV
		}
	}
	Expect(channelCSV).NotTo(BeNil())

	sub := CreateSubscription(suiteCtx, &CreateSubscriptionRequest{
		SubscriptionName:       operatorSubscriptionName,
		SubscriptionNamespace:  operatorNamespace,
		Package:                utils.OLMApicurioPackageManifestName,
		CatalogSourceName:      catalogSourceName,
		CatalogSourceNamespace: catalogSourceNamespace,
		ChannelCSV:             channelCSV,
		ChannelName:            channelName,
	})
	kubernetesutils.WaitForOperatorDeploymentReady(suiteCtx.Clientset, sub.Namespace)

	return &OLMInstallationInfo{
		CatalogSource: catalog,
		OperatorGroup: operatorGroup,
		Subscription:  sub,
		clusterwide:   clusterwide,
	}

}

func UninstallOperatorOLM(suiteCtx *types.SuiteContext, olminfo *OLMInstallationInfo) {

	logs.SaveOperatorLogs(suiteCtx.Clientset, suiteCtx.SuiteID, olminfo.Subscription.Namespace)

	log.Info("Uninstalling operator")

	DeleteSubscription(suiteCtx, olminfo.Subscription, true)

	if olminfo.OperatorGroup != nil {
		DeleteOperatorGroup(suiteCtx, olminfo.OperatorGroup.Namespace, olminfo.OperatorGroup.Name)
	}

	DeleteCatalogSource(suiteCtx, olminfo.CatalogSource.Namespace, olminfo.CatalogSource.Name)

	if !olminfo.clusterwide {
		kubernetesutils.DeleteTestNamespace(suiteCtx.Clientset, olminfo.Subscription.Namespace)
	}

}

func findApicurioPackageManifest(pkgsList *v1.PackageManifestList) *v1.PackageManifest {
	for _, pkg := range pkgsList.Items {
		if pkg.Name == utils.OLMApicurioPackageManifestName {
			return &pkg
		}
	}
	return nil
}

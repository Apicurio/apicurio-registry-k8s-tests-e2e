package olm

import (
	"time"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

var log = logf.Log.WithName("olm-testsuite")

type CreateSubscriptionRequest struct {
	SubscriptionNamespace string
	SubscriptionName      string

	CatalogSourceName      string
	CatalogSourceNamespace string

	ChannelName string
	ChannelCSV  string
}

func CreateCatalogSource(suiteCtx *types.SuiteContext, catalogSourceNamespace string, catalogSourceName string) *operatorsv1alpha1.CatalogSource {
	log.Info("Creating catalog source " + catalogSourceName)
	catalog, err := suiteCtx.OLMClient.OperatorsV1alpha1().CatalogSources(catalogSourceNamespace).Create(&operatorsv1alpha1.CatalogSource{
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

	timeout := 300 * time.Second
	log.Info("Waiting for catalog source", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := suiteCtx.OLMClient.OperatorsV1alpha1().CatalogSources(catalogSourceNamespace).Get(catalogSourceName, metav1.GetOptions{})
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
	err := suiteCtx.OLMClient.OperatorsV1alpha1().CatalogSources(catalogSourceNamespace).Delete(catalogSourceName, &metav1.DeleteOptions{})
	if err != nil && !kubeerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}

func CreateOperatorGroup(suiteCtx *types.SuiteContext, operatorNamespace string, operatorGroupName string) *operatorsv1.OperatorGroup {
	log.Info("Creating operator group")
	group, err := suiteCtx.OLMClient.OperatorsV1().OperatorGroups(operatorNamespace).Create(&operatorsv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorGroupName,
			Namespace: operatorNamespace,
		},
		Spec: operatorsv1.OperatorGroupSpec{
			TargetNamespaces: []string{operatorNamespace},
		},
	})
	if err != nil {
		logPodsAll(operatorNamespace)
	}
	Expect(err).ToNot(HaveOccurred())

	return group
}

func DeleteOperatorGroup(suiteCtx *types.SuiteContext, operatorNamespace string, operatorGroupName string) {
	err := suiteCtx.OLMClient.OperatorsV1().OperatorGroups(operatorNamespace).Delete(operatorGroupName, &metav1.DeleteOptions{})
	if err != nil && !kubeerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}

func CreateSubscription(suiteCtx *types.SuiteContext, req *CreateSubscriptionRequest) *operatorsv1alpha1.Subscription {
	log.Info("Creating operator subscription", "package", utils.OLMApicurioPackageManifestName, "channel", req.ChannelName, "csv", req.ChannelCSV)
	sub, err := suiteCtx.OLMClient.OperatorsV1alpha1().Subscriptions(req.SubscriptionNamespace).Create(&operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.SubscriptionName,
			Namespace: req.SubscriptionNamespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			Package:                utils.OLMApicurioPackageManifestName,
			CatalogSource:          req.CatalogSourceName,
			CatalogSourceNamespace: req.CatalogSourceNamespace,
			StartingCSV:            req.ChannelCSV,
			Channel:                req.ChannelName,
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
		},
	})
	if err != nil {
		logPodsAll(req.SubscriptionNamespace)
	}
	Expect(err).ToNot(HaveOccurred())

	kubernetesutils.WaitForOperatorDeploymentReady(suiteCtx.Clientset, req.SubscriptionNamespace)

	return sub
}

func DeleteSubscription(suiteCtx *types.SuiteContext, sub *operatorsv1alpha1.Subscription) {
	log.Info("Going to delete subscription " + sub.Name)
	err := suiteCtx.OLMClient.OperatorsV1alpha1().Subscriptions(sub.Namespace).Delete(sub.Name, &metav1.DeleteOptions{})
	if err != nil && !kubeerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	if sub.Spec.StartingCSV != "" {
		log.Info("Going to delete CSV " + sub.Spec.StartingCSV)
		err = suiteCtx.OLMClient.OperatorsV1alpha1().ClusterServiceVersions(sub.Namespace).Delete(sub.Spec.StartingCSV, &metav1.DeleteOptions{})
		if err != nil {
			if kubeerrors.IsNotFound(err) {
				//don't wait for operator
				return
			}
			Expect(err).ToNot(HaveOccurred())
		}
		kubernetesutils.WaitForOperatorDeploymentRemoved(suiteCtx.Clientset, sub.Namespace)
	}
}

func logPodsAll(operatorNamespace string) {
	kubernetescli.Execute("get", "pod", "-n", operatorNamespace, "-o", "yaml")
}

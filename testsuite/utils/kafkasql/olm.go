package kafkasql

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/olm"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

func deployStrimziOperatorOLM(suiteCtx *types.SuiteContext, namespace string) (*operatorsv1alpha1.Subscription, *operatorsv1.OperatorGroup) {

	var operatorGroupName string = namespace + "-operator-group"

	var og *operatorsv1.OperatorGroup = nil
	if olm.AnyOperatorGroupExists(suiteCtx, namespace) {
		log.Info("Skipping operator group creation because it already exists")
	} else {
		og = olm.CreateOperatorGroup(suiteCtx, namespace, operatorGroupName)
	}

	var sr *olm.CreateSubscriptionRequest
	sr = &olm.CreateSubscriptionRequest{
		SubscriptionNamespace:  namespace,
		SubscriptionName:       "strimzi-kafka-operator",
		Package:                "strimzi-kafka-operator",
		CatalogSourceName:      "operatorhubio-catalog",
		CatalogSourceNamespace: "olm",
		ChannelName:            "stable",
		ChannelCSV:             "0.23.0",
	}
	if suiteCtx.IsOpenshift {
		sr = &olm.CreateSubscriptionRequest{
			SubscriptionNamespace:  namespace,
			SubscriptionName:       "amq-streams",
			Package:                "amq-streams",
			CatalogSourceName:      "redhat-operators",
			CatalogSourceNamespace: "openshift-marketplace",
			ChannelName:            "amq-streams-1.x",
		}
	}

	packageManifest, err := suiteCtx.PackageClient.OperatorsV1().PackageManifests(sr.CatalogSourceNamespace).Get(context.TODO(), sr.Package, metav1.GetOptions{})

	Expect(err).ToNot(HaveOccurred())

	channelName := packageManifest.Status.DefaultChannel
	channelCSV := ""
	for _, channel := range packageManifest.Status.Channels {
		if channel.Name == channelName {
			channelCSV = channel.CurrentCSV
		}
	}

	sr.ChannelName = channelName
	sr.ChannelCSV = channelCSV

	sub := olm.CreateSubscription(suiteCtx, sr)

	timeout := 180 * time.Second
	log.Info("Waiting for strimzi operator to be ready ", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		list, err := suiteCtx.Clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, item := range list.Items {
			if strings.HasPrefix(item.Name, "strimzi-cluster-operator") {
				if item.Status.AvailableReplicas > int32(0) {
					return true, nil
				}
			}
		}
		return false, nil
	})
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())

	return sub, og
}

func removeStrimziOperatorOLM(suiteCtx *types.SuiteContext, namespace string, sub *operatorsv1alpha1.Subscription, og *operatorsv1.OperatorGroup) {

	if sub != nil {
		olm.DeleteSubscription(suiteCtx, sub, false)
	}

	if og != nil {
		var operatorGroupName string = namespace + "-operator-group"
		olm.DeleteOperatorGroup(suiteCtx, namespace, operatorGroupName)
	}

	timeout := 120 * time.Second
	log.Info("Waiting for strimzi operator to be removed ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := suiteCtx.Clientset.AppsV1().Deployments(namespace).Get(context.TODO(), "strimzi-cluster-operator", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())
}

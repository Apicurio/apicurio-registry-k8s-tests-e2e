package streams

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var log = logf.Log.WithName("streams")

var bundlePath string = utils.OperatorBundlePath

var registryName string

//DeployStreamsRegistry deploys a kafka cluster using strimzi operator and deploys an ApicurioRegistry CR using the kafka cluster
func DeployStreamsRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	log.Info("Deploying strimzi operator")

	if strings.HasPrefix(utils.StrimziOperatorBundlePath, "https://") {
		bundlePath = "/tmp/strimzi-operator-bundle-" + strconv.Itoa(rand.Intn(1000)) + ".yaml"
		utils.DownloadFile(bundlePath, utils.StrimziOperatorBundlePath)
		utils.ExecuteCmdOrDie(false, "sed", "-i", "s/namespace: .*/namespace: "+utils.OperatorNamespace+"/", bundlePath)
	} else {
		//TODO implement installing strimzi from local directory
	}

	utils.ExecuteCmdOrDie(true, "kubectl", "apply", "-f", bundlePath, "-n", utils.OperatorNamespace)

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	// sh("oc wait deployment/strimzi-cluster-operator --for condition=available --timeout=180s")
	timeout := 120 * time.Second
	log.Info("Waiting for strimzi operator to be ready ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get("strimzi-cluster-operator", metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		if od != nil {
			if od.Status.AvailableReplicas > int32(0) {
				return true, nil
			}
		}
		return false, nil
	})
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	//install kafka CR and topics
	kafkaClusterName := "registry-kafka"
	utils.ExecuteCmdOrDie(true, "kubectl", "apply", "-f", utils.SuiteProjectDirValue+"/kubefiles/kafka-cluster.yaml", "-n", utils.OperatorNamespace)
	utils.ExecuteCmdOrDie(true, "kubectl", "apply", "-f", utils.SuiteProjectDirValue+"/kubefiles/kafka-topics.yaml", "-n", utils.OperatorNamespace)

	//wait for kafka cluster
	// sh("oc wait deployment/my-cluster-entity-operator --for condition=available --timeout=180s")
	timeout = 120 * time.Second
	log.Info("Waiting for kafka cluster to be ready ", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get(kafkaClusterName+"-entity-operator", metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		if od != nil {
			if od.Status.AvailableReplicas > int32(0) {
				return true, nil
			}
		}
		return false, nil
	})
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "deployment", "-n", utils.OperatorNamespace)
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	svc, err := clientset.CoreV1().Services(utils.OperatorNamespace).Get(kafkaClusterName+"-kafka-bootstrap", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	bootstrapServers := svc.Spec.ClusterIP + ":9092"

	log.Info("Deploying apicurio registry")

	registryName = "apicurio-registry-" + ctx.Storage
	registry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: utils.OperatorNamespace,
			Name:      registryName,
		},
		Spec: apicurio.ApicurioRegistrySpec{
			Deployment: apicurio.ApicurioRegistrySpecDeployment{
				//TODO detect if cluster is kind and do this workaround only in that case
				Host: "localhost",
			},
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				LogLevel:    "DEBUG",
				Persistence: utils.StorageStreams,
				Streams: apicurio.ApicurioRegistrySpecConfigurationStreams{
					ApplicationId:    "registry-application-id",
					BootstrapServers: bootstrapServers,
				},
			},
		},
	}

	err = suiteCtx.K8sClient.Create(context.TODO(), &registry)
	Expect(err).ToNot(HaveOccurred())

	utils.WaitForRegistryReady(suiteCtx.K8sClient, clientset, registryName, ctx.Storage)

}

//RemoveStreamsRegistry uninstalls registry CR, kafka cluster and strimzi operator
func RemoveStreamsRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {
	defer os.Remove(bundlePath)

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	utils.DeleteRegistryAndWait(suiteCtx.K8sClient, clientset, registryName, ctx.Storage)

	log.Info("Removing kafka cluster")

	utils.ExecuteCmdOrDie(true, "kubectl", "delete", "-f", utils.SuiteProjectDirValue+"/kubefiles/kafka-cluster.yaml", "-n", utils.OperatorNamespace)
	utils.ExecuteCmdOrDie(true, "kubectl", "delete", "-f", utils.SuiteProjectDirValue+"/kubefiles/kafka-topics.yaml", "-n", utils.OperatorNamespace)

	utils.ExecuteCmdOrDie(true, "kubectl", "get", "deployment", "-n", utils.OperatorNamespace)
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)

	log.Info("Removing strimzi operator")

	utils.ExecuteCmdOrDie(false, "kubectl", "delete", "-f", bundlePath, "-n", utils.OperatorNamespace)

	timeout := 120 * time.Second
	log.Info("Waiting for strimzi cluster operator to be ready ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get("strimzi-cluster-operator", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "deployment", "-n", utils.OperatorNamespace)
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

}

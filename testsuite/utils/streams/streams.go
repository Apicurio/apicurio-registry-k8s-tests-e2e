package streams

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	apicurioutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var log = logf.Log.WithName("streams")

var bundlePath string = utils.StrimziOperatorBundlePath
var registryKafkaClusterName string = "registry-kafka"
var registryKafkaTopics []string = []string{"storage-topic", "global-id-topic"}

var registryName string

//DeployStreamsRegistry deploys a kafka cluster using strimzi operator and deploys an ApicurioRegistry CR using the kafka cluster
func DeployStreamsRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	var clientset *kubernetes.Clientset = suiteCtx.Clientset

	kafkaClusterInfo := DeployKafkaCluster(clientset, 3, registryKafkaClusterName, registryKafkaTopics)

	bootstrapServers := kafkaClusterInfo.BootstrapServers

	log.Info("Deploying apicurio registry")

	registryName = "apicurio-registry-" + ctx.Storage
	registry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: utils.OperatorNamespace,
			Name:      registryName,
		},
		Spec: apicurio.ApicurioRegistrySpec{
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

	apicurioutils.CreateRegistryAndWait(suiteCtx, ctx, &registry)

}

//RemoveStreamsRegistry uninstalls registry CR, kafka cluster and strimzi operator
func RemoveStreamsRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	defer os.Remove(bundlePath)

	apicurioutils.DeleteRegistryAndWait(suiteCtx, registryName)

	RemoveKafkaCluster(suiteCtx.Clientset, registryKafkaClusterName, registryKafkaTopics)

	RemoveStrimziOperator(suiteCtx.Clientset)

	//TODO wait properly to cleanup

}

//KafkaClusterInfo holds useful info to use a kafka cluster
type KafkaClusterInfo struct {
	StrimziDeployed  bool
	BootstrapServers string
}

//DeployKafkaCluster deploys a kafka cluster and some topics, returns a flag to indicate if strimzi operator has been deployed(useful to know if it was already installed)
func DeployKafkaCluster(clientset *kubernetes.Clientset, replicas int, name string, topics []string) KafkaClusterInfo {

	strimziDeployed := deployStrimziOperator(clientset)

	var replicasStr string = strconv.Itoa(replicas)
	minisr := "1"
	if replicas > 1 {
		minisr = "2"
	}
	kafkaClusterManifestFile := utils.Template("kafka-cluster",
		utils.SuiteProjectDirValue+"/kubefiles/kafka-cluster-template.yaml",
		utils.Replacement{Old: "{NAMESPACE}", New: utils.OperatorNamespace},
		utils.Replacement{Old: "{NAME}", New: name},
		utils.Replacement{Old: "{REPLICAS}", New: replicasStr},
		utils.Replacement{Old: "{MIN_ISR}", New: minisr},
	)
	kafkaClusterManifest := kafkaClusterManifestFile.Name()

	log.Info("Deploying kafka cluster " + name)
	kubernetescli.Execute("apply", "-f", kafkaClusterManifest, "-n", utils.OperatorNamespace)

	for _, topic := range topics {
		kafkaTopicManifestFile := utils.Template("kafka-topic-"+topic,
			utils.SuiteProjectDirValue+"/kubefiles/kafka-topic-template.yaml",
			utils.Replacement{Old: "{NAMESPACE}", New: utils.OperatorNamespace},
			utils.Replacement{Old: "{TOPIC_NAME}", New: topic},
			utils.Replacement{Old: "{CLUSTER_NAME}", New: name},
			utils.Replacement{Old: "{REPLICAS}", New: replicasStr},
			utils.Replacement{Old: "{PARTITIONS}", New: replicasStr},
		)
		kafkaTopicManifest := kafkaTopicManifestFile.Name()

		log.Info("Deploying kafka topic " + topic)
		kubernetescli.Execute("apply", "-f", kafkaTopicManifest, "-n", utils.OperatorNamespace)
	}

	//wait for kafka cluster
	// sh("oc wait deployment/my-cluster-entity-operator --for condition=available --timeout=180s")
	timeout := 4 * time.Minute
	log.Info("Waiting for kafka cluster to be ready ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get(name+"-entity-operator", metav1.GetOptions{})
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
	kubernetescli.GetDeployments(utils.OperatorNamespace)
	kubernetescli.GetPods(utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	svc, err := clientset.CoreV1().Services(utils.OperatorNamespace).Get(name+"-kafka-bootstrap", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	bootstrapServers := svc.Spec.ClusterIP + ":9092"
	return KafkaClusterInfo{StrimziDeployed: strimziDeployed, BootstrapServers: bootstrapServers}
}

func deployStrimziOperator(clientset *kubernetes.Clientset) bool {

	_, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get("strimzi-cluster-operator", metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	} else if err == nil {
		log.Info("Strimzi operator is already deployed")
		return false
	}

	log.Info("Deploying strimzi operator")

	if strings.HasPrefix(utils.StrimziOperatorBundlePath, "https://") {
		bundlePath = "/tmp/strimzi-operator-bundle-" + strconv.Itoa(rand.Intn(1000)) + ".yaml"
		utils.DownloadFile(bundlePath, utils.StrimziOperatorBundlePath)
		utils.ExecuteCmdOrDie(false, "sed", "-i", "s/namespace: .*/namespace: "+utils.OperatorNamespace+"/", bundlePath)
	} else {
		//TODO implement installing strimzi from local directory
	}

	kubernetescli.Execute("apply", "-f", bundlePath, "-n", utils.OperatorNamespace)

	// sh("oc wait deployment/strimzi-cluster-operator --for condition=available --timeout=180s")
	timeout := 120 * time.Second
	log.Info("Waiting for strimzi operator to be ready ", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
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
	kubernetescli.GetPods(utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())
	return true
}

//RemoveKafkaCluster removes a kafka cluster
func RemoveKafkaCluster(clientset *kubernetes.Clientset, name string, topics []string) {

	log.Info("Removing kafka cluster")

	kubernetescli.Execute("delete", "kafka", name, "-n", utils.OperatorNamespace)
	for _, topic := range topics {
		kubernetescli.Execute("delete", "kafkatopic", topic, "-n", utils.OperatorNamespace)
	}

	timeout := 120 * time.Second
	log.Info("Waiting for kafka cluster to be removed ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"strimzi.io/cluster": name})
		l, err := clientset.CoreV1().Pods(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
		// _, err := clientset.AppsV1().StatefulSets(utils.OperatorNamespace).Get(name+"-kafka", metav1.GetOptions{})
		// _, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get(registryKafkaClusterName+"-entity-operator", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return len(l.Items) == 0, nil
	})
	kubernetescli.GetDeployments(utils.OperatorNamespace)
	kubernetescli.GetStatefulSets(utils.OperatorNamespace)
	kubernetescli.GetPods(utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

}

//RemoveStrimziOperator uninstalls strimzi operator
func RemoveStrimziOperator(clientset *kubernetes.Clientset) {
	log.Info("Removing strimzi operator")
	kubernetescli.Execute("delete", "-f", bundlePath, "-n", utils.OperatorNamespace)

	timeout := 120 * time.Second
	log.Info("Waiting for strimzi cluster operator to be removed ", "timeout", timeout)
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
	kubernetescli.GetDeployments(utils.OperatorNamespace)
	kubernetescli.GetPods(utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())
}

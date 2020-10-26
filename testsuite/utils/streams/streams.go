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
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var log = logf.Log.WithName("streams")

var bundlePath string = utils.StrimziOperatorBundlePath
var registryKafkaClusterName string = "registry-kafka"
var registryKafkaTopics []string = []string{"storage-topic", "global-id-topic"}

var registryName string

//DeployStreamsRegistry deploys a kafka cluster using strimzi operator and deploys an ApicurioRegistry CR using the kafka cluster
func DeployStreamsRegistry(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	kafkaClusterInfo := DeployKafkaCluster(suiteCtx, ctx.RegistryNamespace, 3, registryKafkaClusterName, registryKafkaTopics)

	bootstrapServers := kafkaClusterInfo.BootstrapServers

	log.Info("Deploying apicurio registry")

	registryName = "apicurio-registry-" + ctx.Storage
	registry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Name: registryName,
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
func RemoveStreamsRegistry(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	defer os.Remove(bundlePath)

	apicurioutils.DeleteRegistryAndWait(suiteCtx, ctx.RegistryNamespace, registryName)

	RemoveKafkaCluster(suiteCtx.Clientset, ctx.RegistryNamespace, registryKafkaClusterName, registryKafkaTopics)

	RemoveStrimziOperator(suiteCtx.Clientset, ctx.RegistryNamespace)

}

//KafkaClusterInfo holds useful info to use a kafka cluster
type KafkaClusterInfo struct {
	StrimziDeployed          bool
	BootstrapServers         string
	ExternalBootstrapServers string
}

func DeployKafkaCluster(suiteCtx *types.SuiteContext, namespace string, replicas int, name string, topics []string) *KafkaClusterInfo {
	return DeployKafkaClusterV2(suiteCtx, namespace, replicas, false, name, topics)
}

//DeployKafkaCluster deploys a kafka cluster and some topics, returns a flag to indicate if strimzi operator has been deployed(useful to know if it was already installed)
func DeployKafkaClusterV2(suiteCtx *types.SuiteContext, namespace string, replicas int, exposeExternal bool, name string, topics []string) *KafkaClusterInfo {

	strimziDeployed := deployStrimziOperator(suiteCtx.Clientset, namespace)

	clusterInfo := &KafkaClusterInfo{StrimziDeployed: strimziDeployed}

	var replicasStr string = strconv.Itoa(replicas)
	minisr := "1"
	if replicas > 1 {
		minisr = "2"
	}
	var kafkaClusterManifest string = ""
	kindBoostrapHost := "bootstrap.127.0.0.1.nip.io"
	if exposeExternal {
		brokerHost := "broker-0.127.0.0.1.nip.io"
		template := "kafka-cluster-external-template.yaml"
		if suiteCtx.IsOpenshift {
			template = "kafka-cluster-external-ocp-template.yaml"
		}

		kafkaClusterManifestFile := utils.Template("kafka-cluster",
			utils.SuiteProjectDir+"/kubefiles/"+template,
			utils.Replacement{Old: "{NAMESPACE}", New: namespace},
			utils.Replacement{Old: "{NAME}", New: name},
			utils.Replacement{Old: "{BOOTSTRAP_HOST}", New: kindBoostrapHost},
			utils.Replacement{Old: "{BROKER_HOST}", New: brokerHost},
		)
		kafkaClusterManifest = kafkaClusterManifestFile.Name()
	} else {
		kafkaClusterManifestFile := utils.Template("kafka-cluster",
			utils.SuiteProjectDir+"/kubefiles/kafka-cluster-template.yaml",
			utils.Replacement{Old: "{NAMESPACE}", New: namespace},
			utils.Replacement{Old: "{NAME}", New: name},
			utils.Replacement{Old: "{REPLICAS}", New: replicasStr},
			utils.Replacement{Old: "{MIN_ISR}", New: minisr},
		)
		kafkaClusterManifest = kafkaClusterManifestFile.Name()
	}

	log.Info("Deploying kafka cluster " + name)
	kubernetescli.Execute("apply", "-f", kafkaClusterManifest, "-n", namespace)

	for _, topic := range topics {
		kafkaTopicManifestFile := utils.Template("kafka-topic-"+topic,
			utils.SuiteProjectDir+"/kubefiles/kafka-topic-template.yaml",
			utils.Replacement{Old: "{NAMESPACE}", New: namespace},
			utils.Replacement{Old: "{TOPIC_NAME}", New: topic},
			utils.Replacement{Old: "{CLUSTER_NAME}", New: name},
			utils.Replacement{Old: "{REPLICAS}", New: replicasStr},
			utils.Replacement{Old: "{PARTITIONS}", New: replicasStr},
		)
		kafkaTopicManifest := kafkaTopicManifestFile.Name()

		log.Info("Deploying kafka topic " + topic)
		kubernetescli.Execute("apply", "-f", kafkaTopicManifest, "-n", namespace)
	}

	//wait for kafka cluster
	// sh("oc wait deployment/my-cluster-entity-operator --for condition=available --timeout=180s")
	timeout := 4 * time.Minute
	log.Info("Waiting for kafka cluster to be ready ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := suiteCtx.Clientset.AppsV1().Deployments(namespace).Get(name+"-entity-operator", metav1.GetOptions{})
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
	kubernetescli.GetDeployments(namespace)
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())

	if exposeExternal {
		clusterInfo.ExternalBootstrapServers = kindBoostrapHost + ":443"
		if suiteCtx.IsOpenshift {
			route, err := suiteCtx.OcpRouteClient.Routes(namespace).Get(name+"-kafka-bootstrap", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(route.Status.Ingress)).ToNot(BeIdenticalTo(0))
			host := route.Status.Ingress[0].Host
			clusterInfo.ExternalBootstrapServers = host + ":443"
		}
	}

	svc, err := suiteCtx.Clientset.CoreV1().Services(namespace).Get(name+"-kafka-bootstrap", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	bootstrapServers := svc.Spec.ClusterIP + ":9092"
	clusterInfo.BootstrapServers = bootstrapServers
	return clusterInfo
}

func deployStrimziOperator(clientset *kubernetes.Clientset, namespace string) bool {

	_, err := clientset.AppsV1().Deployments(namespace).Get("strimzi-cluster-operator", metav1.GetOptions{})
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
		utils.ExecuteCmdOrDie(false, "sed", "-i", "s/namespace: .*/namespace: "+namespace+"/", bundlePath)
	} else {
		utils.ExecuteCmdOrDie(true, "sed", "-i", "s/namespace: .*/namespace: "+namespace+"/", utils.StrimziOperatorBundlePath+"/*RoleBinding*.yaml")
		bundlePath = utils.StrimziOperatorBundlePath
	}

	kubernetescli.Execute("apply", "-f", bundlePath, "-n", namespace)

	// sh("oc wait deployment/strimzi-cluster-operator --for condition=available --timeout=180s")
	timeout := 120 * time.Second
	log.Info("Waiting for strimzi operator to be ready ", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(namespace).Get("strimzi-cluster-operator", metav1.GetOptions{})
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
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())
	return true
}

//RemoveKafkaCluster removes a kafka cluster
func RemoveKafkaCluster(clientset *kubernetes.Clientset, namespace string, name string, topics []string) {

	log.Info("Removing kafka cluster")

	kubernetescli.Execute("delete", "kafka", name, "-n", namespace)
	for _, topic := range topics {
		kubernetescli.Execute("delete", "kafkatopic", topic, "-n", namespace)
	}

	timeout := 120 * time.Second
	log.Info("Waiting for kafka cluster to be removed ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		labelsSet := labels.Set(map[string]string{"strimzi.io/cluster": name})
		l, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
		// _, err := clientset.AppsV1().StatefulSets(namespace).Get(name+"-kafka", metav1.GetOptions{})
		// _, err := clientset.AppsV1().Deployments(namespace).Get(registryKafkaClusterName+"-entity-operator", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return len(l.Items) == 0, nil
	})
	kubernetescli.GetDeployments(namespace)
	kubernetescli.GetStatefulSets(namespace)
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())

}

//RemoveStrimziOperator uninstalls strimzi operator
func RemoveStrimziOperator(clientset *kubernetes.Clientset, namespace string) {
	log.Info("Removing strimzi operator")
	kubernetescli.Execute("delete", "-f", bundlePath, "-n", namespace)

	timeout := 120 * time.Second
	log.Info("Waiting for strimzi cluster operator to be removed ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := clientset.AppsV1().Deployments(namespace).Get("strimzi-cluster-operator", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	kubernetescli.GetDeployments(namespace)
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())
}

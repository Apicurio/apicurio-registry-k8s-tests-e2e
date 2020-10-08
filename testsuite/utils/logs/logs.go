package logs

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var log = logf.Log.WithName("logs")

func SaveOperatorLogs(clientset *kubernetes.Clientset, suiteID string) {
	//TODO collect first all pods statuses and cluster events

	operatorDeployment, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get(utils.OperatorDeploymentName, metav1.GetOptions{})
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			log.Info("Skipping storing operator logs because operator deployment not found")
			return
		}
		Expect(err).ToNot(HaveOccurred())
	}
	if operatorDeployment.Status.AvailableReplicas == int32(0) {
		log.Info("Skipping storing operator logs because operator deployment is not ready")
		return
	}
	labelsSet := labels.Set(operatorDeployment.Spec.Selector.MatchLabels)
	pods, err := clientset.CoreV1().Pods(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	Expect(err).ToNot(HaveOccurred())

	for _, pod := range pods.Items {
		req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{})
		podLogs, err := req.Stream()
		Expect(err).ToNot(HaveOccurred())
		defer podLogs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		Expect(err).ToNot(HaveOccurred())
		// str := buf.String()

		logsDir := utils.SuiteProjectDir + "/tests-logs/" + suiteID + "/operator/namespaces/" + pod.Namespace + "/"
		os.MkdirAll(logsDir, os.ModePerm)
		logFile := logsDir + pod.Name + ".log"
		log.Info("Storing operator logs", "file", logFile)
		err = ioutil.WriteFile(logFile, buf.Bytes(), os.ModePerm)
		Expect(err).ToNot(HaveOccurred())
	}
}

//SaveTestPodsLogs stores logs of all pods in OperatorNamespace
func SaveTestPodsLogs(clientset *kubernetes.Clientset, suiteID string, testDescription ginkgo.GinkgoTestDescription) {

	testName := ""
	for _, comp := range testDescription.ComponentTexts {
		testName += (comp + "-")
	}
	testName = testName[0 : len(testName)-1]

	log.Info("Collecting test logs", "suite", suiteID, "test", testName)

	pods, err := clientset.CoreV1().Pods(utils.OperatorNamespace).List(metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	logsDir := utils.SuiteProjectDir + "/tests-logs/" + suiteID + "/" + testName + "/namespaces/" + utils.OperatorNamespace + "/"
	os.MkdirAll(logsDir, os.ModePerm)

	//first we collect all pods statuses and cluster events
	currentPodsFile, err := os.Create(logsDir + "pods.log")
	Expect(err).ToNot(HaveOccurred())
	kubernetescli.RedirectOutput(currentPodsFile, os.Stderr, "get", "pods", "-n", utils.OperatorNamespace)
	kubernetescli.RedirectOutput(currentPodsFile, os.Stderr, "get", "pods", "-n", utils.OperatorNamespace, "-o", "yaml")
	defer currentPodsFile.Close()

	eventsFile, err := os.Create(logsDir + "events.log")
	Expect(err).ToNot(HaveOccurred())
	kubernetescli.RedirectOutput(eventsFile, os.Stderr, "get", "events", "-n", utils.OperatorNamespace, "--sort-by=\"{.metadata.creationTimestamp}\"")
	defer eventsFile.Close()

	//then collect logs for each running pod
	for _, pod := range pods.Items {
		if pod.Status.Phase != v1.PodRunning {
			log.Info("Skipping storing pod logs because pod is not ready", "pod", pod.Name)
			continue
		}
		for _, container := range pod.Status.ContainerStatuses {
			saveContainerLogs(clientset, logsDir, container.Name, pod)
		}
	}
}

func saveContainerLogs(clientset *kubernetes.Clientset, logsDir string, container string, pod v1.Pod) {
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{Container: container})
	containerLogs, err := req.Stream()
	Expect(err).ToNot(HaveOccurred())
	defer containerLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, containerLogs)
	Expect(err).ToNot(HaveOccurred())

	logFile := logsDir + pod.Name + "-" + container + ".log"
	log.Info("Storing pod logs", "file", logFile)
	//0644
	err = ioutil.WriteFile(logFile, buf.Bytes(), os.ModePerm)
	Expect(err).ToNot(HaveOccurred())
}

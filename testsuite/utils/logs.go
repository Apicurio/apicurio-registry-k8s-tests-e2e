package utils

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	v1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/gomega"
)

func SaveOperatorLogs(clientset *kubernetes.Clientset, suiteID string) {
	//TODO collect first all pods statuses and cluster events

	operatorDeployment, err := clientset.AppsV1().Deployments(OperatorNamespace).Get(OperatorDeploymentName, metav1.GetOptions{})
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
	pods, err := clientset.CoreV1().Pods(OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
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

		logsDir := SuiteProjectDirValue + "/tests-logs/" + suiteID + "/namespaces/" + pod.Namespace + "/"
		os.MkdirAll(logsDir, os.ModePerm)
		logFile := logsDir + pod.Name + ".log"
		log.Info("Storing operator logs", "file", logFile)
		err = ioutil.WriteFile(logFile, buf.Bytes(), os.ModePerm)
		Expect(err).ToNot(HaveOccurred())
	}
}

func SaveTestPodsLogs(clientset *kubernetes.Clientset, suiteID string, testName string) {
	log.Info("Collecting test logs", "suite", suiteID, "test", testName)

	pods, err := clientset.CoreV1().Pods(OperatorNamespace).List(metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	//TODO collect first all pods statuses and cluster events

	logsDir := SuiteProjectDirValue + "/tests-logs/" + suiteID + "/" + testName + "/namespaces/" + OperatorNamespace + "/"
	os.MkdirAll(logsDir, os.ModePerm)

	for _, pod := range pods.Items {
		if pod.Status.Phase != v1.PodRunning {
			log.Info("Skipping storing pod logs because pod is not ready", "pod", pod.Name)
			continue
		}
		req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{})
		podLogs, err := req.Stream()
		Expect(err).ToNot(HaveOccurred())
		defer podLogs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		Expect(err).ToNot(HaveOccurred())
		// str := buf.String()

		logFile := logsDir + pod.Name + ".log"
		log.Info("Storing pod logs", "file", logFile)
		//0644
		err = ioutil.WriteFile(logFile, buf.Bytes(), os.ModePerm)
		Expect(err).ToNot(HaveOccurred())
	}
}

package kubernetescli

import (
	"os"
	"sync"

	. "github.com/onsi/gomega"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
)

type CLIKubernetesClient string

var (
	Kubectl CLIKubernetesClient = "kubectl"
	Oc      CLIKubernetesClient = "oc"
)

type KubernetesClient struct {
	cmd CLIKubernetesClient
}

var lock = &sync.Mutex{}

var instance *KubernetesClient

func NewCLIKubernetesClient(cmd CLIKubernetesClient) *KubernetesClient {

	lock.Lock()
	defer lock.Unlock()

	if instance == nil {
		instance = &KubernetesClient{
			cmd: cmd,
		}
	}

	return instance
}

func GetCLIKubernetesClient() *KubernetesClient {
	Expect(instance).ToNot(BeNil())
	return instance
}

func GetDeployments(namespace string) {
	Execute("get", "deployment", "-n", namespace)
}

func GetStatefulSets(namespace string) {
	Execute("get", "statefulset", "-n", namespace)
}

func GetPods(namespace string) {
	Execute("get", "pod", "-n", namespace)
}

func GetVolumes(namespace string) {
	Execute("get", "pvc", "-n", namespace)
	Execute("get", "pv")
}

func Execute(args ...string) {
	utils.ExecuteCmdOrDie(true, string(GetCLIKubernetesClient().cmd), args...)
}

func RedirectOutput(stdOutFile *os.File, stdErrFile *os.File, args ...string) {
	err := utils.Execute(&utils.Command{Cmd: append([]string{string(GetCLIKubernetesClient().cmd)}, args...)}, stdOutFile, stdErrFile)
	Expect(err).ToNot(HaveOccurred())
}

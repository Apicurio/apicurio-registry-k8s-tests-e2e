package bundle

import (
	"math/rand"
	"os"
	"strconv"
	"strings"

	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	testcase "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/testcase"
)

var _ = Describe("bundle installation", func() {

	testcase.CommonTestCases(suiteCtx)

})

var bundlePath string = utils.OperatorBundlePath

func installOperator() {

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	utils.CreateTestNamespace(clientset, utils.OperatorNamespace)

	if strings.HasPrefix(utils.OperatorBundlePath, "https://") {
		bundlePath = "/tmp/apicurio-operator-bundle-" + strconv.Itoa(rand.Intn(1000)) + ".yaml"
		utils.DownloadFile(bundlePath, utils.OperatorBundlePath)
	}

	file := utils.Template("operator-bundle", bundlePath,
		utils.Replacement{Old: "{NAMESPACE}", New: utils.OperatorNamespace},
	)

	bundlePath = file.Name()

	log.Info("Installing operator")
	utils.ExecuteCmdOrDie(true, "kubectl", "apply", "-f", bundlePath, "-n", utils.OperatorNamespace)

	utils.WaitForOperatorDeploymentReady(clientset)

}

func uninstallOperator() {

	defer os.Remove(bundlePath)

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	utils.SaveOperatorLogs(clientset, suiteCtx.SuiteID)

	log.Info("Uninstalling operator")
	utils.ExecuteCmdOrDie(false, "kubectl", "delete", "-f", bundlePath, "-n", utils.OperatorNamespace)

	utils.WaitForOperatorDeploymentRemoved(clientset)

	utils.DeleteTestNamespace(clientset, utils.OperatorNamespace)

}

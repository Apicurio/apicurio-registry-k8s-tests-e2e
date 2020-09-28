package bundle

import (
	"math/rand"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	testcase "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/testcase"
)

var _ = Describe("bundle installation", func() {

	testcase.CommonTestCases(suiteCtx)

})

var bundlePath string = utils.OperatorBundlePath

func installOperator() {

	kubernetesutils.CreateTestNamespace(suiteCtx.Clientset, utils.OperatorNamespace)

	if strings.HasPrefix(utils.OperatorBundlePath, "https://") {
		bundlePath = "/tmp/apicurio-operator-bundle-" + strconv.Itoa(rand.Intn(1000)) + ".yaml"
		utils.DownloadFile(bundlePath, utils.OperatorBundlePath)
	}

	file := utils.Template("operator-bundle", bundlePath,
		utils.Replacement{Old: "{NAMESPACE}", New: utils.OperatorNamespace},
	)

	bundlePath = file.Name()

	log.Info("Installing operator")
	kubernetescli.Execute("apply", "-f", bundlePath, "-n", utils.OperatorNamespace)

	kubernetesutils.WaitForOperatorDeploymentReady(suiteCtx.Clientset)

}

func uninstallOperator() {

	if strings.HasPrefix(bundlePath, "/tmp/") {
		defer os.Remove(bundlePath)
	}

	utils.SaveOperatorLogs(suiteCtx.Clientset, suiteCtx.SuiteID)

	log.Info("Uninstalling operator")
	kubernetescli.Execute("delete", "-f", bundlePath, "-n", utils.OperatorNamespace)

	kubernetesutils.WaitForOperatorDeploymentRemoved(suiteCtx.Clientset)

	kubernetesutils.DeleteTestNamespace(suiteCtx.Clientset, utils.OperatorNamespace)

}

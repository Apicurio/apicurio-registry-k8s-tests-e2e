package bundle

import (
	"math/rand"
	"os"
	"path/filepath"
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

	log.Info("Installing operator")
	if strings.HasPrefix(utils.OperatorBundlePath, "https://") {
		bundlePath = "/tmp/apicurio-operator-bundle-" + strconv.Itoa(rand.Intn(1000)) + ".yaml"
		utils.DownloadFile(bundlePath, utils.OperatorBundlePath)

		kubernetescli.Execute("apply", "-n", utils.OperatorNamespace, "-f", bundlePath)
	} else if strings.HasSuffix(utils.OperatorBundlePath, ".yaml") {
		file := utils.Template("operator-bundle", bundlePath,
			utils.Replacement{Old: "{NAMESPACE}", New: utils.OperatorNamespace},
		)
		bundlePath = file.Name()

		kubernetescli.Execute("apply", "-n", utils.OperatorNamespace, "-f", bundlePath)
	} else {

		kubernetescli.Execute("apply", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "service_account.yaml"))
		//maybe set pull secret
		kubernetescli.Execute("apply", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "role.yaml"))
		kubernetescli.Execute("apply", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "role_binding.yaml"))
		kubernetescli.Execute("apply", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "cluster_role.yaml"))
		clusterRoleBindingFile := utils.Template("cluster_role_binding", filepath.Join(bundlePath, "cluster_role_binding.yaml"),
			utils.Replacement{Old: "{NAMESPACE}", New: utils.OperatorNamespace},
		)
		defer os.Remove(clusterRoleBindingFile.Name())
		kubernetescli.Execute("apply", "-n", utils.OperatorNamespace, "-f", clusterRoleBindingFile.Name())
		kubernetescli.Execute("apply", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "crds/apicur.io_apicurioregistries_crd.yaml"))
		kubernetescli.Execute("apply", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "operator.yaml"))
	}

	kubernetesutils.WaitForOperatorDeploymentReady(suiteCtx.Clientset)

}

func uninstallOperator() {

	if strings.HasPrefix(bundlePath, "/tmp/") {
		defer os.Remove(bundlePath)
	}

	utils.SaveOperatorLogs(suiteCtx.Clientset, suiteCtx.SuiteID)

	log.Info("Uninstalling operator")

	if strings.HasSuffix(bundlePath, ".yaml") {
		kubernetescli.Execute("delete", "-n", utils.OperatorNamespace, "-f", bundlePath)
	} else {
		kubernetescli.Execute("delete", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "operator.yaml"))
		kubernetescli.Execute("delete", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "crds/apicur.io_apicurioregistries_crd.yaml"))
		kubernetescli.Execute("delete", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "cluster_role_binding.yaml"))
		kubernetescli.Execute("delete", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "cluster_role.yaml"))
		kubernetescli.Execute("delete", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "role_binding.yaml"))
		kubernetescli.Execute("delete", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "role.yaml"))
		kubernetescli.Execute("delete", "-n", utils.OperatorNamespace, "-f", filepath.Join(bundlePath, "service_account.yaml"))
	}

	kubernetesutils.WaitForOperatorDeploymentRemoved(suiteCtx.Clientset)

	kubernetesutils.DeleteTestNamespace(suiteCtx.Clientset, utils.OperatorNamespace)

}

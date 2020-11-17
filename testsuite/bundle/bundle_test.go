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
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/logs"
	testcase "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/testcase"
)

var _ = Describe("bundle installation", func() {

	testcase.CommonTestCases(suiteCtx)
	testcase.BundleOnlyTestCases(suiteCtx)

})

var bundlePath string = utils.OperatorBundlePath
var operatorNamespace string = utils.OperatorNamespace

func installOperator() {

	kubernetesutils.CreateTestNamespace(suiteCtx.Clientset, operatorNamespace)

	log.Info("Installing operator")
	if strings.HasPrefix(utils.OperatorBundlePath, "https://") {
		bundlePath = "/tmp/apicurio-operator-bundle-" + strconv.Itoa(rand.Intn(1000)) + ".yaml"
		utils.DownloadFile(bundlePath, utils.OperatorBundlePath)

		kubernetescli.Execute("apply", "-n", operatorNamespace, "-f", bundlePath)
	} else if strings.HasSuffix(utils.OperatorBundlePath, ".yaml") {
		file := utils.Template("operator-bundle", bundlePath,
			utils.Replacement{Old: "{NAMESPACE}", New: operatorNamespace},
		)
		bundlePath = file.Name()

		kubernetescli.Execute("apply", "-n", operatorNamespace, "-f", bundlePath)
	} else {

		kubernetescli.Execute("apply", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "service_account.yaml"))
		if utils.ImagePullSecretUser != "" {
			kubernetesutils.SetPullSecret(suiteCtx.Clientset, "apicurio-registry-operator", operatorNamespace)
		}
		kubernetescli.Execute("apply", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "role.yaml"))
		kubernetescli.Execute("apply", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "role_binding.yaml"))
		kubernetescli.Execute("apply", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "cluster_role.yaml"))
		clusterRoleBindingFile := utils.Template("cluster_role_binding", filepath.Join(bundlePath, "cluster_role_binding.yaml"),
			utils.Replacement{Old: "{NAMESPACE}", New: operatorNamespace},
		)
		defer os.Remove(clusterRoleBindingFile.Name())
		kubernetescli.Execute("apply", "-n", operatorNamespace, "-f", clusterRoleBindingFile.Name())
		kubernetescli.Execute("apply", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "crds/apicur.io_apicurioregistries_crd.yaml"))
		kubernetescli.Execute("apply", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "operator.yaml"))
	}

	kubernetesutils.WaitForOperatorDeploymentReady(suiteCtx.Clientset, operatorNamespace)

}

func uninstallOperator() {

	if strings.HasPrefix(bundlePath, "/tmp/") {
		defer os.Remove(bundlePath)
	}

	logs.SaveOperatorLogs(suiteCtx.Clientset, suiteCtx.SuiteID, operatorNamespace)

	log.Info("Uninstalling operator")

	if strings.HasSuffix(bundlePath, ".yaml") {
		kubernetescli.Execute("delete", "-n", operatorNamespace, "-f", bundlePath)
	} else {
		kubernetescli.Execute("delete", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "operator.yaml"))
		kubernetescli.Execute("delete", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "crds/apicur.io_apicurioregistries_crd.yaml"))
		kubernetescli.Execute("delete", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "cluster_role_binding.yaml"))
		kubernetescli.Execute("delete", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "cluster_role.yaml"))
		kubernetescli.Execute("delete", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "role_binding.yaml"))
		kubernetescli.Execute("delete", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "role.yaml"))
		kubernetescli.Execute("delete", "-n", operatorNamespace, "-f", filepath.Join(bundlePath, "service_account.yaml"))
	}

	kubernetesutils.WaitForOperatorDeploymentRemoved(suiteCtx.Clientset, operatorNamespace)

	kubernetesutils.DeleteTestNamespace(suiteCtx.Clientset, operatorNamespace)

}

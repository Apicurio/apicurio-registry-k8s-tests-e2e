package functional

import (
	"net/http"
	"os"
	"time"

	. "github.com/onsi/gomega"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/util/wait"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	types "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("functional")

//ExecuteRegistryFunctionalTests invokes via maven the integration tests in apicurio-registry repo
func ExecuteRegistryFunctionalTests(ctx *types.TestContext) {
	testProfile := "smoke"

	oldDir, err := os.Getwd()
	apicurioProjectDir := utils.SuiteProjectDirValue + "/apicurio-registry"
	if utils.ApicurioProjectDir != "" {
		apicurioProjectDir = utils.ApicurioProjectDir
	}
	os.Chdir(apicurioProjectDir)

	var command = []string{"mvn", "verify", "-P" + testProfile, "-P" + ctx.Storage, "-pl", "tests", "-am", "-Dmaven.javadoc.skip=true", "-Dstyle.color=always", "--no-transfer-progress", "-DtrimStackTrace=false"}
	if utils.ExtraMavenArgs != "" {
		command = append(command, utils.ExtraMavenArgs)
	}

	var env = []string{
		"EXTERNAL_REGISTRY=true",
		"TEST_REGISTRY_CLIENT=create",
		"REGISTRY_HOST=" + ctx.RegistryHost,
		"REGISTRY_PORT=" + ctx.RegistryPort,
	}

	err = utils.ExecuteCmd(true, &utils.Command{Cmd: command, Env: env})
	os.Chdir(oldDir)
	Expect(err).NotTo(HaveOccurred())
}

//BasicRegistryAPITest simple test against apicurio registry api to just verify it's up and running
func BasicRegistryAPITest(ctx *types.TestContext) {

	log.Info("Testing registry API")
	timeout := 30 * time.Second
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		res, err := http.Get("http://" + ctx.RegistryHost + ":" + ctx.RegistryPort + "/api/artifacts")
		if err != nil {
			return false, err
		}
		if res.StatusCode != 200 {
			return false, nil
		}
		log.Info("Status code is " + res.Status)
		return true, nil
	})
	if err != nil {
		log.Info("Registry API verification failed with error")
	}
	Expect(err).NotTo(HaveOccurred())
	log.Info("Successful registry API verification")

}

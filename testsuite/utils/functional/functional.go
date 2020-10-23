package functional

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/util/wait"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	types "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("functional")

//ExecuteRegistryFunctionalTests invokes via maven the integration tests in apicurio-registry repo
func ExecuteRegistryFunctionalTests(suiteCtx *types.SuiteContext, ctx *types.TestContext) {
	testProfile := "smoke"
	if utils.ApicurioTestsProfile != "" {
		testProfile = utils.ApicurioTestsProfile
	}

	oldDir, err := os.Getwd()
	apicurioProjectDir := utils.SuiteProjectDir + "/apicurio-registry"
	if utils.ApicurioProjectDir != "" {
		apicurioProjectDir = utils.ApicurioProjectDir
	}
	log.Info("Apicurio Registry Tests", "directory", apicurioProjectDir)
	os.Chdir(apicurioProjectDir)

	var command = []string{"mvn", "verify", "-P" + testProfile, "-P" + ctx.Storage, "-pl", "tests", "-am", "-Dmaven.javadoc.skip=true", "-Dstyle.color=always", "--no-transfer-progress", "-DtrimStackTrace=false"}
	if utils.ExtraMavenArgs != "" {
		for _, arg := range strings.Split(utils.ExtraMavenArgs, " ") {
			command = append(command, arg)
		}
	}

	var env = []string{
		"EXTERNAL_REGISTRY=true",
		"TEST_REGISTRY_CLIENT=create",
		"REGISTRY_HOST=" + ctx.RegistryHost,
		"REGISTRY_PORT=" + ctx.RegistryPort,
		"SELENIUM_HOST=" + suiteCtx.SeleniumHost,
		"SELENIUM_PORT=" + suiteCtx.SeleniumPort,
		"REGISTRY_SELENIUM_HOST=" + ctx.RegistryInternalHost,
		"REGISTRY_SELENIUM_PORT=" + ctx.RegistryInternalPort,
	}

	err = utils.ExecuteCmd(true, &utils.Command{Cmd: command, Env: env})
	os.Chdir(oldDir)
	if err != nil {
		Expect(errors.New("There are test failures")).NotTo(HaveOccurred())
	}
}

//BasicRegistryAPITest simple test against apicurio registry api to just verify it's up and running
func BasicRegistryAPITest(ctx *types.TestContext) {

	log.Info("Testing registry API")
	timeout := 60 * time.Second
	statusCode := ""
	body := ""
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		res, err := http.Get("http://" + ctx.RegistryHost + ":" + ctx.RegistryPort + "/api/artifacts")
		if err != nil {
			return false, err
		}
		statusCode = res.Status
		body = utils.ReaderToString(res.Body)
		if res.StatusCode != 200 {
			return false, nil
		}
		log.Info("Status code is " + res.Status)
		return true, nil
	})
	if err != nil {
		log.Info("Registry API verification failed with error")
		log.Info("Status " + statusCode)
		log.Info("Response " + body)
	}
	Expect(err).NotTo(HaveOccurred())
	log.Info("Successful registry API verification")

}

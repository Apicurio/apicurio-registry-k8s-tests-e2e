package functional

import (
	"os"

	. "github.com/onsi/gomega"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	types "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

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

package functional

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	if ctx.FunctionalTestsProfile != "" {
		testProfile = ctx.FunctionalTestsProfile
	} else if utils.ApicurioTestsProfile != "" {
		testProfile = utils.ApicurioTestsProfile
	}

	oldDir, err := os.Getwd()
	apicurioProjectDir := utils.SuiteProjectDir + "/apicurio-registry"
	if utils.ApicurioProjectDir != "" {
		apicurioProjectDir = utils.ApicurioProjectDir
	}
	log.Info("Apicurio Registry Tests", "directory", apicurioProjectDir)
	os.Chdir(apicurioProjectDir)

	// "--no-transfer-progress"
	var command = []string{"mvn", "verify", "-P" + testProfile, "-P" + ctx.Storage, "-Pintegration-tests", "-pl", "integration-tests/testsuite", "-am", "-Dmaven.javadoc.skip=true", "-Dstyle.color=always", "-DtrimStackTrace=false", "--no-transfer-progress"}
	if utils.ExtraMavenArgs != "" {
		for _, arg := range strings.Split(utils.ExtraMavenArgs, " ") {
			command = append(command, arg)
		}
	}

	var env = []string{
		"EXTERNAL_REGISTRY=true",
	}

	if ctx.RegistryHost != "" {
		registryEnvs := []string{
			"REGISTRY_HOST=" + ctx.RegistryHost,
			"REGISTRY_PORT=" + ctx.RegistryPort,
			"SELENIUM_HOST=" + suiteCtx.SeleniumHost,
			"SELENIUM_PORT=" + suiteCtx.SeleniumPort,
			"REGISTRY_SELENIUM_HOST=" + ctx.RegistryInternalHost,
			"REGISTRY_SELENIUM_PORT=" + ctx.RegistryInternalPort,
		}
		env = append(env, registryEnvs...)
	}

	if ctx.FunctionalTestsExtraEnv != nil {
		env = append(env, ctx.FunctionalTestsExtraEnv...)
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

func BasicRegistryAPITestWithAuthentication(ctx *types.TestContext, user string, pwd string) {

	accessToken := issueAccessToken(ctx, user, pwd)

	log.Info("Testing secured registry API")
	timeout := 60 * time.Second
	statusCode := ""
	body := ""
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {

		req, err := http.NewRequest("GET", "http://"+ctx.RegistryHost+":"+ctx.RegistryPort+"/api/artifacts", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Add("Authorization", "Bearer "+accessToken)
		res, err := http.DefaultClient.Do(req)

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

	verifyUnauthorized(ctx)

}

func issueAccessToken(ctx *types.TestContext, user string, pwd string) string {
	keycloakUrl := ctx.RegistryResource.Spec.Configuration.Security.Keycloak.Url
	realm := ctx.RegistryResource.Spec.Configuration.Security.Keycloak.Realm
	realmUrl := keycloakUrl + "/realms/" + realm + "/protocol/openid-connect/token"
	clientId := ctx.RegistryResource.Spec.Configuration.Security.Keycloak.ApiClientId

	values := url.Values{}
	values.Set("grant_type", "password")
	values.Set("client_id", clientId)
	values.Set("username", user)
	values.Set("password", pwd)

	log.Info("Requesting access token")

	res, err := http.PostForm(realmUrl, values)
	Expect(err).NotTo(HaveOccurred())
	if res.StatusCode > 299 {
		b := utils.ReaderToString(res.Body)
		Expect(errors.New("Keycloak request status code is " + strconv.Itoa(res.StatusCode) + " body is " + b)).NotTo(HaveOccurred())
	}

	jsonMap := make(map[string]interface{})
	err = json.Unmarshal(utils.ReaderToBytes(res.Body), &jsonMap)
	Expect(err).NotTo(HaveOccurred())

	return jsonMap["access_token"].(string)
}

func verifyUnauthorized(ctx *types.TestContext) {
	log.Info("Testing secured registry API rejects unauthorized access")
	timeout := 20 * time.Second
	statusCode := ""
	body := ""
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {

		req, err := http.NewRequest("GET", "http://"+ctx.RegistryHost+":"+ctx.RegistryPort+"/api/artifacts", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Add("Authorization", "Bearer foo")
		res, err := http.DefaultClient.Do(req)

		if err != nil {
			return false, err
		}
		statusCode = res.Status
		body = utils.ReaderToString(res.Body)
		if res.StatusCode != 401 {
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
}

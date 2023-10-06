package suite

import (
	"flag"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	ocp_route_client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	kubernetescli "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/selenium"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apicurioScheme "github.com/Apicurio/apicurio-registry-operator/api/v1"
	olmapiversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	pmversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/client/clientset/versioned"
)

var log = logf.Log.WithName("suite")

var onlyTestOperator bool
var setupSelenium bool
var disableClusteredTests bool
var disableConvertersTests bool
var disableAuthTests bool
var olmRunAdvancedTestcases bool
var installStrimziOLM bool

//SetFlags call this function on init function on test suite package
func SetFlags() {
	flag.BoolVar(&onlyTestOperator, "only-test-operator", false, "to only test operator installation and registry installation")
	flag.BoolVar(&setupSelenium, "setup-selenium", false, "to deploy selenium, used for ui testing, if this flag is not passed testsuite will deploy selenium anyway if it detects it's required")
	flag.BoolVar(&disableClusteredTests, "disable-clustered-tests", false, "to disable tests for clustered registry deployments")
	flag.BoolVar(&disableConvertersTests, "disable-converters-tests", false, "to disable tests for kafka connect converters")
	flag.BoolVar(&disableAuthTests, "disable-auth-tests", false, "to disable tests for keycloak authentication")
	flag.BoolVar(&olmRunAdvancedTestcases, "enable-olm-advanced-tests", false, "to enable advanced tests for OLM testsuite")
	flag.BoolVar(&installStrimziOLM, "install-strimzi-olm", false, "to enable the installation of Strimzi operator using OLM")
}

//NewSuiteContext creates the SuiteContext instance and loads some data like flags into the context
func NewSuiteContext(suiteID string) *types.SuiteContext {
	var suiteCtx types.SuiteContext = types.SuiteContext{}
	suiteCtx.SuiteID = suiteID

	suiteCtx.OnlyTestOperator = onlyTestOperator
	if suiteCtx.OnlyTestOperator {
		log.Info("Only testing operator functionality")
	}

	suiteCtx.DisableClusteredTests = disableClusteredTests
	if suiteCtx.DisableClusteredTests {
		log.Info("Clustered registry tests disabled")
	}

	suiteCtx.DisableConvertersTests = disableConvertersTests
	if suiteCtx.DisableConvertersTests {
		log.Info("Converters tests disabled")
	}

	suiteCtx.DisableAuthTests = disableAuthTests
	if suiteCtx.DisableAuthTests {
		log.Info("Keycloak Authentication tests disabled")
	}

	suiteCtx.OLMRunAdvancedTestcases = olmRunAdvancedTestcases
	if suiteCtx.OLMRunAdvancedTestcases {
		log.Info("Running Advanced Testcases with OLM deployment")
	}

	suiteCtx.InstallStrimziOLM = installStrimziOLM
	if suiteCtx.InstallStrimziOLM {
		log.Info("Strimzi operator will be installed using OLM")
	}

	suiteCtx.SetupSelenium = setupSelenium

	return &suiteCtx
}

//InitSuite performs common logic for Ginkgo's BeforeSuite
func InitSuite(suiteCtx *types.SuiteContext) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")

	useCluster := true
	suiteCtx.TestEnv = &envtest.Environment{
		UseExistingCluster:       &useCluster,
		AttachControlPlaneOutput: false,
	}

	var err error
	suiteCtx.Cfg, err = suiteCtx.TestEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(suiteCtx.Cfg).ToNot(BeNil())

	//

	err = apicurioScheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	suiteCtx.PackageClient = pmversioned.NewForConfigOrDie(suiteCtx.Cfg)

	suiteCtx.OLMClient = olmapiversioned.NewForConfigOrDie(suiteCtx.Cfg)

	suiteCtx.K8sManager, err = ctrl.NewManager(suiteCtx.Cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = suiteCtx.K8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	suiteCtx.K8sClient = suiteCtx.K8sManager.GetClient()
	Expect(suiteCtx.K8sClient).ToNot(BeNil())

	suiteCtx.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(suiteCtx.Clientset).ToNot(BeNil())

	isocp, err := kubernetesutils.IsOCP(suiteCtx.Cfg)
	Expect(err).ToNot(HaveOccurred())
	suiteCtx.IsOpenshift = isocp

	if suiteCtx.IsOpenshift {
		log.Info("Openshift cluster detected")
		suiteCtx.OcpRouteClient = ocp_route_client.NewForConfigOrDie(suiteCtx.Cfg)
	}

	cmd := kubernetescli.Kubectl
	if suiteCtx.IsOpenshift {
		cmd = kubernetescli.Oc
	}
	suiteCtx.CLIKubernetesClient = kubernetescli.NewCLIKubernetesClient(cmd)
	Expect(suiteCtx.CLIKubernetesClient).ToNot(BeNil())

	selenium.DeploySeleniumIfNeeded(suiteCtx)

}

func PreTearDown(suiteCtx *types.SuiteContext) {
	selenium.CollectSeleniumLogsIfNeeded(suiteCtx)
}

//TearDownSuite performs common logic for Ginkgo's AfterSuite
func TearDownSuite(suiteCtx *types.SuiteContext) {
	By("tearing down the test environment")

	selenium.RemoveSeleniumIfNeeded(suiteCtx)

	err := suiteCtx.TestEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
}

//RunSuite starts the execution of a test suite
func RunSuite(t *testing.T, suiteName string, suiteCtx *types.SuiteContext) {

	if utils.SuiteProjectDir == "" {
		panic("Env var utils.suiteProjectDirEnvVar is required")
	}

	RegisterFailHandler(Fail)

	RunSpecs(t, suiteName)
}

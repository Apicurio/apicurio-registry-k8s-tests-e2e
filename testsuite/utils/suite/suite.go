package suite

import (
	"flag"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"

	ocp_apps_client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	ocp_route_client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	kubernetescli "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/selenium"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	customreporters "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite/reporters"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apicurioScheme "github.com/Apicurio/apicurio-registry-operator/pkg/apis"
	olmapiversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	pmversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/client/clientset/versioned"
)

var log = logf.Log.WithName("suite")

var onlyTestOperator bool
var setupSelenium bool
var disableClusteredTests bool

//SetFlags call this function on init function on test suite package
func SetFlags() {
	flag.BoolVar(&onlyTestOperator, "only-test-operator", false, "to only test operator installation and registry installation")
	flag.BoolVar(&setupSelenium, "setup-selenium", false, "to deploy selenium, used for ui testing, if this flag is not passed testsuite will deploy selenium anyway if it detects it's required")
	flag.BoolVar(&disableClusteredTests, "disable-clustered-tests", false, "to disable tests for clustered registry deployments")
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

	suiteCtx.SetupSelenium = setupSelenium

	return &suiteCtx
}

//InitSuite performs common logic for Ginkgo's BeforeSuite
func InitSuite(suiteCtx *types.SuiteContext) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

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
		suiteCtx.OcpAppsClient = ocp_apps_client.NewForConfigOrDie(suiteCtx.Cfg)
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

	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf(utils.SuiteProjectDir+"/tests-logs/"+suiteCtx.SuiteID+"/TEST-ginkgo-junit_%s.xml", time.Now().Format("20060102150405")))

	r := []Reporter{printer.NewlineReporter{}, junitReporter}

	if utils.SummaryFile != "" {
		r = append(r, customreporters.NewTextSummaryReporter(utils.SummaryFile))
	}

	RunSpecsWithDefaultAndCustomReporters(t, suiteName, r)
}

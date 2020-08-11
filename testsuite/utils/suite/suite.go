package suite

import (
	"flag"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apicurioScheme "github.com/Apicurio/apicurio-registry-operator/pkg/apis"
	olmapiversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	pmversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/client/clientset/versioned"
)

var log = logf.Log.WithName("suite")

//SuiteContext holds common info used in a testsuite
type SuiteContext struct {
	SuiteID       string
	Cfg           *rest.Config
	K8sClient     client.Client
	k8sManager    ctrl.Manager
	testEnv       *envtest.Environment
	PackageClient pmversioned.Interface
	OLMClient     olmapiversioned.Interface

	OnlyTestOperator bool
}

var onlyTestOperator bool

//SetFlags call this function on init function on test suite package
func SetFlags() {
	flag.BoolVar(&onlyTestOperator, "only-test-operator", false, "to only test operator installation and registry installation")
}

//InitSuite performs common logic for Ginkgo's BeforeSuite
func InitSuite(suiteID string) *SuiteContext {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")

	var suiteCtx SuiteContext = SuiteContext{}
	suiteCtx.SuiteID = suiteID

	useCluster := true
	suiteCtx.testEnv = &envtest.Environment{
		UseExistingCluster:       &useCluster,
		AttachControlPlaneOutput: false,
	}

	var err error
	suiteCtx.Cfg, err = suiteCtx.testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(suiteCtx.Cfg).ToNot(BeNil())

	//

	err = apicurioScheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	suiteCtx.PackageClient = pmversioned.NewForConfigOrDie(suiteCtx.Cfg)

	suiteCtx.OLMClient = olmapiversioned.NewForConfigOrDie(suiteCtx.Cfg)

	suiteCtx.OnlyTestOperator = onlyTestOperator
	if suiteCtx.OnlyTestOperator {
		log.Info("Only testing operator functionality")
	}

	//

	suiteCtx.k8sManager, err = ctrl.NewManager(suiteCtx.Cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = suiteCtx.k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	suiteCtx.K8sClient = suiteCtx.k8sManager.GetClient()
	Expect(suiteCtx.K8sClient).ToNot(BeNil())

	return &suiteCtx
}

//TearDownSuite performs common logic for Ginkgo's AfterSuite
func TearDownSuite(suiteCtx *SuiteContext) {
	By("tearing down the test environment")

	err := suiteCtx.testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
}

//RunSuite starts the execution of a test suite
func RunSuite(t *testing.T, suiteName string, suiteID string) {

	if utils.SuiteProjectDirValue == "" {
		panic("Env var " + utils.SuiteProjectDirEnvVar + " is required")
	}

	if utils.OLMCatalogSourceImage == "" {
		panic("Env var " + utils.OLMCatalogSourceImageEnvVar + " is required")
	}

	RegisterFailHandler(Fail)

	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf(utils.SuiteProjectDirValue+"/tests-logs/"+suiteID+"/TEST-ginkgo-junit_%s.xml", time.Now().String()))

	RunSpecsWithDefaultAndCustomReporters(t, suiteName,
		[]Reporter{printer.NewlineReporter{}, junitReporter},
	)
}

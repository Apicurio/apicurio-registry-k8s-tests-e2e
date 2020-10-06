package suite

import (
	"flag"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"

	ocp_apps_client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	ocp_route_client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	utils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	kubernetescli "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"

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
	Clientset     *kubernetes.Clientset
	IsOpenshift   bool

	OcpAppsClient  *ocp_apps_client.AppsV1Client
	OcpRouteClient *ocp_route_client.RouteV1Client

	CLIKubernetesClient *kubernetescli.KubernetesClient

	OnlyTestOperator bool
}

type OcpImageReference struct {
	ExternalImage string
	InternalImage string
}

func (ctx *SuiteContext) OcpInternalImage(namespace string, imageName string, tag string) *OcpImageReference {
	// oc get route -n openshift-image-registry
	ocpImageRegistryRoute, err := ctx.OcpRouteClient.Routes("openshift-image-registry").Get("default-route", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	ocpImageRegistryHost := ocpImageRegistryRoute.Status.Ingress[0].Host

	return &OcpImageReference{
		ExternalImage: ocpImageRegistryHost + "/" + namespace + "/" + imageName + ":" + tag,
		InternalImage: "image-registry.openshift-image-registry.svc:5000" + "/" + namespace + "/" + imageName + ":" + tag,
	}
	// return ocpImageRegistryHost + "/" + namespace + "/" + imageName + ":" + tag
}

var onlyTestOperator bool

//SetFlags call this function on init function on test suite package
func SetFlags() {
	flag.BoolVar(&onlyTestOperator, "only-test-operator", false, "to only test operator installation and registry installation")
}

//NewSuiteContext creates the SuiteContext instance and loads some data like flags into the context
func NewSuiteContext(suiteID string) *SuiteContext {
	var suiteCtx SuiteContext = SuiteContext{}
	suiteCtx.SuiteID = suiteID

	suiteCtx.OnlyTestOperator = onlyTestOperator
	if suiteCtx.OnlyTestOperator {
		log.Info("Only testing operator functionality")
	}

	return &suiteCtx
}

//InitSuite performs common logic for Ginkgo's BeforeSuite
func InitSuite(suiteCtx *SuiteContext) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")

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

	// suiteCtx.OnlyTestOperator = onlyTestOperator
	// if suiteCtx.OnlyTestOperator {
	// 	log.Info("Only testing operator functionality")
	// }

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

}

//TearDownSuite performs common logic for Ginkgo's AfterSuite
func TearDownSuite(suiteCtx *SuiteContext) {
	By("tearing down the test environment")

	err := suiteCtx.testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
}

//RunSuite starts the execution of a test suite
func RunSuite(t *testing.T, suiteName string, suiteCtx *SuiteContext) {

	if utils.SuiteProjectDirValue == "" {
		panic("Env var " + utils.SuiteProjectDirEnvVar + " is required")
	}

	if utils.OLMCatalogSourceImage == "" {
		panic("Env var " + utils.OLMCatalogSourceImageEnvVar + " is required")
	}

	RegisterFailHandler(Fail)

	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf(utils.SuiteProjectDirValue+"/tests-logs/"+suiteCtx.SuiteID+"/TEST-ginkgo-junit_%s.xml", time.Now().String()))

	RunSpecsWithDefaultAndCustomReporters(t, suiteName,
		[]Reporter{printer.NewlineReporter{}, junitReporter},
	)
}

package types

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	ocp_apps_client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	ocp_route_client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	kubernetescli "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"

	olmapiversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	pmversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/client/clientset/versioned"
)

//TestContext holds the common information of for a functional test
type TestContext struct {
	ID       string
	Storage  string
	Replicas int

	RegistryNamespace string

	RegistryName         string
	RegistryHost         string
	RegistryPort         string
	RegistryInternalHost string
	RegistryInternalPort string

	cleanupFunctions []func()
}

func (ctx *TestContext) RegisterCleanup(cleanup func()) {
	ctx.cleanupFunctions = append(ctx.cleanupFunctions, cleanup)
}

func (ctx *TestContext) ExecuteCleanups() {
	for i := len(ctx.cleanupFunctions) - 1; i >= 0; i-- {
		ctx.cleanupFunctions[i]()
	}
}

//SuiteContext holds common info used in a testsuite
type SuiteContext struct {
	SuiteID       string
	Cfg           *rest.Config
	K8sClient     client.Client
	K8sManager    ctrl.Manager
	TestEnv       *envtest.Environment
	PackageClient pmversioned.Interface
	OLMClient     olmapiversioned.Interface
	Clientset     *kubernetes.Clientset
	IsOpenshift   bool

	OcpAppsClient  *ocp_apps_client.AppsV1Client
	OcpRouteClient *ocp_route_client.RouteV1Client

	CLIKubernetesClient *kubernetescli.KubernetesClient

	OnlyTestOperator bool

	SetupSelenium bool
	SeleniumHost  string
	SeleniumPort  string
}

type OcpImageReference struct {
	ExternalImage string
	InternalImage string
}

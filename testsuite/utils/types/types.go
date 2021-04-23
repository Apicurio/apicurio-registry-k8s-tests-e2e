package types

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	ocp_route_client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	kubernetescli "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"

	apicurio "github.com/Apicurio/apicurio-registry-operator/api/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	olmapiversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	pmversioned "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/client/clientset/versioned"
)

var log = logf.Log.WithName("ctx")

//TestContext holds the common information of for a functional test
type TestContext struct {
	ID            string
	Storage       string
	Replicas      int
	Auth          bool
	Size          DeploymentSize
	KafkaSecurity kafkaSecurity

	RegistryNamespace string

	RegistryName         string
	RegistryHost         string
	RegistryPort         string
	RegistryInternalHost string
	RegistryInternalPort string

	RegistryResource     *apicurio.ApicurioRegistry
	KafkaClusterInfo     *KafkaClusterInfo
	KeycloakURL          string
	KeycloakSubscription *operatorsv1alpha1.Subscription

	cleanupFunctions []func()

	FunctionalTestsProfile  string
	FunctionalTestsExtraEnv []string

	SkipInfraRemoval bool
}

type kafkaSecurity string
type DeploymentSize string

var (
	NormalSize DeploymentSize = DeploymentSize("normal")
	SmallSize  DeploymentSize = DeploymentSize("small")

	Scram kafkaSecurity = kafkaSecurity("scram")
	Tls   kafkaSecurity = kafkaSecurity("tls")
)

func (ctx *TestContext) RegisterCleanup(cleanup func()) {
	ctx.cleanupFunctions = append(ctx.cleanupFunctions, cleanup)
}

func (ctx *TestContext) ExecuteCleanups() {
	log.Info("Executing cleanups", "context", ctx.ID)

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

	OcpRouteClient *ocp_route_client.RouteV1Client

	CLIKubernetesClient *kubernetescli.KubernetesClient

	OnlyTestOperator       bool
	DisableClusteredTests  bool
	DisableConvertersTests bool
	DisableAuthTests       bool

	SetupSelenium bool
	SeleniumHost  string
	SeleniumPort  string
}

type OcpImageReference struct {
	ExternalImage string
	InternalImage string
}

//KafkaClusterInfo holds useful info to use a kafka cluster
type KafkaClusterInfo struct {
	Name                     string
	Namespace                string
	Replicas                 int
	Topics                   []string
	StrimziDeployed          bool
	BootstrapServers         string
	ExternalBootstrapServers string
	AuthType                 string
	Username                 string
}

type KafkaConnectPlugin struct {
	URL       string
	SHA512SUM string
}

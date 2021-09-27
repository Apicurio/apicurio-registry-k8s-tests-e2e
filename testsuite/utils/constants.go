package utils

import (
	"os"
	"time"
)

//constants to be used in testsuite
const (
	suiteProjectDirEnvVar           = "E2E_SUITE_PROJECT_DIR"
	apicurioProjectDirEnvVar        = "E2E_APICURIO_PROJECT_DIR"
	apicurioTestsProfileEnvVar      = "E2E_APICURIO_TESTS_PROFILE"
	extraMavenArgsEnvVar            = "E2E_EXTRA_MAVEN_ARGS"
	operatorBundlePathEnvVar        = "E2E_OPERATOR_BUNDLE_PATH"
	strimziOperatorBundlePathEnvVar = "E2E_STRIMZI_BUNDLE_PATH"

	convertersURLEnvVar             = "E2E_CONVERTERS_URL"
	convertersDistroSha512SumEnvVar = "E2E_CONVERTERS_SHA512SUM"

	oLMUseDefaultCatalogSourceEnvVar       = "E2E_OLM_USE_DEFAULT_CATALOG_SOURCE"       //optional
	oLMCatalogSourceImageEnvVar            = "E2E_OLM_CATALOG_SOURCE_IMAGE"             //mandatory env var for olm tests
	oLMCatalogSourceNamespaceEnvVar        = "E2E_OLM_CATALOG_SOURCE_NAMESPACE"         //mandatory env var for olm tests
	oLMApicurioPackageManifestNameEnvVar   = "E2E_OLM_PACKAGE_MANIFEST_NAME"            //mandatory env var for olm tests
	oLMApicurioChannelNameEnvVar           = "E2E_OLM_CHANNEL"                          //mandatory env var for olm tests
	oLMApicurioCSVEnvVar                   = "E2E_OLM_CSV"                              //optional
	oLMClusterWideOperatorsNamespaceEnvVar = "E2E_OLM_CLUSTER_WIDE_OPERATORS_NAMESPACE" //mandatory env var for olm tests

	oLMUpgradeChannelEnvVar             = "E2E_OLM_UPGRADE_CHANNEL"
	oLMUpgradeOldCatalogEnvVar          = "E2E_OLM_UPGRADE_OLD_CATALOG"
	oLMUpgradeOldCatalogNamespaceEnvVar = "E2E_OLM_UPGRADE_OLD_CATALOG_NAMESPACE"
	oLMUpgradeOldCSVEnvVar              = "E2E_OLM_UPGRADE_OLD_CSV"
	oLMUpgradeNewCSVEnvVar              = "E2E_OLM_UPGRADE_NEW_CSV"

	imagePullSecretServerEnvVar   = "E2E_PULL_SECRET_SERVER"
	imagePullSecretUserEnvVar     = "E2E_PULL_SECRET_USER"
	imagePullSecretPasswordEnvVar = "E2E_PULL_SECRET_PASSWORD"
	ImagePullSecretName           = "apicurio-registry-pull-secret"

	OperatorNamespace      = "apicurio-registry-e2e"
	OperatorDeploymentName = "apicurio-registry-operator"
	APIPollInterval        = 2 * time.Second
	MediumPollInterval     = 5 * time.Second
	LongPollInterval       = 10 * time.Second

	StorageSql      = "sql"
	StorageKafkaSql = "kafkasql"
)

//SuiteProjectDir value of SuiteProjectDirEnvVar
var SuiteProjectDir string = os.Getenv(suiteProjectDirEnvVar)

//ExtraMavenArgs value of ExtraMavenArgsEnvVar
var ExtraMavenArgs string = os.Getenv(extraMavenArgsEnvVar)

//OLMCatalogSourceImage value of OLMCatalogSourceImageEnvVar
var OLMCatalogSourceImage string = os.Getenv(oLMCatalogSourceImageEnvVar)

//OperatorBundlePath value of OperatorBundlePathEnvVar
var OperatorBundlePath string = os.Getenv(operatorBundlePathEnvVar)

//ApicurioProjectDir value of ApicurioProjectDirEnvVar
var ApicurioProjectDir string = os.Getenv(apicurioProjectDirEnvVar)

//ApicurioTestsProfile value of ApicurioTestsProfileEnvVar
var ApicurioTestsProfile string = os.Getenv(apicurioTestsProfileEnvVar)

//StrimziOperatorBundlePath value of StrimziOperatorBundlePathEnvVar
var StrimziOperatorBundlePath string = os.Getenv(strimziOperatorBundlePathEnvVar)

//OLMUseDefaultCatalogSource value of oLMUseDefaultCatalogSourceEnvVar
var OLMUseDefaultCatalogSource string = os.Getenv(oLMUseDefaultCatalogSourceEnvVar)

//OLMCatalogSourceNamespace value of OLMCatalogSourceNamespaceEnvVar
var OLMCatalogSourceNamespace string = os.Getenv(oLMCatalogSourceNamespaceEnvVar)

//OLMApicurioPackageManifestName value of OLMApicurioPackageManifestNameEnvVar
var OLMApicurioPackageManifestName string = os.Getenv(oLMApicurioPackageManifestNameEnvVar)

//OLMApicurioChannelName value of oLMApicurioChannelNameEnvVar
var OLMApicurioChannelName string = os.Getenv(oLMApicurioChannelNameEnvVar)

//OLMApicurioCSV value of oLMApicurioCSVEnvVar
var OLMApicurioCSV string = os.Getenv(oLMApicurioCSVEnvVar)

//OLMClusterWideOperatorsNamespace value of OLMClusterWideOperatorsNamespaceEnvVar
var OLMClusterWideOperatorsNamespace string = os.Getenv(oLMClusterWideOperatorsNamespaceEnvVar)

var OLMUpgradeOldCSV string = os.Getenv(oLMUpgradeOldCSVEnvVar)
var OLMUpgradeNewCSV string = os.Getenv(oLMUpgradeNewCSVEnvVar)

var OLMUpgradeChannel string = os.Getenv(oLMUpgradeChannelEnvVar)
var OLMUpgradeOldCatalog string = os.Getenv(oLMUpgradeOldCatalogEnvVar)
var OLMUpgradeOldCatalogNamespace string = os.Getenv(oLMUpgradeOldCatalogNamespaceEnvVar)

var ImagePullSecretServer string = os.Getenv(imagePullSecretServerEnvVar)
var ImagePullSecretUser string = os.Getenv(imagePullSecretUserEnvVar)
var ImagePullSecretPassword string = os.Getenv(imagePullSecretPasswordEnvVar)

var ConvertersURL string = os.Getenv(convertersURLEnvVar)
var ConvertersDistroSha512Sum string = os.Getenv(convertersDistroSha512SumEnvVar)

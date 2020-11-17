package utils

import (
	"os"
	"time"
)

//constants to be used in testsuite
const (
	SuiteProjectDirEnvVar                = "E2E_SUITE_PROJECT_DIR"
	ApicurioProjectDirEnvVar             = "E2E_APICURIO_PROJECT_DIR"
	ApicurioTestsProfileEnvVar           = "E2E_APICURIO_TESTS_PROFILE"
	ExtraMavenArgsEnvVar                 = "E2E_EXTRA_MAVEN_ARGS"
	OLMCatalogSourceImageEnvVar          = "E2E_OLM_CATALOG_SOURCE_IMAGE"
	OperatorBundlePathEnvVar             = "E2E_OPERATOR_BUNDLE_PATH"
	StrimziOperatorBundlePathEnvVar      = "E2E_STRIMZI_BUNDLE_PATH"
	OLMCatalogSourceNamespaceEnvVar      = "E2E_OLM_CATALOG_SOURCE_NAMESPACE"
	OLMApicurioPackageManifestNameEnvVar = "E2E_OLM_PACKAGE_MANIFEST_NAME" //mandatory env var for olm tests

	OLMUpgradeChannelEnvVar             = "E2E_OLM_UPGRADE_CHANNEL"
	OLMUpgradeOldCatalogEnvVar          = "E2E_OLM_UPGRADE_OLD_CATALOG"
	OLMUpgradeOldCatalogNamespaceEnvVar = "E2E_OLM_UPGRADE_OLD_CATALOG_NAMESPACE"
	OLMUpgradeOldCSVEnvVar              = "E2E_OLM_UPGRADE_OLD_CSV"
	OLMUpgradeNewCSVEnvVar              = "E2E_OLM_UPGRADE_NEW_CSV"

	ImagePullSecretServerEnvVar   = "E2E_PULL_SECRET_SERVER"
	ImagePullSecretUserEnvVar     = "E2E_PULL_SECRET_USER"
	ImagePullSecretPasswordEnvVar = "E2E_PULL_SECRET_PASSWORD"
	ImagePullSecretName           = "apicurio-registry-pull-secret"

	OperatorNamespace      = "apicurio-registry-e2e"
	OperatorDeploymentName = "apicurio-registry-operator"
	APIPollInterval        = 2 * time.Second
	MediumPollInterval     = 5 * time.Second
	LongPollInterval       = 10 * time.Second

	StorageJpa        = "jpa"
	StorageStreams    = "streams"
	StorageInfinispan = "infinispan"
)

//SuiteProjectDir value of SuiteProjectDirEnvVar
var SuiteProjectDir string = os.Getenv(SuiteProjectDirEnvVar)

//ExtraMavenArgs value of ExtraMavenArgsEnvVar
var ExtraMavenArgs string = os.Getenv(ExtraMavenArgsEnvVar)

//OLMCatalogSourceImage value of OLMCatalogSourceImageEnvVar
var OLMCatalogSourceImage string = os.Getenv(OLMCatalogSourceImageEnvVar)

//OperatorBundlePath value of OperatorBundlePathEnvVar
var OperatorBundlePath string = os.Getenv(OperatorBundlePathEnvVar)

//ApicurioProjectDir value of ApicurioProjectDirEnvVar
var ApicurioProjectDir string = os.Getenv(ApicurioProjectDirEnvVar)

//ApicurioTestsProfile value of ApicurioTestsProfileEnvVar
var ApicurioTestsProfile string = os.Getenv(ApicurioTestsProfileEnvVar)

//StrimziOperatorBundlePath value of StrimziOperatorBundlePathEnvVar
var StrimziOperatorBundlePath string = os.Getenv(StrimziOperatorBundlePathEnvVar)

//OLMCatalogSourceNamespace value of OLMCatalogSourceNamespaceEnvVar
var OLMCatalogSourceNamespace string = os.Getenv(OLMCatalogSourceNamespaceEnvVar)

//OLMApicurioPackageManifestName value of OLMApicurioPackageManifestNameEnvVar
var OLMApicurioPackageManifestName string = os.Getenv(OLMApicurioPackageManifestNameEnvVar)

var OLMUpgradeOldCSV string = os.Getenv(OLMUpgradeOldCSVEnvVar)
var OLMUpgradeNewCSV string = os.Getenv(OLMUpgradeNewCSVEnvVar)

var OLMUpgradeChannel string = os.Getenv(OLMUpgradeChannelEnvVar)
var OLMUpgradeOldCatalog string = os.Getenv(OLMUpgradeOldCatalogEnvVar)
var OLMUpgradeOldCatalogNamespace string = os.Getenv(OLMUpgradeOldCatalogNamespaceEnvVar)

var ImagePullSecretServer string = os.Getenv(ImagePullSecretServerEnvVar)
var ImagePullSecretUser string = os.Getenv(ImagePullSecretUserEnvVar)
var ImagePullSecretPassword string = os.Getenv(ImagePullSecretPasswordEnvVar)

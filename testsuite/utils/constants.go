package utils

import (
	"os"
	"time"
)

//constants to be used in testsuite
const (
	SuiteProjectDirEnvVar           = "E2E_SUITE_PROJECT_DIR"
	ApicurioProjectDirEnvVar        = "E2E_APICURIO_PROJECT_DIR"
	ApicurioTestsProfileEnvVar      = "E2E_APICURIO_TESTS_PROFILE"
	ExtraMavenArgsEnvVar            = "E2E_EXTRA_MAVEN_ARGS"
	OLMCatalogSourceImageEnvVar     = "E2E_OLM_CATALOG_SOURCE_IMAGE"
	OperatorBundlePathEnvVar        = "E2E_OPERATOR_BUNDLE_PATH"
	StrimziOperatorBundlePathEnvVar = "E2E_STRIMZI_BUNDLE_PATH"

	OperatorNamespace      = "apicurio-registry-e2e"
	OperatorDeploymentName = "apicurio-registry-operator"
	APIPollInterval        = 2 * time.Second

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

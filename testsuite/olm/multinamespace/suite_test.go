package multinamespace

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/olm"
	suite "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("olm-testsuite")

var suiteCtx *types.SuiteContext

func init() {
	suite.SetFlags()
}

func TestApicurioE2E(t *testing.T) {
	suiteCtx = suite.NewSuiteContext("olm")
	suite.RunSuite(t, "Operator OLM Multinamespace Testsuite", suiteCtx)
}

var olminfo *olm.OLMInstallationInfo

var _ = BeforeSuite(func() {

	suite.InitSuite(suiteCtx)
	Expect(suiteCtx).ToNot(BeNil())

	olminfo = olm.InstallOperatorOLM(suiteCtx, utils.OLMClusterWideOperatorsNamespace, true)

})

var _ = AfterSuite(func() {

	suite.PreTearDown(suiteCtx)

	olm.UninstallOperatorOLM(suiteCtx, utils.OLMClusterWideOperatorsNamespace, true, olminfo)

	suite.TearDownSuite(suiteCtx)

})

package singlenamespace

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
	suite.RunSuite(t, "Operator OLM Singlenamespace Testsuite", suiteCtx)
}

var olminfo *olm.OLMInstallationInfo

var operatorNamespace string = utils.OperatorNamespace

var _ = BeforeSuite(func() {

	suite.InitSuite(suiteCtx)
	Expect(suiteCtx).ToNot(BeNil())

	olminfo = olm.InstallOperatorOLM(suiteCtx, operatorNamespace, false)

})

var _ = AfterSuite(func() {

	suite.PreTearDown(suiteCtx)

	olm.UninstallOperatorOLM(suiteCtx, operatorNamespace, false, olminfo)

	suite.TearDownSuite(suiteCtx)

})

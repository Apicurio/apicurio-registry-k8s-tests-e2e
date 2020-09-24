package olm

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	suite "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
)

var log = logf.Log.WithName("olm-testsuite")

var suiteCtx *suite.SuiteContext

func init() {
	suite.SetFlags()
}

func TestApicurioE2E(t *testing.T) {
	suiteCtx = suite.NewSuiteContext("olm")
	suite.RunSuite(t, "Operator OLM Testsuite", suiteCtx)
}

var _ = BeforeSuite(func(done Done) {

	suite.InitSuite(suiteCtx)
	Expect(suiteCtx).ToNot(BeNil())

	installOperatorOLM()

	close(done)

}, 120+120+5)

var _ = AfterSuite(func() {

	uninstallOperatorOLM()

	suite.TearDownSuite(suiteCtx)

}, 245)

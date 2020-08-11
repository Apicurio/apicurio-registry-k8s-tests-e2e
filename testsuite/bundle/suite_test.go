package bundle

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	suite "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
)

var log = logf.Log.WithName("bundle-testsuite")

var suiteCtx *suite.SuiteContext

func init() {
	suite.SetFlags()
}

func TestApicurioE2E(t *testing.T) {
	suite.RunSuite(t, "Operator Bundle Testsuite", "bundle")
}

var _ = BeforeSuite(func(done Done) {

	suiteCtx = suite.InitSuite("bundle")
	Expect(suiteCtx).ToNot(BeNil())

	installOperator()

	close(done)

}, 120+120+5)

var _ = AfterSuite(func() {

	uninstallOperator()

	suite.TearDownSuite(suiteCtx)

})

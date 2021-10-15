package bundle

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	suite "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("bundle-testsuite")

var suiteCtx *types.SuiteContext

func init() {
	suite.SetFlags()
}

func TestApicurioE2E(t *testing.T) {
	suiteCtx = suite.NewSuiteContext("bundle")
	suite.RunSuite(t, "Operator Bundle Testsuite", suiteCtx)
}

var _ = BeforeSuite(func(done Done) {

	suite.InitSuite(suiteCtx)
	Expect(suiteCtx).ToNot(BeNil())

	installOperator()

	close(done)

}, 5*60)

var _ = AfterSuite(func() {

	suite.PreTearDown(suiteCtx)

	uninstallOperator()

	suite.TearDownSuite(suiteCtx)

}, 5*60)

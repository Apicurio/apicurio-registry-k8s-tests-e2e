package multinamespace

import (
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/testcase"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("multinamespaced olm installation", func() {
	testcase.MultinamespacedTestCase(suiteCtx)
})

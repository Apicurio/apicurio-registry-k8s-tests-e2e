package singlenamespace

import (
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/testcase"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("olm installation", func() {
	testcase.CommonTestCases(suiteCtx, operatorNamespace)
	if suiteCtx.OLMRunAdvancedTestcases {
		testcase.AdvancedTestCases(suiteCtx, operatorNamespace)
	}
})

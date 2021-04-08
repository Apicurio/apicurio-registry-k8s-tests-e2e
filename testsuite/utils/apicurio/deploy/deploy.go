package deploy

import (
	"errors"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kafkasql"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/sql"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	. "github.com/onsi/gomega"
)

func DeployRegistryStorage(suiteCtx *types.SuiteContext, ctx *types.TestContext) {
	if ctx.Storage == utils.StorageSql {
		sql.DeploySqlRegistry(suiteCtx, ctx)
	} else if ctx.Storage == utils.StorageKafkaSql {
		kafkasql.DeployKafkaSqlRegistry(suiteCtx, ctx)
	} else {
		Expect(errors.New("Storage not implemented")).ToNot(HaveOccurred())
	}
}

func RemoveRegistryDeployment(suiteCtx *types.SuiteContext, ctx *types.TestContext) {
	if ctx.Storage == utils.StorageSql {
		sql.RemoveJpaRegistry(suiteCtx, ctx)
	} else if ctx.Storage == utils.StorageKafkaSql {
		kafkasql.RemoveKafkaSqlRegistry(suiteCtx, ctx)
	} else {
		Expect(errors.New("Storage not implemented")).ToNot(HaveOccurred())
	}
}

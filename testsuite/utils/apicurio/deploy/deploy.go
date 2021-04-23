package deploy

import (
	"errors"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	apicurioutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kafkasql"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/keycloak"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/sql"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
	apicurio "github.com/Apicurio/apicurio-registry-operator/api/v1"

	. "github.com/onsi/gomega"
)

func DeployRegistryStorage(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	var registry *apicurio.ApicurioRegistry
	if ctx.Storage == utils.StorageSql {
		registry = sql.SqlDeployResource(suiteCtx, ctx)
	} else if ctx.Storage == utils.StorageKafkaSql {
		registry = kafkasql.KafkaSqlDeployResource(suiteCtx, ctx)
	} else {
		Expect(errors.New("Storage not implemented")).ToNot(HaveOccurred())
	}

	if ctx.Auth {
		registry.Spec.Configuration.Security.Keycloak = keycloak.KeycloakConfigResource(ctx)
	}

	apicurioutils.CreateRegistryAndWait(suiteCtx, ctx, registry)

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

package kafkasql

import (
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
	"github.com/google/uuid"
)

func DeploySharedKafkaIfNeeded(suiteCtx *types.SuiteContext, ctx *types.TestContext) *types.KafkaClusterInfo {
	if isSharedKafkaNeeded(suiteCtx) {
		log.Info("Deploying Shared Kafka cluster for tests")
		kafkaRequest := &CreateKafkaClusterRequest{
			Name:           "shared-kafka-" + uuid.NewString()[:5],
			Namespace:      ctx.RegistryNamespace,
			ExposeExternal: true,
			Replicas:       1,
			Topics:         []string{},
			Security:       "",
		}
		return DeployKafkaCluster(suiteCtx, kafkaRequest)
	}
	return nil
}

func RemoveSharedKafkaIfNeeded(suiteCtx *types.SuiteContext, ctx *types.TestContext, kafkaCluster *types.KafkaClusterInfo) {
	if isSharedKafkaNeeded(suiteCtx) && kafkaCluster != nil {
		log.Info("Removing Shared Kafka cluster for tests")
		RemoveKafkaCluster(suiteCtx.Clientset, ctx.RegistryNamespace, kafkaCluster)
	}
}

func isSharedKafkaNeeded(suiteCtx *types.SuiteContext) bool {
	return utils.ApicurioTestsProfile != "" && (utils.ApicurioTestsProfile == "all" || utils.ApicurioTestsProfile == "acceptance" || utils.ApicurioTestsProfile == "serdes")
}

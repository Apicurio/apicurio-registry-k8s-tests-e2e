package converters

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/jpa"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/streams"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var log = logf.Log.WithName("postgresql")

var debeziumName string = "apicurio-debezium"
var labels map[string]string = map[string]string{"apicurio": "qe"}

var databaseName = "test-db"
var databaseUser = "testuser"
var databasePassword = "testpwd"

func ConvertersTestCase(suiteCtx *suite.SuiteContext, testContext *types.TestContext) {
	apicurioURL := "http://" + testContext.RegistryHost + ":" + testContext.RegistryPort + "/api/"

	oldDir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	apicurioDebeziumDistroDir := utils.SuiteProjectDirValue + "/scripts/converters"
	os.Chdir(apicurioDebeziumDistroDir)
	err = utils.ExecuteCmd(true, &utils.Command{Cmd: []string{"make", "build", "push"}})
	os.Chdir(oldDir)

	apicurioDebeziumImage := "localhost:5000/apicurio-debezium:latest"

	kafkaClusterName := "test-debezium-kafka"
	var kafkaClusterInfo streams.KafkaClusterInfo = streams.DeployKafkaCluster(suiteCtx.Clientset, 1, kafkaClusterName, []string{})
	if kafkaClusterInfo.StrimziDeployed {
		kafkaCleanup := func() {
			streams.RemoveKafkaCluster(suiteCtx.Clientset, kafkaClusterName, []string{})
			streams.RemoveStrimziOperator(suiteCtx.Clientset)
		}
		testContext.RegisterCleanup(kafkaCleanup)
	}

	jpa.DeployPostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, databaseName, databaseName, databaseUser, databasePassword)
	postgresCleanup := func() {
		jpa.RemovePostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, databaseName)
	}
	testContext.RegisterCleanup(postgresCleanup)

	log.Info("Deploying debezium")
	err = suiteCtx.K8sClient.Create(context.TODO(), debeziumDeployment(apicurioDebeziumImage, kafkaClusterInfo.BootstrapServers))
	Expect(err).ToNot(HaveOccurred())
	err = suiteCtx.K8sClient.Create(context.TODO(), debeziumService())
	Expect(err).ToNot(HaveOccurred())
	err = suiteCtx.K8sClient.Create(context.TODO(), debeziumIngress(suiteCtx))
	Expect(err).ToNot(HaveOccurred())

	//TODO adapt this to work on openshift
	debeziumURL := "http://localhost:80/debezium"

	debeziumCleanup := func() {
		log.Info("Removing debezium")
		err := suiteCtx.K8sClient.Delete(context.TODO(), debeziumDeployment(apicurioDebeziumImage, kafkaClusterInfo.BootstrapServers))
		Expect(err).ToNot(HaveOccurred())
		err = suiteCtx.K8sClient.Delete(context.TODO(), debeziumService())
		Expect(err).ToNot(HaveOccurred())
		err = suiteCtx.K8sClient.Delete(context.TODO(), debeziumIngress(nil))
		Expect(err).ToNot(HaveOccurred())
	}
	testContext.RegisterCleanup(debeziumCleanup)

	kubernetesutils.WaitForDeploymentReady(suiteCtx.Clientset, 120*time.Second, debeziumName, 1)

	postgresqlPodName := jpa.GetPostgresqlDatabasePod(suiteCtx.Clientset, databaseName).Name
	executeSQL(postgresqlPodName, databaseName, "drop schema if exists todo cascade")
	executeSQL(postgresqlPodName, databaseName, "create schema todo")
	executeSQL(postgresqlPodName, databaseName, "create table todo.Todo (id int8 not null, title varchar(255), primary key (id))")
	executeSQL(postgresqlPodName, databaseName, "alter table todo.Todo replica identity full")

	createDebeziumJdbcConnector(debeziumURL, "my-connector-avro", "io.apicurio.registry.utils.converter.AvroConverter", apicurioURL, map[string]interface{}{
		"key.converter.apicurio.registry.converter.serializer":     "io.apicurio.registry.utils.serde.AvroKafkaSerializer",
		"key.converter.apicurio.registry.converter.deserializer":   "io.apicurio.registry.utils.serde.AvroKafkaDeserializer",
		"value.converter.apicurio.registry.converter.serializer":   "io.apicurio.registry.utils.serde.AvroKafkaSerializer",
		"value.converter.apicurio.registry.converter.deserializer": "io.apicurio.registry.utils.serde.AvroKafkaDeserializer",
	})

	executeSQL(postgresqlPodName, databaseName, "insert into todo.Todo values (1, 'Be Awesome')")
	executeSQL(postgresqlPodName, databaseName, "insert into todo.Todo values (2, 'Even more')")
	executeSQL(postgresqlPodName, databaseName, "select * from todo.Todo")

	kafkaConsumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaClusterInfo.BootstrapServers,
		"group.id":          "apicurio-registry-test",
		"auto.offset.reset": "earliest",
	})
	Expect(err).ToNot(HaveOccurred())

	kafkaConsumer.SubscribeTopics([]string{"dbserver2.todo.todo"}, nil)

	records := drainKafka(kafkaConsumer, 2)

	log.Info("Kafka record")
	log.Info(records[0].String())
	log.Info(string(records[0].Key))
	log.Info(string(records[0].Value))

	Expect(records[0].Key[0]).To(Equal(byte(0)))
	Expect(records[0].Value[0]).To(Equal(byte(0)))

	artifactsRes, err := http.Get(apicurioURL + "artifacts")
	Expect(err).ToNot(HaveOccurred())
	artifactsStr := utils.ReaderToString(artifactsRes.Body)
	log.Info("Artifacts after debezium are " + artifactsStr)

	kafkaConsumer.Unsubscribe()
	kafkaConsumer.Close()

}

func drainKafka(c *kafka.Consumer, expectedRecords int) []*kafka.Message {
	var records []*kafka.Message = make([]*kafka.Message, 0)
	log.Info("Waiting for kafka consumer to receive " + strconv.Itoa(expectedRecords) + " records")
	timeout := 60 * time.Second
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		msg, err := c.ReadMessage(50 * time.Millisecond)
		if err != nil {
			if err.(kafka.Error).Code() == kafka.ErrTimedOut {
				return false, nil
			}
			return false, err
		}
		records = append(records, msg)
		return len(records) >= expectedRecords, nil
	})
	Expect(err).ToNot(HaveOccurred())
	return records
}

type DebeziumConnector struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

func createDebeziumJdbcConnector(debeziumURL string, connectorName string, converter string, apicurioURL string, extraConfig map[string]interface{}) {
	connector := &DebeziumConnector{
		Name: connectorName,
		Config: map[string]interface{}{
			"tasks.max":         1,
			"database.hostname": databaseName,
			"database.port":     5432,
			"database.user":     databaseUser,
			"database.password": databasePassword,
			"database.dbname":   databaseName,
			"connector.class":   "io.debezium.connector.postgresql.PostgresConnector",
			//test specific
			"database.server.name":                        "dbserver2",
			"slot.name":                                   "debezium_2",
			"key.converter":                               converter,
			"key.converter.apicurio.registry.url":         apicurioURL,
			"key.converter.apicurio.registry.global-id":   "io.apicurio.registry.utils.serde.strategy.AutoRegisterIdStrategy",
			"value.converter":                             converter,
			"value.converter.apicurio.registry.url":       apicurioURL,
			"value.converter.apicurio.registry.global-id": "io.apicurio.registry.utils.serde.strategy.AutoRegisterIdStrategy",
		},
	}
	for k, v := range extraConfig {
		connector.Config[k] = v
	}

	json, err := json.Marshal(connector)
	Expect(err).ToNot(HaveOccurred())

	log.Info("Going to create debezium connector " + string(json))

	body := bytes.NewReader(json)

	//register connector
	res, err := http.Post(debeziumURL+"/connectors/", "application/json", body)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.StatusCode >= 200 && res.StatusCode <= 299).To(BeTrue())
	log.Info("Create connector response is " + res.Status)
	log.Info("Create connector response is " + utils.ReaderToString(res.Body))

	log.Info("Waiting for debezium connector to be configured")
	timeout := 45 * time.Second
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		res, err := http.Get(debeziumURL + "/connectors/" + connectorName)
		if err != nil {
			return false, err
		}
		if res.StatusCode >= 200 && res.StatusCode <= 299 {
			log.Info("Status code is " + res.Status)
			log.Info("Debezium connector is " + utils.ReaderToString(res.Body))
			return true, nil
		}
		return false, nil
	})
	Expect(err).ToNot(HaveOccurred())
}

func executeSQL(podName string, databaseName string, sql string) {
	kubernetescli.Execute("exec", podName, "--", "psql", "-d", databaseName, "-c", sql)
}

func debeziumDeployment(image string, bootstrapServers string) *v1.Deployment {
	var replicas int32 = 1
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      debeziumName,
			Namespace: utils.OperatorNamespace,
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  debeziumName,
							Image: image,
							Env: []corev1.EnvVar{
								{
									Name:  "BOOTSTRAP_SERVERS",
									Value: bootstrapServers,
								},
								{
									Name:  "GROUP_ID",
									Value: "1",
								},
								{
									Name:  "CONFIG_STORAGE_TOPIC",
									Value: "debezium_connect_config",
								},
								{
									Name:  "OFFSET_STORAGE_TOPIC",
									Value: "debezium_connect_offsets",
								},
								{
									Name:  "STATUS_STORAGE_TOPIC",
									Value: "debezium_connect_status",
								},
								{
									Name:  "CONNECT_KEY_CONVERTER_SCHEMAS_ENABLE",
									Value: "false",
								},
								{
									Name:  "CONNECT_VALUE_CONVERTER_SCHEMAS_ENABLE",
									Value: "false",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8083,
									Name:          "http",
									Protocol:      "TCP",
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/connectors",
										Port: intstr.FromInt(8083),
									},
								},
								InitialDelaySeconds: 25,
								PeriodSeconds:       10,
								TimeoutSeconds:      300,
								SuccessThreshold:    2,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/connectors",
										Port: intstr.FromInt(8083),
									},
								},
								InitialDelaySeconds: 25,
								PeriodSeconds:       15,
							},
						},
					},
				},
			},
		},
	}
}

func debeziumService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      debeziumName,
			Namespace: utils.OperatorNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       8083,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(8083),
				},
			},
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func debeziumIngress(suite *suite.SuiteContext) *v1beta1.Ingress {
	ingress := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      debeziumName,
			Namespace: utils.OperatorNamespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/debezium(/|$)(.*)",
									Backend: v1beta1.IngressBackend{
										ServiceName: debeziumName,
										ServicePort: intstr.FromInt(8083),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if suite != nil && !suite.IsOpenshift {
		//this is just a workaround to make it work with Kind nginx ingress
		ingress.Spec.Rules[0].Host = "localhost"
	}
	return ingress
}

package converters

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"github.com/segmentio/kafka-go"
	"k8s.io/apimachinery/pkg/util/wait"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	routev1 "github.com/openshift/api/route/v1"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	apicurioclient "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio/client"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/jpa"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/openshift"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/streams"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("converters")

var debeziumName string = "apicurio-debezium"
var labels map[string]string = map[string]string{"apicurio": "qe"}

var databaseName = "testdb"
var databaseUser = "testuser"
var databasePassword = "testpwd"

func ConvertersTestCase(suiteCtx *types.SuiteContext, testContext *types.TestContext) {

	apicurioDebeziumImage := &types.OcpImageReference{
		ExternalImage: "localhost:5000/apicurio-debezium:latest",
		InternalImage: "localhost:5000/apicurio-debezium:latest",
	}

	if suiteCtx.IsOpenshift {
		apicurioDebeziumImage = openshift.OcpInternalImage(suiteCtx, testContext.RegistryNamespace, "apicurio-debezium", "latest")
	}

	apicurioDebeziumDistroDir := utils.SuiteProjectDir + "/scripts/converters"
	utils.ExecuteCmdOrDie(true, "docker", "build", "-t", apicurioDebeziumImage.ExternalImage, apicurioDebeziumDistroDir)
	utils.ExecuteCmdOrDie(true, "docker", "push", apicurioDebeziumImage.ExternalImage)

	kafkaClusterName := "test-debezium-kafka"
	var kafkaClusterInfo *streams.KafkaClusterInfo = streams.DeployKafkaClusterV2(suiteCtx, testContext.RegistryNamespace, 1, true, kafkaClusterName, []string{})
	if kafkaClusterInfo.StrimziDeployed {
		kafkaCleanup := func() {
			streams.RemoveKafkaCluster(suiteCtx.Clientset, testContext.RegistryNamespace, kafkaClusterName, []string{})
			streams.RemoveStrimziOperator(suiteCtx.Clientset, testContext.RegistryNamespace)
		}
		testContext.RegisterCleanup(kafkaCleanup)
	}

	jpa.DeployPostgresqlDatabase(suiteCtx, testContext.RegistryNamespace, databaseName, databaseName, databaseUser, databasePassword)
	postgresCleanup := func() {
		jpa.RemovePostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, testContext.RegistryNamespace, databaseName)
	}
	testContext.RegisterCleanup(postgresCleanup)

	log.Info("Deploying debezium")
	err := suiteCtx.K8sClient.Create(context.TODO(), debeziumDeployment(testContext.RegistryNamespace, apicurioDebeziumImage.InternalImage, kafkaClusterInfo.BootstrapServers))
	Expect(err).ToNot(HaveOccurred())
	err = suiteCtx.K8sClient.Create(context.TODO(), debeziumService(testContext.RegistryNamespace))
	Expect(err).ToNot(HaveOccurred())
	if suiteCtx.IsOpenshift {
		_, err = suiteCtx.OcpRouteClient.Routes(testContext.RegistryNamespace).Create(ocpDebeziumRoute(testContext.RegistryNamespace))
		Expect(err).ToNot(HaveOccurred())
	} else {
		err = suiteCtx.K8sClient.Create(context.TODO(), kindDebeziumIngress(testContext.RegistryNamespace))
		Expect(err).ToNot(HaveOccurred())
	}

	debeziumCleanup := func() {
		log.Info("Removing debezium")
		err := suiteCtx.K8sClient.Delete(context.TODO(), debeziumDeployment(testContext.RegistryNamespace, apicurioDebeziumImage.InternalImage, kafkaClusterInfo.BootstrapServers))
		Expect(err).ToNot(HaveOccurred())
		err = suiteCtx.K8sClient.Delete(context.TODO(), debeziumService(testContext.RegistryNamespace))
		Expect(err).ToNot(HaveOccurred())
		if suiteCtx.IsOpenshift {
			err = suiteCtx.OcpRouteClient.Routes(testContext.RegistryNamespace).Delete(debeziumName, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		} else {
			err = suiteCtx.K8sClient.Delete(context.TODO(), kindDebeziumIngress(testContext.RegistryNamespace))
			Expect(err).ToNot(HaveOccurred())
		}
	}
	testContext.RegisterCleanup(debeziumCleanup)

	kubernetesutils.WaitForDeploymentReady(suiteCtx.Clientset, 120*time.Second, testContext.RegistryNamespace, debeziumName, 1)

	debeziumURL := "http://localhost:80/debezium"
	if suiteCtx.IsOpenshift {
		debeziumRoute, err := suiteCtx.OcpRouteClient.Routes(testContext.RegistryNamespace).Get(debeziumName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		debeziumURL = "http://" + debeziumRoute.Status.Ingress[0].Host
	}

	postgresqlPodName := jpa.GetPostgresqlDatabasePod(suiteCtx.Clientset, testContext.RegistryNamespace, databaseName).Name
	executeSQL(testContext.RegistryNamespace, postgresqlPodName, "drop schema if exists todo cascade")
	executeSQL(testContext.RegistryNamespace, postgresqlPodName, "create schema todo")
	executeSQL(testContext.RegistryNamespace, postgresqlPodName, "create table todo.Todo (id int8 not null, title varchar(255), primary key (id))")
	executeSQL(testContext.RegistryNamespace, postgresqlPodName, "alter table todo.Todo replica identity full")

	var registryInternalURL string = "http://" + testContext.RegistryInternalHost + ":" + testContext.RegistryInternalPort + "/api/"
	var debeziumTopic string = "dbserver2.todo.todo"
	extraConfig := map[string]interface{}{
		"key.converter.apicurio.registry.converter.serializer":     "io.apicurio.registry.utils.serde.AvroKafkaSerializer",
		"key.converter.apicurio.registry.converter.deserializer":   "io.apicurio.registry.utils.serde.AvroKafkaDeserializer",
		"value.converter.apicurio.registry.converter.serializer":   "io.apicurio.registry.utils.serde.AvroKafkaSerializer",
		"value.converter.apicurio.registry.converter.deserializer": "io.apicurio.registry.utils.serde.AvroKafkaDeserializer",
	}
	if suiteCtx.IsOpenshift {
		// because we are using a different postgres image when running on openshift
		// the postgres image we are using is provided by debezium, and the image we are using is prepared to us pgoutput replication
		extraConfig["plugin.name"] = "pgoutput"
	} else {
		// the postgres image we use for kubernetes is as well provided by debezium and it's configured to work with decoderbufs
		extraConfig["plugin.name"] = "decoderbufs"
	}
	createDebeziumJdbcConnector(debeziumURL,
		"my-connector-avro",
		"io.apicurio.registry.utils.converter.AvroConverter",
		registryInternalURL,
		extraConfig,
	)

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		TLS: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaClusterInfo.ExternalBootstrapServers},
		GroupID: "apicurio-registry-test",
		Topic:   debeziumTopic,
		Dialer:  dialer,
	})

	expectedRecords := 2
	executeSQL(testContext.RegistryNamespace, postgresqlPodName, "insert into todo.Todo values (1, 'Be Awesome')")
	executeSQL(testContext.RegistryNamespace, postgresqlPodName, "insert into todo.Todo values (2, 'Even more')")
	executeSQL(testContext.RegistryNamespace, postgresqlPodName, "select * from todo.Todo")

	log.Info("Waiting for kafka consumer to receive " + strconv.Itoa(expectedRecords) + " records")

	var records []*kafka.Message = make([]*kafka.Message, 0)
	for {
		timeout, cf := context.WithTimeout(context.Background(), 60*time.Second)
		m, err := r.ReadMessage(timeout)
		cf()
		if err != nil {
			log.Info("Error " + err.Error())
			break
		}
		log.Info("kafka message received")
		log.Info(string(m.Value))
		records = append(records, &m)
		if len(records) >= expectedRecords {
			break
		}
	}

	if err := r.Close(); err != nil {
		log.Info("failed to close reader:", err)
		Expect(err).NotTo(HaveOccurred())
	}

	Expect(len(records)).To(BeIdenticalTo(expectedRecords))

	Expect(records[0].Key[0]).To(Equal(byte(0)))
	Expect(records[0].Value[0]).To(Equal(byte(0)))

	apicurio := apicurioclient.NewApicurioRegistryApiClient(testContext.RegistryHost, testContext.RegistryPort, http.DefaultClient)
	artifacts, err := apicurio.ListArtifacts()
	Expect(err).ToNot(HaveOccurred())
	log.Info("Artifacts after debezium are " + strings.Join(artifacts, ", "))
	Expect(artifacts).Should(ContainElements(debeziumTopic+"-key", debeziumTopic+"-value"))

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

func executeSQL(namespace string, podName string, sql string) {
	kubernetescli.Execute("-n", namespace, "exec", podName, "--", "psql", "-d", databaseName, "-U", databaseUser, "-c", sql)
}

func debeziumDeployment(namespace string, image string, bootstrapServers string) *v1.Deployment {
	var replicas int32 = 1
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      debeziumName,
			Namespace: namespace,
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

func debeziumService(namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      debeziumName,
			Namespace: namespace,
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

func kindDebeziumIngress(namespace string) *v1beta1.Ingress {
	return &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      debeziumName,
			Namespace: namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "localhost",
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
}

func ocpDebeziumRoute(namespace string) *routev1.Route {
	var weigh int32 = 100
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      debeziumName,
			Namespace: namespace,
		},
		Spec: routev1.RouteSpec{
			Path: "/",
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   debeziumName,
				Weight: &weigh,
			},
		},
	}
}

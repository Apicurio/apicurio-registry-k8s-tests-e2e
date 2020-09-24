package jpa

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	suite "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	types "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var log = logf.Log.WithName("postgresql")

const registryPostgresqlName string = "registry-db"

var registryName string

//DeployJpaRegistry deploys a posgresql database and deploys an ApicurioRegistry CR using that database
func DeployJpaRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	user := "apicurio-registry"
	password := "password"
	dataSourceURL := DeployPostgresqlDatabase(suiteCtx.K8sClient, clientset, registryPostgresqlName, "apicurio-registry", user, password)

	log.Info("Deploying apicurio registry")

	registryName = "apicurio-registry-" + ctx.Storage
	registry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: utils.OperatorNamespace,
			Name:      registryName,
		},
		Spec: apicurio.ApicurioRegistrySpec{
			Deployment: apicurio.ApicurioRegistrySpecDeployment{
				//TODO detect if cluster is kind and do this workaround only in that case
				Host: "localhost",
			},
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				LogLevel:    "DEBUG",
				Persistence: utils.StorageJpa,
				DataSource: apicurio.ApicurioRegistrySpecConfigurationDataSource{
					Url:      string(dataSourceURL),
					UserName: user,
					Password: password,
				},
			},
		},
	}

	err := suiteCtx.K8sClient.Create(context.TODO(), &registry)
	Expect(err).ToNot(HaveOccurred())

	utils.WaitForRegistryReady(suiteCtx.K8sClient, clientset, registryName)

	ctx.RegistryHost = "localhost"
	ctx.RegistryPort = "80"
}

//RemoveJpaRegistry uninstalls registry CR and postgresql database
func RemoveJpaRegistry(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	var clientset *kubernetes.Clientset = kubernetes.NewForConfigOrDie(suiteCtx.Cfg)
	Expect(clientset).ToNot(BeNil())

	utils.DeleteRegistryAndWait(suiteCtx.K8sClient, clientset, registryName)

	RemovePostgresqlDatabase(suiteCtx.K8sClient, clientset, registryPostgresqlName)

}

//DeployPostgresqlDatabase deploys a postgresql database
func DeployPostgresqlDatabase(k8sclient client.Client, clientset *kubernetes.Clientset, name string, database string, user string, password string) string {
	log.Info("Deploying postgresql database " + name)

	err := k8sclient.Create(context.TODO(), postgresqlPersistentVolumeClaim(name))
	Expect(err).ToNot(HaveOccurred())
	err = k8sclient.Create(context.TODO(), postgresqlDeployment(name, database, user, password))
	Expect(err).ToNot(HaveOccurred())
	err = k8sclient.Create(context.TODO(), postgresqlService(name))
	Expect(err).ToNot(HaveOccurred())

	timeout := 120 * time.Second
	log.Info("Waiting for postgresql database to be ready ", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get(name, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		if od != nil {
			if od.Status.AvailableReplicas > int32(0) {
				return true, nil
			}
		}
		return false, nil
	})
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())

	svc, err := clientset.CoreV1().Services(utils.OperatorNamespace).Get(name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return "jdbc:postgresql://" + svc.Spec.ClusterIP + ":5432/"
}

//GetPostgresqlDatabasePod gets the database pod from the name given when created
func GetPostgresqlDatabasePod(clientset *kubernetes.Clientset, name string) *corev1.Pod {
	labelsSet := labels.Set(map[string]string{"app": name})
	podList, err := clientset.CoreV1().Pods(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	Expect(err).ToNot(HaveOccurred())
	return &podList.Items[0]
}

//RemovePostgresqlDatabase removes a postgresql database
func RemovePostgresqlDatabase(k8sclient client.Client, clientset *kubernetes.Clientset, name string) {
	log.Info("Removing postgresql database " + name)

	err := k8sclient.Delete(context.TODO(), postgresqlPersistentVolumeClaim(name))
	Expect(err).ToNot(HaveOccurred())
	err = k8sclient.Delete(context.TODO(), postgresqlDeployment(name, "", "", ""))
	Expect(err).ToNot(HaveOccurred())
	err = k8sclient.Delete(context.TODO(), postgresqlService(name))
	Expect(err).ToNot(HaveOccurred())

	timeout := 30 * time.Second
	log.Info("Waiting for postgresql database to be removed ", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := clientset.AppsV1().Deployments(utils.OperatorNamespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	utils.ExecuteCmdOrDie(true, "kubectl", "get", "pod", "-n", utils.OperatorNamespace)
	Expect(err).ToNot(HaveOccurred())
}

func postgresqlDeployment(name string, database string, user string, password string) *v1.Deployment {
	labels := map[string]string{"app": name}
	var replicas int32 = 1
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      name,
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
							Name:  name,
							Image: "centos/postgresql-10-centos7",
							Env: []corev1.EnvVar{
								{
									Name:  "POSTGRESQL_ADMIN_PASSWORD",
									Value: "admin1234",
								},
								{
									Name:  "POSTGRESQL_DATABASE",
									Value: database,
								},
								{
									Name:  "POSTGRESQL_PASSWORD",
									Value: password,
								},
								{
									Name:  "POSTGRESQL_USER",
									Value: user,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5432,
									Name:          "postgresql",
									Protocol:      "TCP",
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(5432),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(5432),
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       20,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/var/lib/pgsql/data",
									Name:      name,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: name,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: name,
								},
							},
						},
					},
				},
			},
		},
	}
}

func postgresqlService(name string) *corev1.Service {
	labels := map[string]string{"app": name}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      name,
			Namespace: utils.OperatorNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       5432,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(5432),
				},
			},
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func postgresqlPersistentVolumeClaim(name string) *corev1.PersistentVolumeClaim {
	labels := map[string]string{"app": name}
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      name,
			Namespace: utils.OperatorNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("300Mi"),
				},
			},
		},
	}
}

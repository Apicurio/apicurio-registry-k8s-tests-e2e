package sql

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	apicurioutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	kubernetescli "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
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

	apicurio "github.com/Apicurio/apicurio-registry-operator/api/v1"
)

var log = logf.Log.WithName("postgresql")

const registryPostgresqlName string = "registry-db"

//DeploySqlRegistry deploys a posgresql database and deploys an ApicurioRegistry CR using that database
func DeploySqlRegistry(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	user := "apicuriouser"
	password := "password"
	dataSourceURL := DeployPostgresqlDatabase(suiteCtx, ctx.RegistryNamespace, registryPostgresqlName, "apicurioregistry", user, password).DataSourceURL

	log.Info("Deploying apicurio registry")

	replicas := 1
	if ctx.Replicas > 0 {
		replicas = ctx.Replicas
	}

	registry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "apicurio-registry-" + ctx.Storage,
		},
		Spec: apicurio.ApicurioRegistrySpec{
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				LogLevel:    "DEBUG",
				Persistence: utils.StorageSql,
				Sql: apicurio.ApicurioRegistrySpecConfigurationSql{
					DataSource: apicurio.ApicurioRegistrySpecConfigurationDataSource{
						Url:      string(dataSourceURL),
						UserName: user,
						Password: password,
					},
				},
			},
			Deployment: apicurio.ApicurioRegistrySpecDeployment{
				Replicas: int32(replicas),
			},
		},
	}

	apicurioutils.CreateRegistryAndWait(suiteCtx, ctx, &registry)

}

//RemoveJpaRegistry uninstalls registry CR and postgresql database
func RemoveJpaRegistry(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	apicurioutils.DeleteRegistryAndWait(suiteCtx, ctx.RegistryNamespace, ctx.RegistryName)

	RemovePostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, ctx.RegistryNamespace, registryPostgresqlName)

}

type DbData struct {
	Name          string
	Host          string
	Port          string
	Database      string
	User          string
	Password      string
	DataSourceURL string
}

//DeployPostgresqlDatabase deploys a postgresql database
func DeployPostgresqlDatabase(suiteCtx *types.SuiteContext, namespace string, name string, database string, user string, password string) *DbData {
	log.Info("Deploying postgresql database " + name)

	err := suiteCtx.K8sClient.Create(context.TODO(), postgresqlPersistentVolumeClaim(namespace, name))
	Expect(err).ToNot(HaveOccurred())
	if suiteCtx.IsOpenshift {
		err = suiteCtx.K8sClient.Create(context.TODO(), openshiftPostgresqlDeployment(namespace, name, database, user, password))
		Expect(err).ToNot(HaveOccurred())
	} else {
		err = suiteCtx.K8sClient.Create(context.TODO(), postgresqlDeployment(namespace, name, database, user, password))
		Expect(err).ToNot(HaveOccurred())
	}
	err = suiteCtx.K8sClient.Create(context.TODO(), postgresqlService(namespace, name))
	Expect(err).ToNot(HaveOccurred())

	timeout := 180 * time.Second
	log.Info("Waiting for postgresql database to be ready ", "timeout", timeout)
	err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := suiteCtx.Clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
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
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())

	svc, err := suiteCtx.Clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	dbdata := &DbData{
		Name:          name,
		DataSourceURL: "jdbc:postgresql://" + svc.Spec.ClusterIP + ":5432/" + database,
		Host:          svc.Spec.ClusterIP,
		Port:          "5432",
		Database:      database,
		User:          user,
		Password:      password,
	}
	return dbdata
}

//GetPostgresqlDatabasePod gets the database pod from the name given when created
func GetPostgresqlDatabasePod(clientset *kubernetes.Clientset, namespace string, name string) *corev1.Pod {
	labelsSet := labels.Set(map[string]string{"app": name})
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	Expect(err).ToNot(HaveOccurred())
	return &podList.Items[0]
}

//RemovePostgresqlDatabase removes a postgresql database
func RemovePostgresqlDatabase(k8sclient client.Client, clientset *kubernetes.Clientset, namespace string, name string) {
	log.Info("Removing postgresql database " + name)

	dep, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		err = k8sclient.Delete(context.TODO(), dep)
		Expect(err).ToNot(HaveOccurred())
		kubernetesutils.WaitForObjectDeleted("Deployment "+name, func() (interface{}, error) {
			return clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		})
	} else if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		err = k8sclient.Delete(context.TODO(), pvc)
		Expect(err).ToNot(HaveOccurred())
		kubernetesutils.WaitForObjectDeleted("PVC "+name, func() (interface{}, error) {
			return clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		})
	} else if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		err = k8sclient.Delete(context.TODO(), svc)
		Expect(err).ToNot(HaveOccurred())
		kubernetesutils.WaitForObjectDeleted("Service "+name, func() (interface{}, error) {
			return clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		})
	} else if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	kubernetescli.GetPods(namespace)
}

func postgresqlDeployment(namespace string, name string, database string, user string, password string) *v1.Deployment {
	labels := map[string]string{"app": name}
	var replicas int32 = 1
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      name,
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
							Name: name,
							// Image: "centos/postgresql-10-centos7:20200917-804ef01",
							// Image: "quay.io/debezium/example-postgres:1.2",
							// Image: "quay.io/debezium/postgres:10",
							Image: "quay.io/debezium/postgres:12",
							Env: []corev1.EnvVar{
								// {
								// 	Name:  "POSTGRESQL_ADMIN_PASSWORD",
								// 	Value: "admin1234",
								// },
								{
									// Name:  "POSTGRESQL_DATABASE",
									Name:  "POSTGRES_DB",
									Value: database,
								},
								{
									// Name:  "POSTGRESQL_PASSWORD",
									Name:  "POSTGRES_PASSWORD",
									Value: password,
								},
								{
									// Name:  "POSTGRESQL_USER",
									Name:  "POSTGRES_USER",
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
									// MountPath: "/var/lib/pgsql/data",
									MountPath: "/var/lib/postgresql/data",
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

func openshiftPostgresqlDeployment(namespace string, name string, database string, user string, password string) *v1.Deployment {
	labels := map[string]string{"app": name}
	var replicas int32 = 1
	var readinessProbe string = "PGPASSWORD=" + password + " /usr/bin/psql -w -U " + user + " -d " + database + " -c 'SELECT 1'"
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      name,
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
							Name:            name,
							Image:           "quay.io/debezium/example-postgres-ocp:latest",
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
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
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "-i", "-c", readinessProbe},
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(5432),
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      1,
							},
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
							TerminationMessagePath:   "/dev/termination-log",
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
				},
			},
		},
	}
}

func postgresqlService(namespace string, name string) *corev1.Service {
	labels := map[string]string{"app": name}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      name,
			Namespace: namespace,
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

func postgresqlPersistentVolumeClaim(namespace string, name string) *corev1.PersistentVolumeClaim {
	labels := map[string]string{"app": name}
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      name,
			Namespace: namespace,
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

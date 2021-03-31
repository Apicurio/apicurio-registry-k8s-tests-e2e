package sql

import (
	"context"
	"net/http"
	"strconv"
	"time"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	apicurioutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio"
	apicurioclient "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/apicurio/client"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/functional"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	kubernetescli "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
	apicurio "github.com/Apicurio/apicurio-registry-operator/api/v2"
)

var artifactData string = "{\"type\":\"record\",\"name\":\"price\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"symbol\",\"type\":\"string\"},{\"name\":\"price\",\"type\":\"string\"}]}"
var dbplaygroundlabels map[string]string = map[string]string{"apicurio": "dbplayground"}

func ExecuteBackupAndRestoreTestCase(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	//deploy db and registry
	backupDBData := DeployPostgresqlDatabase(suiteCtx, ctx.RegistryNamespace, "backupdb", "backupdb", "test", "test")
	ctx.RegisterCleanup(func() {
		RemovePostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, ctx.RegistryNamespace, backupDBData.Name)
	})

	backupregistry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "backupregistry",
		},
		Spec: apicurio.ApicurioRegistrySpec{
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				LogLevel:    "DEBUG",
				Persistence: utils.StorageSql,
				DataSource: apicurio.ApicurioRegistrySpecConfigurationDataSource{
					Url:      string(backupDBData.DataSourceURL),
					UserName: backupDBData.User,
					Password: backupDBData.Password,
				},
			},
		},
	}
	ctx.RegisterCleanup(func() {
		if apicurioutils.ExistsRegistry(suiteCtx, ctx.RegistryNamespace, backupregistry.Name) {
			apicurioutils.DeleteRegistryAndWait(suiteCtx, ctx.RegistryNamespace, backupregistry.Name)
		}
	})
	apicurioutils.CreateRegistryAndWait(suiteCtx, ctx, &backupregistry)
	functional.BasicRegistryAPITest(ctx)

	//create artifacts on the registry
	backupclient := apicurioclient.NewApicurioRegistryApiClient(ctx.RegistryHost, ctx.RegistryPort, http.DefaultClient)

	for i := 1; i <= 50; i++ {
		err := backupclient.CreateArtifact("bandr-"+strconv.Itoa(i), apicurioclient.Avro, artifactData)
		Expect(err).ToNot(HaveOccurred())
		time.Sleep(1 * time.Second)
	}

	artifacts, err := backupclient.ListArtifacts()
	Expect(err).ToNot(HaveOccurred())
	Expect(len(artifacts)).To(BeIdenticalTo(50))

	// deploy a dummypod to create the backup and store it, and then restore the backup from that pod
	log.Info("Deploying dbplayground")

	dbplaygroundImage := "quay.io/rh_integration/service-registry-dbplayground:pg10"
	kubernetescli.Execute("create", "serviceaccount", "dbplayground", "-n", ctx.RegistryNamespace)
	if suiteCtx.IsOpenshift {
		kubernetescli.Execute("adm", "policy", "add-scc-to-user", "privileged", "system:serviceaccount:"+ctx.RegistryNamespace+":dbplayground", "-n", ctx.RegistryNamespace)
	}
	err = suiteCtx.K8sClient.Create(context.TODO(), dbplaygroundDeployment(ctx.RegistryNamespace, dbplaygroundImage))
	Expect(err).ToNot(HaveOccurred())
	kubernetesutils.WaitForDeploymentReady(suiteCtx.Clientset, 120*time.Second, ctx.RegistryNamespace, "dbplayground", 1)
	time.Sleep(2 * time.Second)
	labelsSet := labels.Set(dbplaygroundlabels)
	podList, err := suiteCtx.Clientset.CoreV1().Pods(ctx.RegistryNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	Expect(err).ToNot(HaveOccurred())
	dbplaygroundPodName := podList.Items[0].Name
	ctx.RegisterCleanup(func() {
		suiteCtx.Clientset.AppsV1().Deployments(ctx.RegistryNamespace).Delete(context.TODO(), "dbplayground", metav1.DeleteOptions{})
	})

	// create the backup
	kubernetescli.Execute("-n", ctx.RegistryNamespace, "exec", dbplaygroundPodName, "--", "./create_backup.sh", backupDBData.Host, backupDBData.Port, backupDBData.Database, backupDBData.User, backupDBData.Password)
	log.Info("Backup performed successfully")

	// shut down the registry and the first db
	apicurioutils.DeleteRegistryAndWait(suiteCtx, ctx.RegistryNamespace, backupregistry.Name)
	RemovePostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, ctx.RegistryNamespace, backupDBData.Name)

	// deploy the new db, this deployment already creates the database
	restoreDBData := DeployPostgresqlDatabase(suiteCtx, ctx.RegistryNamespace, "restoredb", "restoredb", "test", "test")
	ctx.RegisterCleanup(func() {
		RemovePostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, ctx.RegistryNamespace, restoreDBData.Name)
	})

	// restore the backup
	kubernetescli.Execute("-n", ctx.RegistryNamespace, "exec", dbplaygroundPodName, "--", "./restore_backup.sh", restoreDBData.Host, restoreDBData.Port, restoreDBData.Database, restoreDBData.User, restoreDBData.Password)
	log.Info("DB restored")

	// deploy registry using restored db
	restoreregistry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctx.RegistryNamespace,
			Name:      "restoreregistry",
		},
		Spec: apicurio.ApicurioRegistrySpec{
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				LogLevel:    "DEBUG",
				Persistence: utils.StorageSql,
				DataSource: apicurio.ApicurioRegistrySpecConfigurationDataSource{
					Url:      string(restoreDBData.DataSourceURL),
					UserName: restoreDBData.User,
					Password: restoreDBData.Password,
				},
			},
		},
	}
	ctx.RegisterCleanup(func() {
		if apicurioutils.ExistsRegistry(suiteCtx, ctx.RegistryNamespace, restoreregistry.Name) {
			apicurioutils.DeleteRegistryAndWait(suiteCtx, ctx.RegistryNamespace, restoreregistry.Name)
		}
	})
	apicurioutils.CreateRegistryAndWait(suiteCtx, ctx, &restoreregistry)
	functional.BasicRegistryAPITest(ctx)

	// verify new registry have old data
	restoreclient := apicurioclient.NewApicurioRegistryApiClient(ctx.RegistryHost, ctx.RegistryPort, http.DefaultClient)
	artifacts, err = restoreclient.ListArtifacts()
	Expect(err).ToNot(HaveOccurred())
	Expect(len(artifacts)).To(BeIdenticalTo(50))

}

func dbplaygroundDeployment(namespace string, image string) *v1.Deployment {
	var replicas int32 = 1
	var privileged bool = true
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dbplayground",
			Namespace: namespace,
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: dbplaygroundlabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: dbplaygroundlabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "dbplayground",
					Containers: []corev1.Container{
						{
							Name:  "dbplayground",
							Image: image,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
						},
					},
				},
			},
		},
	}
}

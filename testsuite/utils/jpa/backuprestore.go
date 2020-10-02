package jpa

import (
	"context"
	"net/http"
	"os"
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
	suite "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/suite"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
	apicurio "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
)

var artifactData string = "{\"type\":\"record\",\"name\":\"price\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"symbol\",\"type\":\"string\"},{\"name\":\"price\",\"type\":\"string\"}]}"
var dbplaygroundlabels map[string]string = map[string]string{"apicurio": "dbplayground"}

func ExecuteBackupAndRestoreTestCase(suiteCtx *suite.SuiteContext, ctx *types.TestContext) {

	//deploy db and registry
	backupDBData := DeployPostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, "backupdb", "backupdb", "test", "test")
	ctx.RegisterCleanup(func() {
		RemovePostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, backupDBData.Name)
	})

	backupregistry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: utils.OperatorNamespace,
			Name:      "backupregistry",
		},
		Spec: apicurio.ApicurioRegistrySpec{
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				LogLevel:    "DEBUG",
				Persistence: utils.StorageJpa,
				DataSource: apicurio.ApicurioRegistrySpecConfigurationDataSource{
					Url:      string(backupDBData.DataSourceURL),
					UserName: backupDBData.User,
					Password: backupDBData.Password,
				},
			},
		},
	}
	apicurioutils.CreateRegistryAndWait(suiteCtx, ctx, &backupregistry)
	functional.BasicRegistryAPITest(ctx)
	ctx.RegisterCleanup(func() {
		if apicurioutils.ExistsRegistry(suiteCtx, backupregistry.Name) {
			apicurioutils.DeleteRegistryAndWait(suiteCtx, backupregistry.Name)
		}
	})

	//create artifacts on the registry
	client := apicurioclient.NewApicurioRegistryApiClient(ctx.RegistryHost, ctx.RegistryPort, http.DefaultClient)

	for i := 1; i <= 50; i++ {
		err := client.CreateArtifact("bandr-"+strconv.Itoa(i), apicurioclient.Avro, artifactData)
		Expect(err).ToNot(HaveOccurred())
		time.Sleep(1 * time.Second)
	}

	artifacts, err := client.ListArtifacts()
	Expect(err).ToNot(HaveOccurred())
	Expect(len(artifacts)).To(BeIdenticalTo(50))

	// deploy a dummypod to create the backup and store it, and then restore the backup from that pod
	log.Info("Deploying dbplayground")

	oldDir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	dbplaygroundBuildDir := utils.SuiteProjectDirValue + "/scripts/dbplayground"
	os.Chdir(dbplaygroundBuildDir)
	err = utils.ExecuteCmd(true, &utils.Command{Cmd: []string{"make", "build", "push"}})
	os.Chdir(oldDir)

	//TODO make this work on openshift
	dbplaygroundImage := "localhost:5000/dbplayground:latest"

	err = suiteCtx.K8sClient.Create(context.TODO(), dbplaygroundDeployment(dbplaygroundImage))
	Expect(err).ToNot(HaveOccurred())
	kubernetesutils.WaitForDeploymentReady(suiteCtx.Clientset, 120*time.Second, "dbplayground", 1)
	time.Sleep(10 * time.Second)
	labelsSet := labels.Set(dbplaygroundlabels)
	podList, err := suiteCtx.Clientset.CoreV1().Pods(utils.OperatorNamespace).List(metav1.ListOptions{LabelSelector: labelsSet.AsSelector().String()})
	Expect(err).ToNot(HaveOccurred())
	dbplaygroundPodName := podList.Items[0].Name

	// create the backup
	kubernetescli.Execute("exec", dbplaygroundPodName, "--", "./create_backup.sh", backupDBData.Host, backupDBData.Port, backupDBData.Database, backupDBData.User, backupDBData.Password)
	log.Info("Backup performed successfully")

	// shut down the registry and the first db
	apicurioutils.DeleteRegistryAndWait(suiteCtx, backupregistry.Name)
	RemovePostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, backupDBData.Name)

	// deploy the new db, this deployment already creates the database
	restoreDBData := DeployPostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, "restoredb", "restoredb", "test2", "test2")
	ctx.RegisterCleanup(func() {
		RemovePostgresqlDatabase(suiteCtx.K8sClient, suiteCtx.Clientset, restoreDBData.Name)
	})

	// restore the backup
	kubernetescli.Execute("exec", dbplaygroundPodName, "--", "./restore_backup.sh", restoreDBData.Host, restoreDBData.Port, restoreDBData.Database, restoreDBData.User, restoreDBData.Password)
	log.Info("DB restored")

	// deploy registry using restored db
	restoreregistry := apicurio.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: utils.OperatorNamespace,
			Name:      "restoreregistry",
		},
		Spec: apicurio.ApicurioRegistrySpec{
			Configuration: apicurio.ApicurioRegistrySpecConfiguration{
				LogLevel:    "DEBUG",
				Persistence: utils.StorageJpa,
				DataSource: apicurio.ApicurioRegistrySpecConfigurationDataSource{
					Url:      string(restoreDBData.DataSourceURL),
					UserName: restoreDBData.User,
					Password: restoreDBData.Password,
				},
			},
		},
	}
	apicurioutils.CreateRegistryAndWait(suiteCtx, ctx, &restoreregistry)
	functional.BasicRegistryAPITest(ctx)
	ctx.RegisterCleanup(func() {
		apicurioutils.DeleteRegistryAndWait(suiteCtx, restoreregistry.Name)
	})

	// verify new registry have old data
	artifacts, err = client.ListArtifacts()
	Expect(err).ToNot(HaveOccurred())
	Expect(len(artifacts)).To(BeIdenticalTo(50))

}

func dbplaygroundDeployment(image string) *v1.Deployment {
	var replicas int32 = 1
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dbplayground",
			Namespace: utils.OperatorNamespace,
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
					Containers: []corev1.Container{
						{
							Name:  "dbplayground",
							Image: image,
						},
					},
				},
			},
		},
	}
}

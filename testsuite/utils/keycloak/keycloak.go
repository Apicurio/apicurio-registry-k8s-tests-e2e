package keycloak

import (
	"context"
	"path/filepath"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	routev1 "github.com/openshift/api/route/v1"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetescli"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/olm"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"

	apicurio "github.com/Apicurio/apicurio-registry-operator/api/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

var log = logf.Log.WithName("keycloak")

var operatorDir string = filepath.Join(utils.SuiteProjectDir, "/keycloak-operator")

const keycloakHttp string = "keycloak-http"

func KeycloakConfigResource(ctx *types.TestContext) apicurio.ApicurioRegistrySpecConfigurationSecurityKeycloak {
	// info hardcoded in kubefiles/keycloak/*.yaml
	return apicurio.ApicurioRegistrySpecConfigurationSecurityKeycloak{
		Url:         ctx.KeycloakURL + "/auth",
		Realm:       "registry",
		ApiClientId: "registry-client-api",
		UiClientId:  "registry-client-ui",
	}
}

func DeployKeycloak(suiteCtx *types.SuiteContext, ctx *types.TestContext) string {

	keycloakSub := installKeycloakOperator(suiteCtx, ctx.RegistryNamespace)
	ctx.KeycloakSubscription = keycloakSub

	log.Info("Deploying keycloak server")
	kubernetescli.Execute("apply", "-f", filepath.Join(utils.SuiteProjectDir, "/kubefiles/keycloak/keycloak.yaml"), "-n", ctx.RegistryNamespace)

	timeout := 13 * time.Minute
	log.Info("Waiting for keycloak server to be ready ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := suiteCtx.Clientset.AppsV1().StatefulSets(ctx.RegistryNamespace).Get(context.TODO(), "keycloak", metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		if od != nil {
			if od.Status.ReadyReplicas > int32(0) {
				return true, nil
			}
		}
		return false, nil
	})
	kubernetescli.GetPods(ctx.RegistryNamespace)
	Expect(err).ToNot(HaveOccurred())

	err = suiteCtx.K8sClient.Create(context.TODO(), keycloakHttpService(ctx.RegistryNamespace))
	Expect(err).ToNot(HaveOccurred())
	if suiteCtx.IsOpenshift {
		_, err = suiteCtx.OcpRouteClient.Routes(ctx.RegistryNamespace).Create(context.TODO(), ocpKeycloakRoute(ctx.RegistryNamespace), metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	} else {
		err = suiteCtx.K8sClient.Create(context.TODO(), kindKeycloakIngress(ctx.RegistryNamespace))
		Expect(err).ToNot(HaveOccurred())
	}

	keycloakURL := "http://example-keycloak.127.0.0.1.nip.io:80"
	if suiteCtx.IsOpenshift {
		kubernetescli.Execute("get", "route", "-n", ctx.RegistryNamespace)

		timeout := 90 * time.Second
		log.Info("Waiting for keycloak route to be ready", "timeout", timeout)
		err = wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
			route, err := suiteCtx.OcpRouteClient.Routes(ctx.RegistryNamespace).Get(context.TODO(), keycloakHttp, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			return len(route.Status.Ingress) != 0, nil
		})

		httpRoute, err := suiteCtx.OcpRouteClient.Routes(ctx.RegistryNamespace).Get(context.TODO(), keycloakHttp, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		keycloakURL = "http://" + httpRoute.Status.Ingress[0].Host
	}

	log.Info("Creating keycloak realm")
	kubernetescli.Execute("apply", "-f", filepath.Join(utils.SuiteProjectDir, "/kubefiles/keycloak/keycloak-realm.yaml"), "-n", ctx.RegistryNamespace)

	return keycloakURL
}

func RemoveKeycloak(suiteCtx *types.SuiteContext, ctx *types.TestContext) {

	log.Info("Removing keycloak realm")
	kubernetescli.Execute("delete", "-f", filepath.Join(utils.SuiteProjectDir, "/kubefiles/keycloak/keycloak-realm.yaml"), "-n", ctx.RegistryNamespace)

	log.Info("Removing keycloak server")
	kubernetescli.Execute("delete", "-f", filepath.Join(utils.SuiteProjectDir, "/kubefiles/keycloak/keycloak.yaml"), "-n", ctx.RegistryNamespace)

	timeout := 3 * time.Minute
	log.Info("Waiting for keycloak server to be deleted ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := suiteCtx.Clientset.AppsV1().StatefulSets(ctx.RegistryNamespace).Get(context.TODO(), "keycloak", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	kubernetescli.GetPods(ctx.RegistryNamespace)
	Expect(err).ToNot(HaveOccurred())

	log.Info("Removing keycloak networking")
	err = suiteCtx.K8sClient.Delete(context.TODO(), keycloakHttpService(ctx.RegistryNamespace))
	Expect(err).ToNot(HaveOccurred())
	if suiteCtx.IsOpenshift {
		err = suiteCtx.OcpRouteClient.Routes(ctx.RegistryNamespace).Delete(context.TODO(), keycloakHttp, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	} else {
		err = suiteCtx.K8sClient.Delete(context.TODO(), kindKeycloakIngress(ctx.RegistryNamespace))
		Expect(err).ToNot(HaveOccurred())
	}

	removeKeycloakOperator(suiteCtx, ctx.RegistryNamespace, ctx.KeycloakSubscription)
}

func installKeycloakOperator(suiteCtx *types.SuiteContext, namespace string) *operatorsv1alpha1.Subscription {

	var operatorGroupName string = namespace + "-operator-group"

	olm.CreateOperatorGroup(suiteCtx, namespace, operatorGroupName)

	subreq := &olm.CreateSubscriptionRequest{
		SubscriptionNamespace:  namespace,
		SubscriptionName:       "keycloak-operator",
		Package:                "keycloak-operator",
		CatalogSourceName:      "operatorhubio-catalog",
		CatalogSourceNamespace: "olm",
		ChannelName:            "alpha",
		ChannelCSV:             "keycloak-operator.v12.0.3",
	}
	if suiteCtx.IsOpenshift {
		subreq.CatalogSourceName = "community-operators"
		subreq.CatalogSourceNamespace = "openshift-marketplace"
	}

	sub := olm.CreateSubscription(suiteCtx, subreq)

	// // https://github.com/keycloak/keycloak-operator.git
	// if _, err := os.Stat(operatorDir); !os.IsNotExist(err) {
	// 	currentDir, err := os.Getwd()
	// 	if err != nil {
	// 		Expect(err).ToNot(HaveOccurred())
	// 	}
	// 	os.Chdir(operatorDir)
	// 	utils.ExecuteCmdOrDie(true, "git", "pull")
	// 	os.Chdir(currentDir)
	// } else {
	// 	utils.ExecuteCmdOrDie(true, "git", "clone", "https://github.com/keycloak/keycloak-operator.git", operatorDir)
	// }

	// kubernetescli.Execute("apply", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/crds/"))
	// kubernetescli.Execute("apply", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/role.yaml"))
	// kubernetescli.Execute("apply", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/role_binding.yaml"))
	// kubernetescli.Execute("apply", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/service_account.yaml"))
	// kubernetescli.Execute("apply", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/operator.yaml"))

	timeout := 120 * time.Second
	log.Info("Waiting for keycloak operator to be ready ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		od, err := suiteCtx.Clientset.AppsV1().Deployments(namespace).Get(context.TODO(), "keycloak-operator", metav1.GetOptions{})
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

	return sub
}

func removeKeycloakOperator(suiteCtx *types.SuiteContext, namespace string, sub *operatorsv1alpha1.Subscription) {

	olm.DeleteSubscription(suiteCtx, sub, false)

	var operatorGroupName string = namespace + "-operator-group"
	olm.DeleteOperatorGroup(suiteCtx, namespace, operatorGroupName)

	// kubernetescli.Execute("delete", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/operator.yaml"))
	// kubernetescli.Execute("delete", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/service_account.yaml"))
	// kubernetescli.Execute("delete", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/role_binding.yaml"))
	// kubernetescli.Execute("delete", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/role.yaml"))
	// kubernetescli.Execute("delete", "-n", namespace, "-f", filepath.Join(operatorDir, "deploy/crds/"))

	timeout := 120 * time.Second
	log.Info("Waiting for keycloak operator to be removed ", "timeout", timeout)
	err := wait.Poll(utils.APIPollInterval, timeout, func() (bool, error) {
		_, err := suiteCtx.Clientset.AppsV1().Deployments(namespace).Get(context.TODO(), "keycloak-operator", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	kubernetescli.GetPods(namespace)
	Expect(err).ToNot(HaveOccurred())
}

var labels map[string]string = map[string]string{"app": "keycloak", "component": "keycloak"}

func keycloakHttpService(namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakHttp,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func kindKeycloakIngress(namespace string) *networking.Ingress {
	pathTypePrefix := networking.PathTypePrefix
	return &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakHttp,
			Namespace: namespace,
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "example-keycloak.127.0.0.1.nip.io",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: keycloakHttp,
											Port: networking.ServiceBackendPort{
												Number: 8080,
											},
										},
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

func ocpKeycloakRoute(namespace string) *routev1.Route {
	var weigh int32 = 100
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakHttp,
			Namespace: namespace,
		},
		Spec: routev1.RouteSpec{
			Path: "/",
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   keycloakHttp,
				Weight: &weigh,
			},
		},
	}
}

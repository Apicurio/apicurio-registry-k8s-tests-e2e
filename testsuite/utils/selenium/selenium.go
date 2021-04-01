package selenium

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	routev1 "github.com/openshift/api/route/v1"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils"
	kubernetesutils "github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/kubernetes"
	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

var log = logf.Log.WithName("selenium")

var labels map[string]string = map[string]string{"app": "selenium-chrome"}
var seleniumName string = "selenium-chrome"
var seleniumNamespace string = "selenium"

func DeploySeleniumIfNeeded(suiteCtx *types.SuiteContext) {
	if isSeleniumNeeded(suiteCtx) {
		deploySeleniumChrome(suiteCtx)
	}
}

func RemoveSeleniumIfNeeded(suiteCtx *types.SuiteContext) {
	if isSeleniumNeeded(suiteCtx) {
		removeSeleniumChrome(suiteCtx)
	}
}

func isSeleniumNeeded(suiteCtx *types.SuiteContext) bool {
	return suiteCtx.SetupSelenium || (utils.ApicurioTestsProfile != "" && (utils.ApicurioTestsProfile == "ui" || utils.ApicurioTestsProfile == "all" || utils.ApicurioTestsProfile == "acceptance"))
}

func deploySeleniumChrome(suiteCtx *types.SuiteContext) {
	log.Info("Deploying selenium")

	kubernetesutils.CreateTestNamespace(suiteCtx.Clientset, seleniumNamespace)

	err := suiteCtx.K8sClient.Create(context.TODO(), seleniumDeployment(seleniumNamespace))
	Expect(err).ToNot(HaveOccurred())

	err = suiteCtx.K8sClient.Create(context.TODO(), seleniumService(seleniumNamespace))
	Expect(err).ToNot(HaveOccurred())

	if suiteCtx.IsOpenshift {
		_, err = suiteCtx.OcpRouteClient.Routes(seleniumNamespace).Create(context.TODO(), ocpSeleniumRoute(seleniumNamespace), metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	} else {
		kubernetesutils.WaitForDeploymentReady(suiteCtx.Clientset, 150*time.Second, "ingress-nginx", "ingress-nginx-controller", 1)

		err = suiteCtx.K8sClient.Create(context.TODO(), seleniumIngress(seleniumNamespace))
		Expect(err).ToNot(HaveOccurred())
	}

	kubernetesutils.WaitForDeploymentReady(suiteCtx.Clientset, 180*time.Second, seleniumNamespace, seleniumName, 1)

	if suiteCtx.IsOpenshift {
		seleniumRoute, err := suiteCtx.OcpRouteClient.Routes(seleniumNamespace).Get(context.TODO(), seleniumName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		suiteCtx.SeleniumHost = seleniumRoute.Status.Ingress[0].Host
		suiteCtx.SeleniumPort = "80"
	} else {
		suiteCtx.SeleniumHost = "selenium-chrome.127.0.0.1.nip.io"
		suiteCtx.SeleniumPort = "80"
	}
}

func removeSeleniumChrome(suiteCtx *types.SuiteContext) {
	log.Info("Removing selenium")
	err := suiteCtx.K8sClient.Delete(context.TODO(), seleniumDeployment(seleniumNamespace))
	Expect(err).ToNot(HaveOccurred())
	err = suiteCtx.K8sClient.Delete(context.TODO(), seleniumService(seleniumNamespace))
	Expect(err).ToNot(HaveOccurred())
	if suiteCtx.IsOpenshift {
		err = suiteCtx.OcpRouteClient.Routes(seleniumNamespace).Delete(context.TODO(), seleniumName, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	} else {
		err = suiteCtx.K8sClient.Delete(context.TODO(), seleniumIngress(seleniumNamespace))
		Expect(err).ToNot(HaveOccurred())
	}
	kubernetesutils.DeleteTestNamespace(suiteCtx.Clientset, seleniumNamespace)
}

func seleniumDeployment(namespace string) *v1.Deployment {
	var replicas int32 = 1
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      seleniumName,
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
							Name:  seleniumName,
							Image: "selenium/standalone-chrome",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 4444,
									Name:          "http",
									Protocol:      "TCP",
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/wd/hub",
										Port: intstr.FromInt(4444),
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       2,
							},
							// Resources: corev1.ResourceRequirements{
							// 	Requests: corev1.ResourceList{
							// 		"requests": resource.Quantity{

							// 		}
							// 	},
							// },
						},
					},
				},
			},
		},
	}
}

func seleniumService(namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      seleniumName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     4444,
					Protocol: "TCP",
					Name:     "http",
				},
			},
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func seleniumIngress(namespace string) *networking.Ingress {
	pathTypePrefix := networking.PathTypePrefix
	return &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      seleniumName,
			Namespace: namespace,
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "selenium-chrome.127.0.0.1.nip.io",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									PathType: &pathTypePrefix,
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: seleniumName,
											Port: networking.ServiceBackendPort{
												Number: 4444,
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

func ocpSeleniumRoute(namespace string) *routev1.Route {
	var weigh int32 = 100
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      seleniumName,
			Namespace: namespace,
		},
		Spec: routev1.RouteSpec{
			Path: "/",
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   seleniumName,
				Weight: &weigh,
			},
		},
	}
}

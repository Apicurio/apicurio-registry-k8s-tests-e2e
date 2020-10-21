package openshift

import (
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Apicurio/apicurio-registry-k8s-tests-e2e/testsuite/utils/types"
)

func OcpInternalImage(ctx *types.SuiteContext, namespace string, imageName string, tag string) *types.OcpImageReference {
	ocpImageRegistryRoute, err := ctx.OcpRouteClient.Routes("openshift-image-registry").Get("default-route", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	ocpImageRegistryHost := ocpImageRegistryRoute.Status.Ingress[0].Host

	return &types.OcpImageReference{
		ExternalImage: ocpImageRegistryHost + "/" + namespace + "/" + imageName + ":" + tag,
		InternalImage: "image-registry.openshift-image-registry.svc:5000" + "/" + namespace + "/" + imageName + ":" + tag,
	}
}

package capability

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	agutil "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	testComponentFeature = "test-component-feature"
	testNamespace        = "test-namespace"
	testName             = "test-name"
	testAPIURL           = "https://demo.dev.dynatracelabs.com/api"
)

func createTestDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
		},
	}
}

func TestCreateService(t *testing.T) {
	agHTTPSPort := corev1.ServicePort{
		Name:       consts.HTTPSServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       consts.HTTPSServicePort,
		TargetPort: intstr.FromString(consts.HTTPSServicePortName),
	}
	agHTTPPort := corev1.ServicePort{
		Name:       consts.HTTPServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       consts.HTTPServicePort,
		TargetPort: intstr.FromString(consts.HTTPServicePortName),
	}

	t.Run("check service name, labels and selector", func(t *testing.T) {
		dk := createTestDynaKube()
		service := CreateService(dk)

		assert.NotNil(t, service)
		assert.Equal(t, dk.Name+"-"+consts.MultiActiveGateName, service.Name)
		assert.Equal(t, dk.Namespace, service.Namespace)

		expectedLabels := map[string]string{
			labels.AppCreatedByLabel: testName,
			labels.AppComponentLabel: labels.ActiveGateComponentLabel,
			labels.AppNameLabel:      version.AppName,
			labels.AppVersionLabel:   version.Version,
		}
		assert.Equal(t, expectedLabels, service.Labels)

		expectedSelector := map[string]string{
			labels.AppCreatedByLabel: testName,
			labels.AppManagedByLabel: version.AppName,
			labels.AppNameLabel:      labels.ActiveGateComponentLabel,
		}
		serviceSpec := service.Spec
		assert.Equal(t, corev1.ServiceTypeClusterIP, serviceSpec.Type)
		assert.Equal(t, expectedSelector, serviceSpec.Selector)
	})

	t.Run("check AG service if metrics-ingest disabled", func(t *testing.T) {
		dk := createTestDynaKube()
		agutil.SwitchCapability(dk, activegate.RoutingCapability, true)

		service := CreateService(dk)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHTTPSPort)
		assert.Contains(t, ports, agHTTPPort)
	})
	t.Run("check AG service if metrics-ingest enabled", func(t *testing.T) {
		dk := createTestDynaKube()
		agutil.SwitchCapability(dk, activegate.MetricsIngestCapability, true)

		service := CreateService(dk)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHTTPSPort)
		assert.Contains(t, ports, agHTTPPort)
	})
}

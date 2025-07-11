//go:build e2e

package supportarchive

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/cmd/supportarchive"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/functional"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	e2ewebhook "github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/event"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/replicaset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/service"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

const (
	diagExecutorLogFile = "var/lib/dynatrace/remotepluginmodule/log/extensions/diagnostics/diag_executor.log"
	lsLogFile           = "ls.txt"
)

type requiredFiles struct {
	t              *testing.T
	ctx            context.Context
	resources      *resources.Resources
	dk             dynakube.DynaKube
	ec             edgeconnect.EdgeConnect
	collectManaged bool
}

func newRequiredFiles(t *testing.T, ctx context.Context, resources *resources.Resources, customResources CustomResources, collectManaged bool) requiredFiles {
	return requiredFiles{
		t:              t,
		ctx:            ctx,
		resources:      resources,
		dk:             customResources.dk,
		ec:             customResources.ec,
		collectManaged: collectManaged,
	}
}

func (r requiredFiles) collectRequiredFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles, supportarchive.OperatorVersionFileName)
	requiredFiles = append(requiredFiles, supportarchive.TroublshootOutputFileName)
	requiredFiles = append(requiredFiles, supportarchive.SupportArchiveOutputFileName)
	requiredFiles = append(requiredFiles, r.getRequiredPodFiles(labels.AppNameLabel, true)...)
	requiredFiles = append(requiredFiles, r.getRequiredPodFiles(labels.AppManagedByLabel, r.collectManaged)...)
	requiredFiles = append(requiredFiles, r.getRequiredPodDiagnosticLogFiles(r.collectManaged)...)
	requiredFiles = append(requiredFiles, r.getRequiredReplicaSetFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredServiceFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredWorkloadFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredNamespaceFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredDynaKubeFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredEdgeConnectFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredStatefulSetFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredDaemonSetFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredWebhookConfigurationFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredCRDFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredConfigMapFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredEventFiles()...)

	return requiredFiles
}

func (r requiredFiles) getRequiredPodFiles(labelKey string, collectManaged bool) []string {
	pods := pod.List(r.t, r.ctx, r.resources, r.dk.Namespace)
	requiredFiles := make([]string, 0)

	podList := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		label, ok := podItem.Labels[labelKey]

		return ok && label == operator.DeploymentName
	})

	for _, operatorPod := range podList {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/pod/%s%s",
				supportarchive.ManifestsDirectoryName,
				operatorPod.Namespace, operatorPod.Name,
				supportarchive.ManifestsFileExtension))
		if collectManaged && (labelKey == "app.kubernetes.io/managed-by" || labelKey == "app.kubernetes.io/name") {
			for _, container := range operatorPod.Spec.Containers {
				requiredFiles = append(requiredFiles,
					fmt.Sprintf("%s/%s/%s.log", supportarchive.LogsDirectoryName, operatorPod.Name, container.Name))
			}
		}
	}

	return requiredFiles
}

func (r requiredFiles) getRequiredPodDiagnosticLogFiles(collectManaged bool) []string {
	requiredFiles := make([]string, 0)

	if !collectManaged {
		return requiredFiles
	}

	pods := pod.List(r.t, r.ctx, r.resources, r.dk.Namespace)

	podList := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		appNamelabel, okAppNamelabel := podItem.Labels[labels.AppNameLabel]
		appManagedByLabel, okAppManagedByLabel := podItem.Labels[labels.AppManagedByLabel]

		return okAppNamelabel && appNamelabel == supportarchive.LabelEecPodName && okAppManagedByLabel && appManagedByLabel == operator.DeploymentName
	})

	for _, pod := range podList {
		requiredFiles = append(requiredFiles, supportarchive.BuildZipFilePath(pod.Name, diagExecutorLogFile))
		requiredFiles = append(requiredFiles, supportarchive.BuildZipFilePath(pod.Name, lsLogFile))
	}

	return requiredFiles
}

func (r requiredFiles) getRequiredReplicaSetFiles() []string {
	replicaSets := replicaset.List(r.t, r.ctx, r.resources, r.dk.Namespace)
	requiredFiles := make([]string, 0)
	for _, replicaSet := range replicaSets.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/replicaset/%s%s",
				supportarchive.ManifestsDirectoryName,
				replicaSet.Namespace, replicaSet.Name,
				supportarchive.ManifestsFileExtension))
	}

	return requiredFiles
}

func (r requiredFiles) getRequiredStatefulSetFiles() []string {
	statefulSet, err := statefulset.NewQuery(r.ctx, r.resources, client.ObjectKey{
		Namespace: r.dk.Namespace,
		Name:      "dynakube-activegate"}).Get()
	require.NoError(r.t, err)
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/statefulset/%s%s",
			supportarchive.ManifestsDirectoryName,
			statefulSet.Namespace, statefulSet.Name,
			supportarchive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredDaemonSetFiles() []string {
	oneagentDaemonSet, err := oneagent.Get(r.ctx, r.resources, r.dk)
	require.NoError(r.t, err)
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/daemonset/%s%s",
			supportarchive.ManifestsDirectoryName,
			oneagentDaemonSet.Namespace,
			oneagentDaemonSet.Name,
			supportarchive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredServiceFiles() []string {
	services := service.List(r.t, r.ctx, r.resources, r.dk.Namespace)
	requiredFiles := make([]string, 0)
	for _, requiredService := range services.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/service/%s%s",
				supportarchive.ManifestsDirectoryName,
				requiredService.Namespace,
				requiredService.Name,
				supportarchive.ManifestsFileExtension))
	}

	return requiredFiles
}

func (r requiredFiles) getRequiredWorkloadFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			r.dk.Namespace,
			"deployment",
			operator.DeploymentName,
			supportarchive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			r.dk.Namespace,
			"deployment",
			e2ewebhook.DeploymentName,
			supportarchive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			r.dk.Namespace,
			"daemonset",
			csi.DaemonSetName,
			supportarchive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			r.ec.Namespace,
			"deployment",
			r.ec.Name,
			supportarchive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredNamespaceFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/namespace-%s%s",
			supportarchive.ManifestsDirectoryName,
			r.dk.Namespace,
			r.dk.Namespace,
			supportarchive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/namespace-%s%s",
			supportarchive.ManifestsDirectoryName,
			supportarchive.InjectedNamespacesManifestsDirectoryName,
			testAppNameInjected,
			supportarchive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredDynaKubeFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			r.dk.Namespace,
			"dynakube",
			r.dk.Name,
			supportarchive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredEdgeConnectFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			r.ec.Namespace,
			"edgeconnect",
			r.ec.Name,
			supportarchive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredWebhookConfigurationFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			supportarchive.WebhookConfigurationsDirectoryName,
			strings.ToLower(supportarchive.MutatingWebhookConfigurationKind),
			supportarchive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			supportarchive.WebhookConfigurationsDirectoryName,
			strings.ToLower(supportarchive.ValidatingWebhookConfigurationKind),
			supportarchive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredCRDFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			supportarchive.CRDDirectoryName,
			strings.Join([]string{strings.ToLower(supportarchive.CRDKindName), "dynakubes"}, "-"),
			supportarchive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			supportarchive.CRDDirectoryName,
			strings.Join([]string{strings.ToLower(supportarchive.CRDKindName), "edgeconnects"}, "-"),
			supportarchive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredConfigMapFiles() []string {
	requiredFiles := make([]string, 0)

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			supportarchive.ManifestsDirectoryName,
			r.dk.Namespace,
			"configmap",
			"kube-root-ca.crt",
			supportarchive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s-%s%s",
			supportarchive.ManifestsDirectoryName,
			r.dk.Namespace,
			"configmap",
			r.dk.Name,
			"deployment-metadata",
			supportarchive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s-%s%s",
			supportarchive.ManifestsDirectoryName,
			r.dk.Namespace,
			"configmap",
			r.dk.Name,
			"oneagent-connection-info",
			supportarchive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s-%s%s",
			supportarchive.ManifestsDirectoryName,
			r.dk.Namespace,
			"configmap",
			r.dk.Name,
			"activegate-connection-info",
			supportarchive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredEventFiles() []string {
	optFunc := func(options *metav1.ListOptions) {
		options.Limit = int64(supportarchive.NumEventsFlagValue)
		options.FieldSelector = fmt.Sprint(supportarchive.DefaultEventFieldSelector)
	}
	events := event.List(r.t, r.ctx, r.resources, r.dk.Namespace, optFunc)
	requiredFiles := make([]string, 0)

	for _, requiredEvent := range events.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/%s/%s%s",
				supportarchive.ManifestsDirectoryName,
				requiredEvent.Namespace,
				"event",
				requiredEvent.Name,
				supportarchive.ManifestsFileExtension))
	}

	return requiredFiles
}

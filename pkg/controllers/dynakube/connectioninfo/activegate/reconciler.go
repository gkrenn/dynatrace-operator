package activegate

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client       client.Client
	apiReader    client.Reader
	dtc          dtclient.Client
	timeProvider *timeprovider.Provider

	dk *dynakube.DynaKube
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:       clt,
		apiReader:    apiReader,
		dk:           dk,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.ActiveGate().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), activeGateConnectionInfoConditionType) == nil {
			return nil
		}

		r.dk.Status.ActiveGate.ConnectionInfo = communication.ConnectionInfo{}
		query := k8ssecret.Query(r.client, r.apiReader, log)

		err := query.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: r.dk.ActiveGate().GetTenantSecretName(), Namespace: r.dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up ActiveGate tenant-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), activeGateConnectionInfoConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	err := r.reconcileConnectionInfo(ctx)

	if err != nil {
		return err
	}

	return err
}

func (r *reconciler) reconcileConnectionInfo(ctx context.Context) error {
	secretNamespacedName := types.NamespacedName{Name: r.dk.ActiveGate().GetTenantSecretName(), Namespace: r.dk.Namespace}

	if !conditions.IsOutdated(r.timeProvider, r.dk, activeGateConnectionInfoConditionType) {
		isSecretPresent, err := connectioninfo.IsTenantSecretPresent(ctx, r.apiReader, secretNamespacedName, log)
		if err != nil {
			return err
		}

		condition := meta.FindStatusCondition(*r.dk.Conditions(), activeGateConnectionInfoConditionType)
		if isSecretPresent {
			log.Info(dynakube.GetCacheValidMessage(
				"activegate connection info update",
				condition.LastTransitionTime,
				r.dk.APIRequestThreshold()))

			return nil
		}
	}

	conditions.SetSecretOutdated(r.dk.Conditions(), activeGateConnectionInfoConditionType, secretNamespacedName.Name+" is not present or outdated, update in progress") // Necessary to update the LastTransitionTime, also it is a nice failsafe

	connectionInfo, err := r.dtc.GetActiveGateConnectionInfo(ctx)
	if err != nil {
		conditions.SetDynatraceAPIError(r.dk.Conditions(), activeGateConnectionInfoConditionType, err)

		return errors.WithMessage(err, "failed to get ActiveGate connection info")
	}

	r.setDynakubeStatus(connectionInfo)

	if len(connectionInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints", "tenant", connectionInfo.TenantUUID)
	}

	err = r.createTenantTokenSecret(ctx, r.dk.ActiveGate().GetTenantSecretName(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	r.dk.Status.ActiveGate.ConnectionInfo.TenantTokenHash, err = hasher.GenerateHash(connectionInfo.TenantToken)
	if err != nil {
		return errors.Wrap(err, "failed to generate TenantTokenHash")
	}

	log.Info("activegate connection info updated")

	return nil
}

func (r *reconciler) setDynakubeStatus(connectionInfo dtclient.ActiveGateConnectionInfo) {
	r.dk.Status.ActiveGate.ConnectionInfo.TenantUUID = connectionInfo.TenantUUID
	r.dk.Status.ActiveGate.ConnectionInfo.Endpoints = connectionInfo.Endpoints
}

func (r *reconciler) createTenantTokenSecret(ctx context.Context, secretName string, connectionInfo dtclient.ConnectionInfo) error {
	secret, err := connectioninfo.BuildTenantSecret(r.dk, secretName, connectionInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	query := k8ssecret.Query(r.client, r.apiReader, log)

	_, err = query.CreateOrUpdate(ctx, secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		conditions.SetKubeAPIError(r.dk.Conditions(), activeGateConnectionInfoConditionType, err)

		return err
	}

	conditions.SetSecretCreated(r.dk.Conditions(), activeGateConnectionInfoConditionType, secret.Name)

	return nil
}

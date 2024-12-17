package webhook

import (
	"context"
	"encoding/json"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/dana-team/env-route-ns-mutator/internal/utils"
	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// RouteMutator is the struct used to mutate Routes
type RouteMutator struct {
	Decoder admission.Decoder
	Client  client.Client
}

// +kubebuilder:rbac:groups="route.openshift.io",resources=routes,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="config.openshift.io",resources=ingresses,verbs=get;list;watch

// +kubebuilder:webhook:path=/mutate-v1-route,mutating=true,failurePolicy=ignore,sideEffects=None,groups=route.openshift.io,resources=routes,verbs=create,versions=v1,name=route.dana.io,admissionReviewVersions=v1;v1beta1

func (r *RouteMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithName("Route").WithValues("name", req.Name)
	logger.Info("webhook request received")

	route := routev1.Route{}
	if err := r.Decoder.Decode(req, &route); err != nil {
		logger.Error(err, "failed to decode route object")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	namespace := corev1.Namespace{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: req.Namespace}, &namespace); err != nil {
		logger.Error(err, "failed to get namespace object")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	clusterIngress, err := utils.GetClusterIngressDomain(ctx, r.Client)
	if err != nil {
		logger.Error(err, "failed to get cluster ingress")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	environments := utils.GetEnvironments()
	r.handleInner(logger, &route, clusterIngress, environments, namespace.ObjectMeta.Labels)

	marshaledRoute, err := json.Marshal(route)
	if err != nil {
		admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledRoute)
}

// handleInner implements the main mutating logic. It modifies the host of an OpenShift Route
// based on environment data and cluster ingress information.
func (r *RouteMutator) handleInner(logger logr.Logger, route *routev1.Route, clusterIngress string, environments []string, labels map[string]string) {
	if utils.CheckBypass(labels) {
		logger.Info("Bypassing mutation")
		return
	}
	for _, env := range environments {
		if labels[utils.Key] == env {
			routeHost := utils.ModifyHostname(logger, route.Name, route.Namespace, route.Spec.Host, env, clusterIngress)
			route.Spec.Host = routeHost
			break
		}
	}
}

/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	apisv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	"github.com/kcp-dev/logicalcluster/v2"
	appstudiov1alpha1 "github.com/redhat-appstudio/api-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApiManagerReconciler reconciles a ApiManager object
type ApiManagerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=apimanagers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=apimanagers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=apimanagers/finalizers,verbs=update

//+kubebuilder:rbac:groups=apis.kcp.dev,resources=apibindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apis.kcp.dev,resources=apibindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apis.kcp.dev,resources=apibindings/finalizers,verbs=update

// Reconcile creates/updates apibidings in users workspaces.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ApiManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Add the logical cluster to the context
	ctx = logicalcluster.WithCluster(ctx, logicalcluster.New(req.ClusterName))

	logger.Info("Getting apimanager ...")
	var w appstudiov1alpha1.ApiManager
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		if errors.IsNotFound(err) {
			// Normal - was deleted
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}
	logger.Info("apimanger found: ", "object", w)

	// TODO: add WRC library integration

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: understand how to properly watch for apibidings
	return ctrl.NewControllerManagedBy(mgr).
		For(&apisv1alpha1.APIBinding{}).
		Complete(r)
}

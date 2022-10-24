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
	"io/ioutil"
	"path/filepath"

	"github.com/go-logr/logr"
	apisv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	"github.com/kcp-dev/logicalcluster/v2"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TempAPIBinding maps the template of an APIBinding to Go structs.
type TempAPIBinding struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Reference struct {
			Workspace struct {
				Path       string `yaml:"path"`
				ExportName string `yaml:"exportName"`
			} `yaml:"workspace"`
		} `yaml:"reference"`
	} `yaml:"spec"`
}

// APIManagerReconciler reconciles a APIManager object
type APIManagerReconciler struct {
	client.Client
	*rest.Config
	APIExportName   string
	SPWorkspacePath string
	ChartPath       string
	logger          logr.Logger
}

//+kubebuilder:rbac:groups=apis.kcp.dev,resources=apibindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apis.kcp.dev,resources=apibindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apis.kcp.dev,resources=apibindings/finalizers,verbs=update

// Reconcile creates/updates apibidings in users workspaces.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *APIManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx)

	// if we're running on kcp, we need to include workspace in context
	if req.ClusterName != "" {
		ctx = logicalcluster.WithCluster(ctx, logicalcluster.New(req.ClusterName))
	}

	r.logger.Info("Getting APIBinding ...")
	// Getting current workspace in order to calculate SP workspaces
	var apiBinding apisv1alpha1.APIBinding
	if err := r.Get(ctx, req.NamespacedName, &apiBinding); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Error(err, "unable to get APIBinding")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	currentWorkspacePath := apiBinding.Spec.Reference.Workspace.Path
	r.logger.Info("apibinding workspace: ", "workspace path", currentWorkspacePath)
	if apiBinding.Spec.Reference.Workspace.ExportName == r.APIExportName {
		r.logger.Info("apibinding matches APIExport name")

		spAPIExportPath := currentWorkspacePath
		if r.SPWorkspacePath != "" {
			spAPIExportPath = r.SPWorkspacePath
			r.logger.Info("forcing apiexport ws path", "path", spAPIExportPath)
		}

		// read all templates from /workspace/chart/templates folder
		templatesPath := filepath.Join(r.ChartPath, "templates")
		apibindingTemplates, err := ioutil.ReadDir(templatesPath)
		if err != nil {
			r.logger.Error(err, "unable to read charts dir content: "+r.ChartPath)
			return ctrl.Result{}, err
		}

		for _, templateFile := range apibindingTemplates {
			// skip dirs
			if templateFile.IsDir() {
				continue
			}

			err, tempAPIBinding := r.getAPIBindingFromTemplate(filepath.Join(templatesPath, templateFile.Name()))
			apiBindingResource := apisv1alpha1.APIBinding{
				TypeMeta:   metav1.TypeMeta{Kind: tempAPIBinding.Kind, APIVersion: tempAPIBinding.APIVersion},
				ObjectMeta: metav1.ObjectMeta{Name: tempAPIBinding.Metadata.Name, Namespace: req.Namespace},
				Spec: apisv1alpha1.APIBindingSpec{Reference: apisv1alpha1.ExportReference{Workspace: &apisv1alpha1.WorkspaceExportReference{
					Path:       spAPIExportPath,
					ExportName: tempAPIBinding.Spec.Reference.Workspace.ExportName,
				}}},
			}

			// Check if apibinding exists, if not create it.
			found := &apisv1alpha1.APIBinding{}
			err = r.Get(ctx, types.NamespacedName{Name: apiBindingResource.Name, Namespace: apiBindingResource.Namespace}, found)
			if err != nil {
				if errors.IsNotFound(err) {
					// Define and create a new apibinding.
					r.logger.Info("going to create apibindings", "workspace path", spAPIExportPath, "chart path", r.ChartPath)
					err := r.Create(ctx, &apiBindingResource)
					if err != nil {
						r.logger.Error(err, "unable to create apibiding: "+apiBindingResource.ObjectMeta.Name)
						return ctrl.Result{}, err
					}
					r.logger.Info("apibinding created.", "binding name", apiBindingResource.ObjectMeta.Name)
					return ctrl.Result{Requeue: true}, nil
				} else {
					return ctrl.Result{}, err
				}
			}

			// accept all permission claims in all APIBindings
			err = r.acceptAllPermissionClaims(ctx, req, apiBindingResource.ObjectMeta.Name)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

	}

	return ctrl.Result{}, nil
}

// getAPIBindingFromTemplate returns an APIBinding object from a YAML template file.
func (r *APIManagerReconciler) getAPIBindingFromTemplate(templateFilePath string) (error, TempAPIBinding) {
	// unmarshal apibindings template to objects
	templateContent, err := ioutil.ReadFile(templateFilePath)
	if err != nil {
		r.logger.Error(err, "unable to read file: "+templateFilePath)
	}
	applicationServiceBinding := TempAPIBinding{}
	err = yaml.Unmarshal(templateContent, &applicationServiceBinding)
	if err != nil {
		r.logger.Error(err, "unmarshal error")
	}
	return err, applicationServiceBinding
}

// acceptAllPermissionClaims, reads all the required permissions from the APIBinding resource and patches the permission claims field.
func (r *APIManagerReconciler) acceptAllPermissionClaims(ctx context.Context, req ctrl.Request, apiBindingName string) error {
	SPAPIBinding := &apisv1alpha1.APIBinding{}
	err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: apiBindingName}, SPAPIBinding)
	if err != nil {
		r.logger.Error(err, "unable to find apibiding: "+apiBindingName)
		return err
	}
	r.logger.Info("apibinding resource found", "resource", SPAPIBinding)

	r.logger.Info("patching apibindings to accept all permission claims", "bindings name", apiBindingName)
	patchApiBiding := client.MergeFrom(SPAPIBinding.DeepCopy())

	var permissionClaims []apisv1alpha1.AcceptablePermissionClaim
	for _, exportClaim := range SPAPIBinding.Status.ExportPermissionClaims {
		permissionClaims = append(permissionClaims, apisv1alpha1.AcceptablePermissionClaim{
			PermissionClaim: exportClaim,
			State:           "Accepted",
		})
	}
	r.logger.Info("following list of permission claims will be used for the patch", "permissionClaims", permissionClaims)
	SPAPIBinding.Spec.PermissionClaims = permissionClaims

	err = r.Patch(ctx, SPAPIBinding, patchApiBiding)
	if err != nil {
		r.logger.Error(err, "unable to patch apibidings.", "apibindings name", apiBindingName, "permission claim", SPAPIBinding.Spec.PermissionClaims)
		return err
	}
	r.logger.Info("permission claim for apibindings accepted", "apibindings name", apiBindingName, "permission claim", SPAPIBinding.Spec.PermissionClaims)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *APIManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: understand how to properly watch for apibidings
	return ctrl.NewControllerManagedBy(mgr).
		For(&apisv1alpha1.APIBinding{}).
		Complete(r)
}

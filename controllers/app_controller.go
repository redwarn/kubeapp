/*
Copyright 2020 redwarn.

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
	"reflect"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "kubeapp/api/v1"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infra.iohub.me,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.iohub.me,resources=apps/status,verbs=get;update;patch

func (r *AppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("app", req.NamespacedName)
	log.Info("reconciling....")
	app := &infrav1.App{}
	err := r.Get(ctx, req.NamespacedName, app)
	if err != nil {
		if apierrs.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		log.Error(err, "unable to fetch app")
		return ctrl.Result{}, err
	}

	log.Info("get app successful.", "name", app.Name)

	// Determine whether the app is being deleted
	if !app.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("get deleted App, clean up subResources.")
		return reconcile.Result{}, nil
	}
	if err := r.syncStatus(app); err != nil {
		log.Error(err, "sync app status error", "namespace", app.Name)
	}
	if err := r.reconcileDeployMent(app); err != nil {
		log.Error(err, "reconcile deploy error", "name", app.Name)
		return reconcile.Result{}, err
	}
	log.Info("reconcile depoloy", "name", app.Name)
	if err := r.reconcileService(app); err != nil {
		log.Error(err, "reconcile svc error", "name", app.Name)
		return reconcile.Result{}, err
	}
	log.Info("reconcile svc", "name", app.Name)

	oldApp := &infrav1.App{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, oldApp); err != nil {
		log.Error(err, "unable to fetch app", "name", app.Name)
		return reconcile.Result{}, err
	}
	if reflect.DeepEqual(oldApp.Spec, app.Spec) {
		oldApp.Spec = app.Spec
		if err := r.Update(ctx, oldApp); err != nil {
			log.Error(err, "modify app error", "name", app.Name)
			return reconcile.Result{}, err
		}
	}
	log.Info("reconcile done")
	return ctrl.Result{}, nil
}

func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.App{}).
		Complete(r)
}

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

	infrav1 "github.com/redwarn/kubeapp/api/v1"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="traefik.containo.us",resources=traefikservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="traefik.containo.us",resources=traefikservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="traefik.containo.us",resources=tlsstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="traefik.containo.us",resources=tlsstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="traefik.containo.us",resources=tlsoptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="traefik.containo.us",resources=tlsoptions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="traefik.containo.us",resources=middlewares,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="traefik.containo.us",resources=middlewares/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="traefik.containo.us",resources=ingressrouteudps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="traefik.containo.us",resources=ingressrouteudps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="traefik.containo.us",resources=ingressroutetcps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="traefik.containo.us",resources=ingressroutetcps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="traefik.containo.us",resources=ingressroutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="traefik.containo.us",resources=ingressroutes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infra.iohub.me,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.iohub.me,resources=apps/status,verbs=get;update;patch

func (r *AppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log
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

	if app.Spec.Unit == "" {
		app.Spec.Unit = defaultUnit
	}
	log.Info("get app successful...")
	// Determine whether the app is being deleted

	finalizerName := "finalizers.iohub.me"
	if app.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(app.ObjectMeta.Finalizers, finalizerName) {
			app.ObjectMeta.Finalizers = append(app.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, app); err != nil {
				r.Log.Error(err, "add finalizers error")
				return ctrl.Result{}, err
			}
		}

	} else {
		if containsString(app.ObjectMeta.Finalizers, finalizerName) {
			if err := r.preDelete(app); err != nil {
				return ctrl.Result{}, nil
			}
			app.ObjectMeta.Finalizers = removeString(app.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, app); err != nil {
				r.Log.Error(err, "add finalizers error")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err := r.reconcileDeployMent(app); err != nil {
		log.Error(err, "reconcile deploy error")
		return ctrl.Result{}, err
	}
	log.Info("reconcile depoloy")
	if err := r.reconcileService(app); err != nil {
		log.Error(err, "reconcile svc error")
		return ctrl.Result{}, err
	}
	log.Info("reconcile svc")
	if err := r.reconclieIngressRoute(app); err != nil {
		log.Error(err, "reconcile ingressRoute error")
		return ctrl.Result{}, err
	}
	log.Info("reconcile ingressRoute")
	if err := r.reconclieMonitor(app); err != nil {
		log.Error(err, "reconcile podMonitor error")
		return ctrl.Result{}, err
	}
	log.Info("reconcile podMonitor")

	oldApp := &infrav1.App{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, oldApp); err != nil {
		log.Error(err, "unable to fetch app")
		return ctrl.Result{}, err
	}
	if reflect.DeepEqual(oldApp.Spec, app.Spec) {
		oldApp.Spec = app.Spec
		if err := r.Update(ctx, oldApp); err != nil {
			log.Error(err, "modify app error")
			return ctrl.Result{}, err
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

func (r *AppReconciler) preDelete(app *infrav1.App) error {
	r.Log.Info("PreDelete")
	return nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

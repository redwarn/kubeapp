package controllers

import (
	"context"
	"reflect"

	infrav1 "github.com/redwarn/kubeapp/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	ndots       string = "2"
	defaultUnit        = "blue"
	bizPortName        = "web"
)

func (r *AppReconciler) reconcileDeployMent(app *infrav1.App) error {
	log := r.Log

	deploy := r.genDeployment(app)

	if err := controllerutil.SetControllerReference(app, deploy, r.Scheme); err != nil {
		log.Error(err, "set App ControllerReference Error")
		return err
	}

	found := &appsv1.Deployment{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)

	if err != nil && apierrs.IsNotFound(err) {
		log.Info("deployment NotFound and Creating new one", "name", deploy.Name)
		if err = r.Create(context.TODO(), deploy); err != nil {
			return err
		}

	} else if err != nil {

		log.Error(err, "get Deployment info Error", "name", deploy.Name)
		return err

	} else if !reflect.DeepEqual(deploy.Spec.Template, found.Spec.Template) {

		// Update the found object and write the result back if there are any changes
		found.Spec = deploy.Spec
		found.ResourceVersion = ""
		log.Info("old deployment changed and Updating Deployment to reconcile", "name", deploy.Name)
		err = r.Update(context.TODO(), found)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *AppReconciler) genDeployment(app *infrav1.App) *appsv1.Deployment {
	labels := map[string]string{
		"app":  app.Spec.Name,
		"unit": app.Spec.Unit,
	}

	if app.Spec.Tag == "" {
		app.Spec.Tag = "latest"
	}
	var envs []v1.EnvVar
	var ports []v1.ContainerPort

	for k, v := range app.Spec.Env {
		envs = append(envs, v1.EnvVar{Name: k, Value: v, ValueFrom: nil})
	}
	envs = append(envs, v1.EnvVar{
		Name: "MEMORY",
		ValueFrom: &v1.EnvVarSource{
			ResourceFieldRef: &v1.ResourceFieldSelector{
				ContainerName: app.Spec.Name,
				Resource:      string(v1.ResourceRequestsMemory)},
		},
	})
	for _, p := range app.Spec.Ports {
		ports = append(ports, v1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      v1.Protocol(p.Protocol),
		})
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas:        &app.Spec.Replicas,
			MinReadySeconds: app.Spec.InitialDelaySeconds,
			Strategy: appsv1.DeploymentStrategy{
				RollingUpdate: &appsv1.RollingUpdateDeployment{},
			},
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					DNSConfig: &v1.PodDNSConfig{
						Options: []v1.PodDNSConfigOption{
							{
								Name: "single-request-reopen",
							}, {
								Name: "edns0",
							}, {
								Name:  "ndots",
								Value: &ndots,
							},
						},
					},
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: v1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: nil,
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      "app",
													Operator: metav1.LabelSelectorOpIn,
													Values:   []string{app.Spec.Name},
												}, {
													Key:      "unit",
													Operator: metav1.LabelSelectorOpIn,
													Values:   []string{app.Spec.Unit},
												},
											},
										},
										Namespaces:  []string{app.Namespace},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  app.Spec.Name,
							Image: app.Spec.Image + ":" + app.Spec.Tag,
							Ports: ports,
							Env:   envs,
						},
					},
				},
			},
		},
	}
	setLifecycle(deploy, app)
	setProbe(deploy, app)
	setResource(deploy, app)
	return deploy
}

func getPort(app *infrav1.App, portName string) intstr.IntOrString {
	var port intstr.IntOrString
	for _, p := range app.Spec.Ports {
		if p.Name == portName {
			port = intstr.FromInt(int(p.ContainerPort))
		}
	}
	return port
}
func getBizContainerIndex(deployment *appsv1.Deployment, app *infrav1.App) int {
	var idx int
	for i, c := range deployment.Spec.Template.Spec.Containers {
		if c.Name == app.Spec.Name {
			idx = i
		}
	}
	return idx
}
func setResource(deployment *appsv1.Deployment, app *infrav1.App) {

	Resources := v1.ResourceRequirements{}
	if app.Spec.Resource.RequestCpu != "" {
		Resources.Requests[v1.ResourceCPU]=resource.MustParse(app.Spec.Resource.RequestCpu)
	}
	
	if app.Spec.Resource.RequestMemory != "" {
		Resources.Requests[v1.ResourceMemory]= resource.MustParse(app.Spec.Resource.RequestMemory)
	}

	if app.Spec.Resource.LimitCpu != "" {
		Resources.Limits[v1.ResourceCPU]=resource.MustParse(app.Spec.Resource.LimitCpu)
	}

	if app.Spec.Resource.LimitMemory != "" {
		Resources.Limits[v1.ResourceMemory]=resource.MustParse(app.Spec.Resource.LimitMemory)
	}
	idx := getBizContainerIndex(deployment, app)
	deployment.Spec.Template.Spec.Containers[idx].Resources = Resources
}
func setLifecycle(deployment *appsv1.Deployment, app *infrav1.App) {
	port := getPort(app, bizPortName)
	Lifecycle := &v1.Lifecycle{}
	if app.Spec.PreStop != "" {
		Lifecycle.PreStop = &v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path:   app.Spec.PreStop,
				Port:   port,
				Scheme: "HTTP",
			},
		}
	}
	if app.Spec.PostStart != "" {
		Lifecycle.PostStart = &v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path:   app.Spec.PostStart,
				Port:   port,
				Scheme: "HTTP",
			},
		}
	}
	idx := getBizContainerIndex(deployment, app)
	deployment.Spec.Template.Spec.Containers[idx].Lifecycle = Lifecycle
}

func setProbe(deployment *appsv1.Deployment, app *infrav1.App) {
	port := getPort(app, bizPortName)
	var LivenessProbe, ReadinessProbe *v1.Probe
	if app.Spec.Health != "" {
		LivenessProbe = &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path:   app.Spec.Health,
					Port:   port,
					Scheme: "HTTP",
				},
			},
			InitialDelaySeconds: app.Spec.InitialDelaySeconds,
			TimeoutSeconds:      5,
			SuccessThreshold:    1,
			FailureThreshold:    5,
		}

		ReadinessProbe = &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path:   app.Spec.Health,
					Port:   port,
					Scheme: "HTTP",
				},
			},
			InitialDelaySeconds: app.Spec.InitialDelaySeconds,
			TimeoutSeconds:      3,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		}
	}
	idx := getBizContainerIndex(deployment, app)
	deployment.Spec.Template.Spec.Containers[idx].LivenessProbe = LivenessProbe
	deployment.Spec.Template.Spec.Containers[idx].ReadinessProbe = ReadinessProbe
}

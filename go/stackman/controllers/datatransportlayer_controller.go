/*
Copyright 2021.

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	stackv1 "github.com/ethereum-optimism/optimism/go/stackman/api/v1"
)

// DataTransportLayerReconciler reconciles a DataTransportLayer object
type DataTransportLayerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=datatransportlayers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=datatransportlayers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=datatransportlayers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DataTransportLayer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *DataTransportLayerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lgr := log.FromContext(ctx)

	crd := &stackv1.DataTransportLayer{}
	if err := r.Get(ctx, req.NamespacedName, crd); err != nil {
		if errors.IsNotFound(err) {
			lgr.Info("dtl resource not found, ignoring")
			return ctrl.Result{}, nil
		}

		lgr.Error(err, "error getting dtl")
		return ctrl.Result{}, err
	}

	created, err := GetOrCreateResource(ctx, r, func() client.Object {
		return r.entrypointsCfgMap(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "dtl-entrypoints"), &corev1.ConfigMap{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.deployment(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "dtl"), &appsv1.Deployment{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.service(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "dtl"), &corev1.Service{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataTransportLayerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stackv1.DataTransportLayer{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *DataTransportLayerReconciler) labels() map[string]string {
	return map[string]string{
		"app": "dtl",
	}
}

func (r *DataTransportLayerReconciler) entrypointsCfgMap(crd *stackv1.DataTransportLayer) *corev1.ConfigMap {
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "dtl-entrypoints", r.labels()),
		Data: map[string]string{
			"entrypoint.sh": DTLEntrypoint,
		},
	}
	ctrl.SetControllerReference(crd, cfgMap, r.Scheme)
	return cfgMap
}

func (r *DataTransportLayerReconciler) deployment(crd *stackv1.DataTransportLayer) *appsv1.Deployment {
	replicas := int32(1)
	defaultMode := int32(0o777)
	initContainers := []corev1.Container{
		{
			Name:            "wait-for-l1",
			Image:           "mslipper/wait-for-it:latest",
			ImagePullPolicy: corev1.PullAlways,
			Args: []string{
				StripHTTP(crd.Spec.L1URL),
				"-t",
				strconv.Itoa(crd.Spec.L1TimeoutSeconds),
			},
		},
	}
	baseEnv := []corev1.EnvVar{
		{
			Name:  "DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT",
			Value: crd.Spec.L1URL,
		},
	}
	if crd.Spec.DeployerURL != "" {
		initContainers = append(initContainers, corev1.Container{
			Name:            "wait-for-deployer",
			Image:           "mslipper/wait-for-it:latest",
			ImagePullPolicy: corev1.PullAlways,
			Args: []string{
				StripHTTP(crd.Spec.DeployerURL),
				"-t",
				strconv.Itoa(crd.Spec.DeployerTimeoutSeconds),
			},
		})
		baseEnv = append(baseEnv, corev1.EnvVar{
			Name:  "DEPLOYER_URL",
			Value: crd.Spec.DeployerURL,
		})
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "dtl", r.labels()),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "dtl",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: r.labels(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyAlways,
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:            "dtl",
							Image:           crd.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"/bin/sh",
								"/opt/entrypoints/entrypoint.sh",
								"node",
								"/opt/optimism/packages/data-transport-layer/dist/src/services/run.js",
							},
							Env: append(baseEnv, crd.Spec.Env...),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "db",
									MountPath: "/db",
								},
								{
									Name:      "entrypoints",
									MountPath: "/opt/entrypoints",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 7878,
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/eth/syncing",
										Port: intstr.FromInt(7878),
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "db",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "entrypoints",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: ObjectName(crd.ObjectMeta, "dtl-entrypoints"),
									},
									DefaultMode: &defaultMode,
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, deployment, r.Scheme)
	return deployment
}

func (r *DataTransportLayerReconciler) service(crd *stackv1.DataTransportLayer) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "dtl", r.labels()),
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "dtl",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 7878,
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, service, r.Scheme)
	return service
}

const DTLEntrypoint = `
#!/bin/sh

if [ -n "$DEPLOYER_URL" ]; then
	echo "Loading addresses from $DEPLOYER_URL."
	ADDRESSES=$(curl --fail --show-error --silent "$DEPLOYER_URL/addresses.json")
	export DATA_TRANSPORT_LAYER__ADDRESS_MANAGER=$(echo $ADDRESSES | jq -r ".AddressManager")
fi

exec "$@"
`

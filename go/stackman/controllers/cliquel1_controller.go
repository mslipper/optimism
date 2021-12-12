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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	stackv1 "github.com/ethereum-optimism/optimism/go/stackman/api/v1"
)

// CliqueL1Reconciler reconciles a CliqueL1 object
type CliqueL1Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=cliquel1s,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=cliquel1s/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=cliquel1s/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services;pods;configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CliqueL1 object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *CliqueL1Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lgr := log.FromContext(ctx)

	crd := &stackv1.CliqueL1{}
	if err := r.Get(ctx, req.NamespacedName, crd); err != nil {
		if errors.IsNotFound(err) {
			lgr.Info("clique resource, not found, ignoring")
			return ctrl.Result{}, nil
		}

		lgr.Error(err, "error getting clique")
		return ctrl.Result{}, err
	}

	created, err := GetOrCreateResource(ctx, r, func() client.Object {
		return r.genesisCfgMap(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "clique-genesis"), &corev1.ConfigMap{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	nsName := ObjectNamespacedName(crd.ObjectMeta, "clique")

	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.deployment(crd)
	}, nsName, &appsv1.Deployment{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.service(crd)
	}, nsName, &corev1.Service{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CliqueL1Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stackv1.CliqueL1{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *CliqueL1Reconciler) genesisCfgMap(crd *stackv1.CliqueL1) *corev1.ConfigMap {
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "clique-genesis", map[string]string{
			"app":        "clique",
			"clique_crd": crd.Namespace,
		}),
		Data: map[string]string{
			"genesis.json": stackv1.CliqueL1GenesisJSON,
			"init.sh":      stackv1.CliqueL1InitScript,
			"privkey.txt":  stackv1.CliqueL1SealerPrivateKey,
		},
	}
	ctrl.SetControllerReference(crd, cfgMap, r.Scheme)
	return cfgMap
}

func (r *CliqueL1Reconciler) deployment(crd *stackv1.CliqueL1) *appsv1.Deployment {
	replicas := int32(1)
	labels := map[string]string{
		"app":        "clique",
		"clique_crd": crd.Namespace,
	}
	image := "ethereum/client-go:v1.10.10"
	defaultMode := int32(0o777)
	deployment := &appsv1.Deployment{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "clique", labels),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "clique",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					InitContainers: []corev1.Container{
						{
							Name:  "clique-init-geth",
							Image: image,
							Command: []string{
								"/bin/sh",
								"-c",
								"/genesis/init.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "genesis",
									MountPath: "/genesis",
								},
								{
									Name:      "data",
									MountPath: "/data",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "clique-geth",
							Image:           image,
							ImagePullPolicy: corev1.PullAlways,
							Command:         makeGethCommand(stackv1.CliqueL1SealerAddress),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "genesis",
									MountPath: "/genesis",
								},
								{
									Name:      "data",
									MountPath: "/data",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8085,
								},
								{
									ContainerPort: 8086,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "genesis",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: ObjectName(crd.ObjectMeta, "clique-genesis"),
									},
									DefaultMode: &defaultMode,
								},
							},
						},
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
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

func (r *CliqueL1Reconciler) service(crd *stackv1.CliqueL1) *corev1.Service {
	labels := map[string]string{
		"app":        "clique",
		"clique_crd": crd.Namespace,
	}
	service := &corev1.Service{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "clique", labels),
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "clique",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "rpc",
					Port:       8545,
					TargetPort: intstr.FromInt(8545),
				},
				{
					Name:       "ws",
					Port:       8546,
					TargetPort: intstr.FromInt(8546),
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, service, r.Scheme)
	return service
}

func makeGethCommand(sealerAddress string) []string {
	return []string{
		"geth",
		"--datadir",
		"/data",
		"--http",
		"--http.corsdomain",
		"*",
		"--http.vhosts",
		"*",
		"--http.addr",
		"0.0.0.0",
		"--http.port",
		"8545",
		"--ws.addr",
		"0.0.0.0",
		"--ws.port",
		"8546",
		"--ws.origins",
		"*",
		"--syncmode",
		"full",
		"--nodiscover",
		"--maxpeers",
		"1",
		"--networkid",
		"777",
		"--unlock",
		sealerAddress,
		"--mine",
		"--password",
		"/data/password.txt",
		"--allow-insecure-unlock",
	}
}

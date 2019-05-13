/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package collectors

import (
	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descDeploymentLabelsName          = "kube_deployment_labels"
	descDeploymentLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descDeploymentLabelsDefaultLabels = []string{"namespace", "deployment"}

	deploymentMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_deployment_created",
			Type: metrics.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapDeploymentFunc(func(d *v1beta1.Deployment) metrics.Family {
				f := metrics.Family{}

				if !d.CreationTimestamp.IsZero() {
					f = append(f, &metrics.Metric{
						Name:  "kube_deployment_created",
						Value: float64(d.CreationTimestamp.Unix()),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_deployment_status_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "The number of replicas per deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1beta1.Deployment) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_deployment_status_replicas",
					Value: float64(d.Status.Replicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_deployment_status_replicas_available",
			Type: metrics.MetricTypeGauge,
			Help: "The number of available replicas per deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1beta1.Deployment) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_deployment_status_replicas_available",
					Value: float64(d.Status.AvailableReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_deployment_status_replicas_unavailable",
			Type: metrics.MetricTypeGauge,
			Help: "The number of unavailable replicas per deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1beta1.Deployment) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_deployment_status_replicas_unavailable",
					Value: float64(d.Status.UnavailableReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_deployment_spec_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "Number of desired pods for a deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1beta1.Deployment) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_deployment_spec_replicas",
					Value: float64(*d.Spec.Replicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: descDeploymentLabelsName,
			Type: metrics.MetricTypeGauge,
			Help: descDeploymentLabelsHelp,
			GenerateFunc: wrapDeploymentFunc(func(d *v1beta1.Deployment) metrics.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(d.Labels)
				return metrics.Family{&metrics.Metric{
					Name:        descDeploymentLabelsName,
					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}}
			}),
		},
	}
)

func wrapDeploymentFunc(f func(*v1beta1.Deployment) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		deployment := obj.(*v1beta1.Deployment)

		metricFamily := f(deployment)

		for _, m := range metricFamily {
			m.LabelKeys = append(descDeploymentLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{deployment.Namespace, deployment.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createDeploymentListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ExtensionsV1beta1().Deployments(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ExtensionsV1beta1().Deployments(ns).Watch(opts)
		},
	}
}

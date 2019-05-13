/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descNamespaceLabelsName          = "kube_namespace_labels"
	descNamespaceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNamespaceLabelsDefaultLabels = []string{"namespace"}

	descNamespaceAnnotationsName          = "kube_namespace_annotations"
	descNamespaceAnnotationsHelp          = "Kubernetes annotations converted to Prometheus labels."
	descNamespaceAnnotationsDefaultLabels = []string{"namespace"}

	namespaceMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: descNamespaceLabelsName,
			Type: metrics.MetricTypeGauge,
			Help: descNamespaceLabelsHelp,
			GenerateFunc: wrapNamespaceFunc(func(n *v1.Namespace) metrics.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(n.Labels)
				return metrics.Family{&metrics.Metric{
					Name:        descNamespaceLabelsName,
					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: descNamespaceAnnotationsName,
			Type: metrics.MetricTypeGauge,
			Help: descNamespaceAnnotationsHelp,
			GenerateFunc: wrapNamespaceFunc(func(n *v1.Namespace) metrics.Family {
				annotationKeys, annotationValues := kubeAnnotationsToPrometheusAnnotations(n.Annotations)
				return metrics.Family{&metrics.Metric{
					Name:        descNamespaceAnnotationsName,
					LabelKeys:   annotationKeys,
					LabelValues: annotationValues,
					Value:       1,
				}}
			}),
		},
	}
)

func wrapNamespaceFunc(f func(*v1.Namespace) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		namespace := obj.(*v1.Namespace)

		metricFamily := f(namespace)

		for _, m := range metricFamily {
			m.LabelKeys = append(descNamespaceLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{namespace.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createNamespaceListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Namespaces().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Namespaces().Watch(opts)
		},
	}
}

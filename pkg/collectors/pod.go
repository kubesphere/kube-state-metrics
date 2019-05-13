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
	"strconv"

	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/util/node"
)

var (
	descPodLabelsDefaultLabels = []string{"namespace", "pod"}
	containerWaitingReasons    = []string{"ContainerCreating", "CrashLoopBackOff", "CreateContainerConfigError", "ErrImagePull", "ImagePullBackOff"}
	containerTerminatedReasons = []string{"OOMKilled", "Completed", "Error", "ContainerCannotRun"}

	podMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_pod_info",
			Type: metrics.MetricTypeGauge,
			Help: "Information about pod.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metrics.Family {
				createdBy := metav1.GetControllerOf(p)
				createdByKind := "<none>"
				createdByName := "<none>"
				if createdBy != nil {
					if createdBy.Kind != "" {
						createdByKind = createdBy.Kind
					}
					if createdBy.Name != "" {
						createdByName = createdBy.Name
					}
				}

				m := metrics.Metric{
					Name:        "kube_pod_info",
					LabelKeys:   []string{"host_ip", "pod_ip", "uid", "node", "created_by_kind", "created_by_name"},
					LabelValues: []string{p.Status.HostIP, p.Status.PodIP, string(p.UID), p.Spec.NodeName, createdByKind, createdByName},
					Value:       1,
				}

				return metrics.Family{&m}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_pod_owner",
			Type: metrics.MetricTypeGauge,
			Help: "Information about the Pod's owner.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metrics.Family {
				labelKeys := []string{"owner_kind", "owner_name", "owner_is_controller"}
				f := metrics.Family{}

				owners := p.GetOwnerReferences()
				if len(owners) == 0 {
					f = append(f, &metrics.Metric{
						Name:        "kube_pod_owner",
						LabelKeys:   labelKeys,
						LabelValues: []string{"<none>", "<none>", "<none>"},
						Value:       1,
					})
				} else {
					for _, owner := range owners {
						if owner.Controller != nil {
							f = append(f, &metrics.Metric{
								Name:        "kube_pod_owner",
								LabelKeys:   labelKeys,
								LabelValues: []string{owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller)},
								Value:       1,
							})
						} else {
							f = append(f, &metrics.Metric{
								Name:        "kube_pod_owner",
								LabelKeys:   labelKeys,
								LabelValues: []string{owner.Kind, owner.Name, "false"},
								Value:       1,
							})
						}
					}
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_pod_status_phase",
			Type: metrics.MetricTypeGauge,
			Help: "The pods current phase.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metrics.Family {
				f := metrics.Family{}

				phase := p.Status.Phase
				if phase == "" {
					return f
				}

				phases := []struct {
					v bool
					n string
				}{
					{phase == v1.PodPending, string(v1.PodPending)},
					{phase == v1.PodSucceeded, string(v1.PodSucceeded)},
					{phase == v1.PodFailed, string(v1.PodFailed)},
					// This logic is directly copied from: https://github.com/kubernetes/kubernetes/blob/d39bfa0d138368bbe72b0eaf434501dcb4ec9908/pkg/printers/internalversion/printers.go#L597-L601
					// For more info, please go to: https://github.com/kubernetes/kube-state-metrics/issues/410
					{phase == v1.PodRunning && !(p.DeletionTimestamp != nil && p.Status.Reason == node.NodeUnreachablePodReason), string(v1.PodRunning)},
					{phase == v1.PodUnknown || (p.DeletionTimestamp != nil && p.Status.Reason == node.NodeUnreachablePodReason), string(v1.PodUnknown)},
				}

				for _, p := range phases {
					f = append(f, &metrics.Metric{
						Name:        "kube_pod_status_phase",
						LabelKeys:   []string{"phase"},
						LabelValues: []string{p.n},
						Value:       boolFloat64(p.v),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_pod_status_ready",
			Type: metrics.MetricTypeGauge,
			Help: "Describes whether the pod is ready to serve requests.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metrics.Family {
				f := metrics.Family{}

				for _, c := range p.Status.Conditions {
					switch c.Type {
					case v1.PodReady:
						ms := addConditionMetrics(c.Status)

						for _, m := range ms {
							metric := m
							metric.Name = "kube_pod_status_ready"
							metric.LabelKeys = []string{"condition"}
							f = append(f, metric)
						}
					}
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_pod_status_scheduled",
			Type: metrics.MetricTypeGauge,
			Help: "Describes the status of the scheduling process for the pod.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metrics.Family {
				f := metrics.Family{}

				for _, c := range p.Status.Conditions {
					switch c.Type {
					case v1.PodScheduled:
						ms := addConditionMetrics(c.Status)

						for _, m := range ms {
							metric := m
							metric.Name = "kube_pod_status_scheduled"
							metric.LabelKeys = []string{"condition"}
							f = append(f, metric)
						}
					}
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_pod_container_status_waiting_reason",
			Type: metrics.MetricTypeGauge,
			Help: "Describes the reason the container is currently in waiting state.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metrics.Family {
				f := metrics.Family{}

				for _, cs := range p.Status.ContainerStatuses {
					for _, reason := range containerWaitingReasons {
						f = append(f, &metrics.Metric{
							Name:        "kube_pod_container_status_waiting_reason",
							LabelKeys:   []string{"container", "reason"},
							LabelValues: []string{cs.Name, reason},
							Value:       boolFloat64(waitingReason(cs, reason)),
						})
					}
				}

				return f
			}),
		},
	}
)

func wrapPodFunc(f func(*v1.Pod) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		pod := obj.(*v1.Pod)

		metricFamily := f(pod)

		for _, m := range metricFamily {
			m.LabelKeys = append(descPodLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{pod.Namespace, pod.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createPodListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Pods(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Pods(ns).Watch(opts)
		},
	}
}

func waitingReason(cs v1.ContainerStatus, reason string) bool {
	if cs.State.Waiting == nil {
		return false
	}
	return cs.State.Waiting.Reason == reason
}

func terminationReason(cs v1.ContainerStatus, reason string) bool {
	if cs.State.Terminated == nil {
		return false
	}
	return cs.State.Terminated.Reason == reason
}

func lastTerminationReason(cs v1.ContainerStatus, reason string) bool {
	if cs.LastTerminationState.Terminated == nil {
		return false
	}
	return cs.LastTerminationState.Terminated.Reason == reason
}

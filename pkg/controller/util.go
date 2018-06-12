package main

import (
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/Nexinto/go-icinga2-client/icinga2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (c *Controller) MakeVars(o metav1.Object, typ string, namespaced bool) map[string]string {
	var nsvar string
	if namespaced {
		nsvar = o.GetNamespace()
	} else {
		nsvar = ""
	}
	return mergeVars(c.DefaultVars, map[string]string{VarName: o.GetName(), VarType: typ, VarCluster: c.Tag, VarNamespace: nsvar})
}

func strmapsDiffer(a map[string]string, b map[string]string) bool {
	return !reflect.DeepEqual(a, b)
}

func varsDiffer(a icinga2.Vars, b icinga2.Vars) bool {
	return !reflect.DeepEqual(a, b)
}

func metadataDiffer(a metav1.Object, b metav1.Object) bool {
	if a.GetName() != b.GetName() || a.GetNamespace() != b.GetNamespace() || !reflect.DeepEqual(a.GetLabels(), b.GetLabels()) {
		return true
	}

	if len(a.GetAnnotations()) != len(b.GetAnnotations()) {
		return true
	}

	for k, v := range a.GetAnnotations() {
		if k == "kubectl.kubernetes.io/last-applied-configuration" {
			continue
		}

		if b.GetAnnotations()[k] != v {
			return true
		}
	}

	return false
}

func mergeVars(maps ...interface{}) map[string]string {
	r := make(map[string]string)

	for _, mm := range maps {
		if m, ok := mm.(icinga2.Vars); ok {
			for k, v := range m {
				if vv, ok := v.(string); ok {
					r[k] = vv
				}
			}
		} else if m, ok := mm.(map[string]string); ok {
			for k, v := range m {
				r[k] = v
			}
		} else {
			panic("called mergeVars with something that is not a stringmap or Vars")
		}
	}

	return r
}

func Vars(m map[string]string) icinga2.Vars {
	vars := make(icinga2.Vars)
	for k, v := range m {
		vars[k] = v
	}
	return vars
}

// True if this object should be monitored
func (c *Controller) monitored(o metav1.Object) bool {
	if a, ok := o.GetAnnotations()[AnnDisableMonitoring]; ok && a != "" {
		return false
	}
	if len(o.GetOwnerReferences()) > 0 {
		return false
	}
	if ns := o.GetNamespace(); ns != "" {
		namespace, err := c.Kubernetes.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
		if err != nil {
			log.Errorf("error getting namespace '%s': %s", ns, err.Error())
			return false
		}
		if a, ok := namespace.GetAnnotations()[AnnDisableMonitoring]; ok && a != "" {
			return false
		}
	}

	return true
}

// Create an event for an object.
func MakeEvent(kube kubernetes.Interface, o metav1.Object, message, kind string, warn bool) error {
	var t string
	if warn {
		t = "Warning"
	} else {
		t = "Normal"
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: o.GetName(),
			Namespace:    o.GetNamespace(),
		},
		InvolvedObject: corev1.ObjectReference{
			Name:            o.GetName(),
			Namespace:       o.GetNamespace(),
			APIVersion:      "v1",
			UID:             o.GetUID(),
			Kind:            kind,
			ResourceVersion: o.GetResourceVersion(),
		},
		Message:        message,
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Type:           t,
	}

	_, err := kube.CoreV1().Events(o.GetNamespace()).Create(event)
	return err
}

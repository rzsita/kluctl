package validation

import (
	"fmt"
	"github.com/codablock/kluctl/pkg/types"
	"github.com/codablock/kluctl/pkg/utils/uo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strconv"
	"strings"
)

const resultAnnotation = "validate-result.kluctl.io/"

type validationFailed struct {
}

func (err *validationFailed) Error() string {
	return "validation failed"
}

type condition struct {
	status  string
	reason  string
	message string
}

func (c condition) getMessage(def string) string {
	if c.message == "" {
		return def
	}
	return c.message
}

func ValidateObject(o *unstructured.Unstructured, notReadyIsError bool) (ret types.ValidateResult) {
	ref := types.RefFromObject(o)

	// We assume all is good in case no validation is performed
	ret.Ready = true

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*validationFailed); ok {
				// all good
			} else if e, ok := r.(error); ok {
				err := fmt.Errorf("panic in ValidateObject: %w", e)
				ret.Errors = append(ret.Errors, types.DeploymentError{Ref: ref, Error: err.Error()})
			} else {
				err := fmt.Errorf("panic in ValidateObject: %v", e)
				ret.Errors = append(ret.Errors, types.DeploymentError{Ref: ref, Error: err.Error()})
			}
			ret.Ready = false
		}
	}()

	for k, v := range o.GetAnnotations() {
		if strings.HasPrefix(k, resultAnnotation) {
			ret.Results = append(ret.Results, types.ValidateResultEntry{
				Ref:        ref,
				Annotation: k,
				Message:    v,
			})
		}
	}

	u := uo.FromUnstructured(o)
	status, ok, _ := u.GetNestedObject("status")
	if !ok {
		return
	}

	addError := func(message string) {
		ret.Errors = append(ret.Errors, types.DeploymentError{
			Ref:   ref,
			Error: message,
		})
	}
	addWarning := func(message string) {
		ret.Warnings = append(ret.Warnings, types.DeploymentError{
			Ref:   ref,
			Error: message,
		})
	}
	addNotReady := func(message string) {
		if notReadyIsError {
			addError(message)
		} else {
			addWarning(message)
		}
		ret.Ready = false
	}

	findConditions := func(typ string, doError bool, doRaise bool) []condition {
		var ret []condition
		l, ok, _ := status.GetNestedObjectList("conditions")
		if ok {
			for _, c := range l {
				t, _, _ := c.GetNestedString("type")
				status, _, _ := c.GetNestedString("status")
				reason, _, _ := c.GetNestedString("reason")
				message, _, _ := c.GetNestedString("message")
				if t == typ {
					ret = append(ret, condition{
						status:  status,
						reason:  reason,
						message: message,
					})
				}
			}
		}
		if len(ret) == 0 && doError {
			err := fmt.Errorf("%s condition not in status", typ)
			addError(err.Error())
			if doRaise {
				panic(&validationFailed{})
			}
		}
		return ret
	}

	getCondition := func(typ string, doError bool, doRaise bool) condition {
		c := findConditions(typ, doError, doRaise)
		if len(c) == 0 {
			return condition{}
		}
		if len(c) != 1 {
			err := fmt.Errorf("%s condition found more then once", typ)
			addError(err.Error())
			if doRaise {
				panic(&validationFailed{})
			}
		}
		return c[0]
	}
	getStatusField := func(field string, doError bool, doRaise bool, def interface{}) interface{} {
		v, ok, _ := status.GetNestedField(field)
		if !ok && doError {
			err := fmt.Errorf("%s field not in status or empty", field)
			addError(err.Error())
			if doRaise {
				panic(&validationFailed{})
			}
		}
		if !ok {
			return def
		}
		return v
	}
	getStatusFieldStr := func(field string, doError bool, doRaise bool, def string) string {
		v := getStatusField(field, doError, doRaise, def)
		if s, ok := v.(string); ok {
			return s
		} else {
			err := fmt.Errorf("%s field is not a string", field)
			addError(err.Error())
			if doRaise {
				panic(&validationFailed{})
			}
		}
		return def
	}
	getStatusFieldInt := func(field string, doError bool, doRaise bool, def int64) int64 {
		v := getStatusField(field, doError, doRaise, def)
		if i, ok := v.(int64); ok {
			return i
		} else if i, ok := v.(uint64); ok {
			return int64(i)
		} else if i, ok := v.(int); ok {
			return int64(i)
		} else {
			err := fmt.Errorf("%s field is not an int", field)
			if doError {
				addError(err.Error())
			}
			if doRaise {
				panic(&validationFailed{})
			}
		}
		return def
	}
	parseIntOrPercent := func(v interface{}) (int64, bool, error) {
		if i, ok := v.(int64); ok {
			return i, false, nil
		}
		if i, ok := v.(uint64); ok {
			return int64(i), false, nil
		}
		if i, ok := v.(int); ok {
			return int64(i), false, nil
		}
		if s, ok := v.(string); ok {
			s = strings.ReplaceAll(s, "%", "")
			i, err := strconv.ParseInt(s, 10, 32)
			if err != nil {
				return 0, false, err
			}
			return i, true, nil
		}
		return 0, false, fmt.Errorf("don't know how to parse %v", v)
	}
	valueFromIntOrPercent := func(v interface{}, total int64) (int64, error) {
		i, isPercent, err := parseIntOrPercent(v)
		if err != nil {
			return 0, err
		}
		if isPercent {
			return int64((float32(i) * float32(total)) / 100), nil
		}
		return i, nil
	}

	switch u.GetK8sGVK().GroupKind() {
	case schema.GroupKind{Group: "", Kind: "Pod"}:
		c := getCondition("Ready", false, false)
		if c.status != "True" {
			addNotReady(c.getMessage("Not ready"))
		}
	case schema.GroupKind{Group: "batch", Kind: "Job"}:
		c := getCondition("Failed", false, false)
		if c.status == "True" {
			addError(c.getMessage("Failed"))
		} else {
			c = getCondition("Complete", false, false)
			if c.status != "True" {
				addNotReady(c.getMessage("Not completed"))
			}
		}
	case schema.GroupKind{Group: "apps", Kind: "Deployment"}:
		readyReplicas := getStatusFieldInt("readyReplicas", true, true, 0)
		replicas := getStatusFieldInt("replicas", true, true, 0)
		if readyReplicas < replicas {
			addNotReady(fmt.Sprintf("readyReplicas (%d) is less then replicas (%d)", readyReplicas, replicas))
		}
	case schema.GroupKind{Group: "", Kind: "PersistentVolumeClaim"}:
		phase := getStatusFieldStr("phase", true, true, "")
		if phase != "Bound" {
			addNotReady("Volume is not bound")
		}
	case schema.GroupKind{Group: "", Kind: "Service"}:
		svcType, _, _ := u.GetNestedString("spec", "type")
		if svcType != "ExternalName" {
			clusterIP, _, _ := u.GetNestedString("spec", "clusterIP")
			if clusterIP == "" {
				addError("Service does not have a cluster IP")
			} else if svcType == "LoadBalancer" {
				externalIPs, _, _ := u.GetNestedList("spec", "externalIPs")
				if len(externalIPs) == 0 {
					ingress, _, _ := status.GetNestedList("loadBalancer", "ingress")
					if len(ingress) == 0 {
						addNotReady("Not ready")
					}
				}
			}
		}
	case schema.GroupKind{Group: "apps", Kind: "DaemonSet"}:
		updateStrategyType, _, _ := u.GetNestedString("spec", "updateStrategy", "type")
		if updateStrategyType == "RollingUpdate" {
			updatedNumberScheduled := getStatusFieldInt("updatedNumberScheduled", true, true, 0)
			desiredNumberScheduled := getStatusFieldInt("desiredNumberScheduled", true, true, 0)
			if updatedNumberScheduled != desiredNumberScheduled {
				addNotReady(fmt.Sprintf("DaemonSet is not ready. %d out of %d expected pods have been scheduled", updatedNumberScheduled, desiredNumberScheduled))
			} else {
				maxUnavailableI, _, _ := u.GetNestedField("spec", "updateStrategy", "maxUnavailable")
				if maxUnavailableI == nil {
					maxUnavailableI = 1
				}
				maxUnavailable, err := valueFromIntOrPercent(maxUnavailableI, desiredNumberScheduled)
				if err != nil {
					maxUnavailable = desiredNumberScheduled
				}
				expectedReady := desiredNumberScheduled - maxUnavailable
				numberReady := getStatusFieldInt("numberReady", true, true, 0)
				if numberReady < expectedReady {
					addNotReady(fmt.Sprintf("DaemonSet is not ready. %d out of %d expected pods are ready", numberReady, expectedReady))
				}
			}
		}
	case schema.GroupKind{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition"}:
		// This is based on how Helm check for ready CRDs.
		// See https://github.com/helm/helm/blob/249d1b5fb98541f5fb89ab11019b6060d6b169f1/pkg/kube/ready.go#L342
		c := getCondition("Established", false, false)
		if c.status != "True" {
			c = getCondition("NamesAccepted", true, true)
			if c.status != "False" {
				addNotReady("CRD is not ready")
			}
		}
	case schema.GroupKind{Group: "apps", Kind: "StatefulSet"}:
		updateStrategyType, _, _ := u.GetNestedString("spec", "updateStrategy", "type")
		if updateStrategyType == "RollingUpdate" {
			partition, _, _ := u.GetNestedInt("spec", "updateStrategy", "rollingUpdate", "partition")
			replicas, ok, _ := u.GetNestedInt("spec", "replicas")
			if !ok {
				replicas = 1
			}
			updatedReplicas := getStatusFieldInt("updatedReplicas", true, true, 0)
			expectedReplicas := replicas - partition
			if updatedReplicas != expectedReplicas {
				addNotReady(fmt.Sprintf("StatefulSet is not ready. %d out of %d expected pods have been scheduled", updatedReplicas, expectedReplicas))
			} else {
				readyReplicas := getStatusFieldInt("readyReplicas", true, true, 0)
				if readyReplicas != replicas {
					addNotReady(fmt.Sprintf("StatefulSet is not ready. %d out of %d expected pods are ready", readyReplicas, replicas))
				}
			}
		}
	}
	return
}
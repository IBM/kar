package sidecar

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.ibm.com/solsa/kar.git/pkg/logger"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	appNameAnnotation     string = "kar.ibm.com/app"
	serviceNameAnnotation string = "kar.ibm.com/service"
	sidecarName           string = "kar"
	sidecarImage          string = "us.icr.io/kar-dev/kar:nightly"
)

func toV1AdmissionResponse(err error) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func possiblyInjectSidecar(ar v1.AdmissionReview) *v1.AdmissionResponse {
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		logger.Error("expected resource to be %s", podResource)
		return nil
	}

	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		logger.Error(err.Error())
		return toV1AdmissionResponse(err)
	}

	reviewResponse := v1.AdmissionResponse{}
	reviewResponse.Allowed = true

	annotations := pod.GetObjectMeta().GetAnnotations()
	if appName, ok := annotations[appNameAnnotation]; ok {
		logger.Info("Pod %v has appName %v", pod.Name, appName)

		sidecar := corev1.Container{Name: sidecarName, Image: sidecarImage}
		patch := []patchOperation{{
			Op:    "add",
			Path:  "/spec/containers/-",
			Value: sidecar,
		}}
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			logger.Error("WTF %v", err)
			return toV1AdmissionResponse(err)
		}

		reviewResponse.Patch = patchBytes
		pt := v1.PatchTypeJSONPatch
		reviewResponse.PatchType = &pt
	} else {
		logger.Info("Pod %v lacks 'kar.ibm.com/app' annotation; no sidecar injected", pod.Name)
	}

	return &reviewResponse
}

// HandleAdmissionRequest decodes and processes the body of an AdmissionRequest.
// If the processing is successful, it returns an AdmissionReview instance and http.StatusOK.
// If the processing is unsuccessful, it returns an http error code and error.
func HandleAdmissionRequest(body []byte) (runtime.Object, int, error) {
	deserializer := codecs.UniversalDeserializer()
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("Request could not be decoded: %w", err)
	}

	var responseObj runtime.Object
	switch *gvk {
	case v1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*v1.AdmissionReview)
		if !ok {
			return nil, http.StatusBadRequest, fmt.Errorf("Expected v1.AdmissionReview but got: %T", obj)
		}
		responseAdmissionReview := &v1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = possiblyInjectSidecar(*requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	default:
		return nil, http.StatusBadRequest, fmt.Errorf("Unsupported group version kind: %v", gvk)
	}

	return responseObj, http.StatusOK, nil
}

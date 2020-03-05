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
	appNameAnnotation     = "kar.ibm.com/app"
	serviceNameAnnotation = "kar.ibm.com/service"
	sendPortAnnotation    = "kar.ibm.com/sendPort"
	recvPortAnnotation    = "kar.ibm.com/recvPort"
	verboseAnnotation     = "kar.ibm.com/verbose"

	sidecarName            = "kar"
	sidecarImage           = "us.icr.io/kar-dev/kar:latest"
	sidecarImagePullPolicy = corev1.PullAlways
)

// HandleAdmissionRequest decodes and processes the body of an AdmissionRequest.
// If the processing is successful, it returns an AdmissionReview instance and http.StatusOK.
// If the processing is unsuccessful, it returns an http error code and error.
func HandleAdmissionRequest(body []byte, config Config) (runtime.Object, int, error) {
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
		responseAdmissionReview.Response = possiblyInjectSidecar(*requestedAdmissionReview, config)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	default:
		return nil, http.StatusBadRequest, fmt.Errorf("Unsupported group version kind: %v", gvk)
	}

	return responseObj, http.StatusOK, nil
}

func possiblyInjectSidecar(ar v1.AdmissionReview, config Config) *v1.AdmissionResponse {
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

		sidecar := corev1.Container{
			Name:            sidecarName,
			Image:           sidecarImage,
			ImagePullPolicy: sidecarImagePullPolicy,
			Command:         []string{"/kar/kar"},
			Args:            constructArgs(pod),
			Env:             constructEnvironment(pod, config),
		}
		patch := []patchOperation{{
			Op:    "add",
			Path:  "/spec/containers/-",
			Value: sidecar,
		}}
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			logger.Error(err.Error())
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

func constructArgs(pod corev1.Pod) []string {
	annotations := pod.GetObjectMeta().GetAnnotations()
	appName := annotations[appNameAnnotation]
	cmd := []string{"-app", appName}
	if serviceName, ok := annotations[serviceNameAnnotation]; ok {
		cmd = append(cmd, "-service", serviceName)
	}
	if sendPort, ok := annotations[sendPortAnnotation]; ok {
		cmd = append(cmd, "-send", sendPort)
	}
	if recvPort, ok := annotations[recvPortAnnotation]; ok {
		cmd = append(cmd, "-recv", recvPort)
	}
	if verbose, ok := annotations[verboseAnnotation]; ok {
		cmd = append(cmd, "-verbose", verbose)
	}
	return cmd
}

func constructEnvironment(pod corev1.Pod, config Config) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{Name: "REDIS_HOST", Value: config.RedisHost},
		{Name: "REDIS_PORT", Value: config.RedisPort},
		{Name: "REDIS_PASSWORD", Value: config.RedisPassword},
		{Name: "KAFKA_BROKERS", Value: config.KafkaBrokers},
		{Name: "KAFKA_PASSWORD", Value: config.KafkaPassword},
		{Name: "KAFKA_USERNAME", Value: config.KafkaUsername},
	}
	return env
}

func toV1AdmissionResponse(err error) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

package main

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gin-gonic/gin"

	"log"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	r := gin.Default()
	r.GET("/healtz", health)
	r.POST("/mutate", handleMutate)

	err := r.RunTLS(":8080", "/etc/certs/tls.crt", "/etc/certs/tls.key")
	if err != nil {
		panic(err)
	}

	log.Println("Starting server ...")

}
func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func handleMutate(c *gin.Context) {

	admissionReview := v1.AdmissionReview{}

	var err error
	if err = c.BindJSON(&admissionReview); err != nil {
		c.Abort()
		return
	}

	admissionReviewReq := admissionReview.Request

	log.Println("Incoming payload", admissionReviewReq)

	var pod *corev1.Pod

	if err = json.Unmarshal(admissionReviewReq.Object.Raw, &pod); err != nil {
		c.Abort()
		return
	}

	response := v1.AdmissionResponse{}

	patchType := v1.PatchTypeJSONPatch
	response.PatchType = &patchType
	response.UID = admissionReviewReq.UID

	if response.Patch, err = GenerateJSONPatch(pod); err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{
			Status: "Failed",
		}
		fmt.Println(err.Error())
	} else {
		response.Allowed = true
		response.Result = &metav1.Status{
			Status: "Success",
		}
	}

	admissionReview.Response = &response

	c.JSON(http.StatusOK, admissionReview)
}

func addResourceLimits(pod *corev1.Pod) ([]byte, error) {

	log.Println("calling addResourceLimits")

	var patch []map[string]interface{}
	for i, container := range pod.Spec.Containers {
		if container.Resources.Limits == nil {
			patch = append(patch, map[string]interface{}{
				"op":   "add",
				"path": fmt.Sprintf("/spec/containers/%d/resources", i),
				"value": map[string]map[string]string{
					"requests": {
						"cpu":    "150m",
						"memory": "128Mi",
					},
					"limits": {
						"cpu":    "300m",
						"memory": "256Mi",
					},
				},
			},
			)
		}
	}

	log.Println(patch)

	return json.Marshal(patch)

}

func addLabels(pod *corev1.Pod) ([]byte, error) {
	var patch []map[string]interface{}

	if pod.Labels["cumulo.ai"] == "" {
		patch = append(patch, map[string]interface{}{
			"op":    "add",
			"path":  "/metadata/labels/cumulo.ai",
			"value": "true",
		})
	} else {
		patch = append(patch, map[string]interface{}{
			"op":    "replace",
			"path":  "/metadata/labels/cumulo.ai",
			"value": "true",
		})
	}

	return json.Marshal(patch)
}

// The resulting JSON patch would look like this:

// [
// 	{
// 		"op": "add",
// 		"path": "/metadata/labels/cumulo.ai",
// 		"value": "true"
// 	}
// ]

// Or if the label already exists:

// [
// 	{
// 		"op": "replace",
// 		"path": "/metadata/labels/cumulo.ai",
// 		"value": "true"
// 	}
// ]

/*
"I need a Go function that generates a JSON patch to add or update a specific label ('cumulo.ai') on a Kubernetes Pod object. The label should be set to 'true' if it doesn't exist and toggled to 'false' if it already exists. The input param should be pointer to Group Version POD object. Create a JSON path as an array of operations and return a byte slice containing the JSON patch. Each operation in the patch should be represented by a struct with fields Op, Path, and Value, serialized as "op", "path", and "value" respectively in the JSON output.  "`

*/

func GenerateJSONPatch(pod *corev1.Pod) ([]byte, error) {
	// Check if the pod object has labels
	labels := pod.GetLabels()

	// Create an empty array to store operations
	operations := make([]Operation, 0)

	// Check if the 'cumulo.ai' label exists
	if _, ok := labels["cumulo.ai"]; !ok {
		// Label does not exist, add it with value 'true'
		operations = append(operations, Operation{Op: "add", Path: "/metadata/labels/cumulo.ai", Value: "true"})
	} else {
		// Label exists, toggle its value to 'false'
		operations = append(operations, Operation{Op: "replace", Path: "/metadata/labels/cumulo.ai", Value: "false"})
	}

	// Serialize the operations into a JSON byte slice
	patch, err := json.Marshal(operations)
	if err != nil {
		return nil, err
	}

	log.Println("added label - GenerateJSONPatch")

	return patch, nil
}

// Operation struct to represent each operation in the JSON patch
type Operation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

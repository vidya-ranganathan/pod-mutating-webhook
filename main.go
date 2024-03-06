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
"I need a Go function that Generates a JSON patch to add or update a specific label ('cumulo.ai') on a Kubernetes Pod object. The label should be set to 'true' if it doesn't exist and toggled to 'false' if it already exists. The input param should be pointer to Group Version POD object. Create a JSON path as an array of operations and return a byte slice containing the JSON patch. Each operation in the patch should be represented by a struct with fields Op, Path, and Value, serialized as "op", "path", and "value" respectively in the JSON output.  "`

*/

func GenerateJSONPatchPrev(pod *corev1.Pod) ([]byte, error) {
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

func GenerateJSONPatchPrev2(pod *corev1.Pod) ([]byte, error) {
	// check to see if the label 'cumulo.ai' already exists
	labelValue, ok := pod.Labels["cumulo.ai"]

	if !ok {
		// if label does not exist, set it to true
		pod.Labels["cumulo.ai"] = "true"
	} else {
		// if label exists, toggle its value to false
		if labelValue == "true" {
			pod.Labels["cumulo.ai"] = "false"
		} else {
			pod.Labels["cumulo.ai"] = "true"
		}
	}

	// create array of operations for the JSON patch
	var patch []struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}

	// add the operation to add or update the label
	patch = append(patch, struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{
		Op:    "add",
		Path:  "/metadata/labels/cumulo.ai",
		Value: pod.Labels["cumulo.ai"],
	})

	// serialize the array of operations to a byte slice containing the JSON patch
	patchByte, err := json.Marshal(patch)

	if err != nil {
		return nil, err
	}

	return patchByte, nil
}

func GenerateJSONPatchPrev3(pod *corev1.Pod) ([]byte, error) {
	// check if label already exists
	if _, ok := pod.Labels["cumulo.ai"]; ok {
		// toggle value to "false"
		pod.Labels["cumulo.ai"] = "false"
	} else {
		// set label to "true" if it doesn't exist
		pod.Labels["cumulo.ai"] = "true"
	}

	// create array of operations
	var operations []interface{}
	// create operation to update label
	updateOperation := struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{
		Op:    "add",
		Path:  "/metadata/labels/cumulo.ai",
		Value: pod.Labels["cumulo.ai"],
	}

	// add operation to array
	operations = append(operations, updateOperation)

	// serialize operations to JSON
	patch, err := json.Marshal(operations)
	if err != nil {
		return nil, err
	}

	return patch, nil
}

// GenerateJSONPatch generates a JSON patch to add or update the label 'cumulo.ai' on a Kubernetes Pod object
func GenerateJSONPatchPrev4(pod *corev1.Pod) ([]byte, error) {
	//create a Variable to hold the JSON patch
	var jsonPatch []byte

	//check if the 'cumulo.ai' label exists in the pod object
	_, exists := pod.ObjectMeta.Labels["cumulo.ai"]
	if exists {
		//if the label exists, toggle its value to false
		pod.ObjectMeta.Labels["cumulo.ai"] = "false"
	} else {
		//if the label doesn't exist, set its value to true
		pod.ObjectMeta.Labels["cumulo.ai"] = "true"
	}

	//create an array of operations for the JSON patch
	var operations []struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}

	//create an operation for adding the 'cumulo.ai' label to the pod
	addOperation := struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{Op: "add", Path: "/metadata/labels/cumulo.ai", Value: pod.ObjectMeta.Labels["cumulo.ai"]}

	//append the operation to the array of operations
	operations = append(operations, addOperation)

	//encode the array of operations to JSON format
	jsonPatch, err := json.Marshal(operations)
	if err != nil {
		return nil, err
	}

	//return the JSON patch
	return jsonPatch, nil
}

/*
"\"I need a Go function \"func GenerateJSONPatch(pod *corev1.Pod) ([]byte, error) \" that generates a JSON patch to add or update a specific label ('cumulo.ai') on a Kubernetes Pod object. The label should be set to 'true' if it doesn't exist and toggled to 'false' if it already exists. The input param should be pointer to Group Version POD object. Create a JSON path as an array of operations and return a byte slice containing the JSON patch. Each operation in the patch should be represented by a struct with fields Op, Path, and Value of type string, serialized as \"op\", \"path\", and \"value\" respectively in the JSON output.  I want only function definition, do not include comments , text, main(), examples, note\n"
*/

func GenerateJSONPatchPrev5(pod *corev1.Pod) ([]byte, error) {

	// check if label already exists on the pod
	if _, ok := pod.Labels["cumulo.ai"]; ok {
		// toggle the value to "false"
		pod.Labels["cumulo.ai"] = "false"
	} else {
		// add label with value "true"
		pod.Labels["cumulo.ai"] = "true"
	}

	// create array of operations for the patch
	ops := []struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{
		{
			Op:    "replace",
			Path:  "/metadata/labels/cumulo.ai",
			Value: pod.Labels["cumulo.ai"],
		},
	}

	// serialize array of operations to JSON
	patch, err := json.Marshal(ops)
	if err != nil {
		return nil, err
	}

	return patch, nil
}

/*
"\"I need a Go function that generates a JSON patch to add or update a specific label ('aulabs.ai') on a Kubernetes Pod object. The label should be set to 'true' if it doesn't exist and toggled to 'false' if it already exists. The input param should be pointer to Group Version POD object. Create a JSON path as an array of operations and return a byte slice containing the JSON patch, error. Each operation in the patch should be represented by a struct with fields Op, Path, and Value of type string, serialized as \"op\", \"path\", and \"value\" respectively in the JSON output.  I want only function definition, strictly do not include comments , text, main(), examples, note. Also do not have unused variables in the function\n"
*/
func GenerateJSONPatch(pod *corev1.Pod) ([]byte, error) {
	// check if label exists
	if _, ok := pod.Labels["cumulo.ai"]; ok {
		// if label exists, toggle to false
		pod.Labels["cumulo.ai"] = "false"
	} else {
		// label doesn't exist, set to true
		pod.Labels["cumulo.ai"] = "true"
	}

	// create list of operations for JSON patch
	patch := []struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{
		{Op: "add", Path: "/metadata/labels/cumulo.ai", Value: pod.Labels["cumulo.ai"]},
	}

	// convert patch to JSON byte slice
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}

	return patchJSON, nil
}

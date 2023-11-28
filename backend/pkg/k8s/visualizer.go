package k8s

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VisualizationData represents the structure of network policy and pod data for visualization.
type VisualizationData struct {
	Policies []PolicyVisualization `json:"policies"`
}

// PolicyVisualization represents a network policy and the pods it affects for visualization purposes.
type PolicyVisualization struct {
	Name       string   `json:"name"`
	TargetPods []string `json:"targetPods"`
}

// gatherVisualizationData retrieves network policies and associated pods for visualization.
func gatherVisualizationData(namespace string) (*VisualizationData, error) {
	clientset, err := GetClientset()
	if err != nil {
		return nil, err
	}

	// Retrieve all network policies in the specified namespace
	policies, err := clientset.NetworkingV1().NetworkPolicies(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	vizData := &VisualizationData{
		Policies: make([]PolicyVisualization, 0), // Initialize as empty slice
	}

	// Iterate over the retrieved policies to build the visualization data
	for _, policy := range policies.Items {
		// For each policy, find the pods that match its pod selector
		selector, err := metav1.LabelSelectorAsSelector(&policy.Spec.PodSelector)
		if err != nil {
			log.Printf("Error parsing selector for policy %s: %v\n", policy.Name, err)
			continue
		}

		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			log.Printf("Error listing pods for policy %s: %v\n", policy.Name, err)
			continue
		}

		podNames := make([]string, 0, len(pods.Items))
		for _, pod := range pods.Items {
			podNames = append(podNames, pod.Name)
		}

		vizData.Policies = append(vizData.Policies, PolicyVisualization{
			Name:       policy.Name,
			TargetPods: podNames,
		})
	}

	return vizData, nil
}

// HandleVisualizationRequest handles the HTTP request for serving visualization data.
func HandleVisualizationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")

	vizData, err := gatherVisualizationData(namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(vizData); err != nil {
		http.Error(w, "Failed to encode visualization data", http.StatusInternalServerError)
	}
}

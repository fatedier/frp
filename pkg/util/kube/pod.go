package kube

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func LabelPodWithCustomDomain(customDomain string) error {
	if len(customDomain) == 0 {
		return fmt.Errorf("no custom domain provided for labeling pod")
	}

	podName := os.Getenv("HOSTNAME")
	if podName == "" {
		return fmt.Errorf("unable to get pod name via environment variable") // No pod name available, cannot label
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		return fmt.Errorf("unable to get pod namespace via environment variable") // No pod namespace available, cannot label
	}

	patchData := fmt.Sprintf(`{"metadata":{"labels":{"%s":"%s"}}}`, customDomain, "true")

	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods(namespace).Patch(
		context.TODO(),
		podName,
		types.MergePatchType,
		[]byte(patchData),
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("error patching pod %s in namespace %s: %v", podName, namespace, err)
	}

	return nil
}

func RemoveCustomDomainLabelFromPod(customDomain string) error {
	if len(customDomain) == 0 {
		return fmt.Errorf("no custom domain provided for removing pod label	")
	}

	podName := os.Getenv("HOSTNAME")
	namespace := os.Getenv("POD_NAMESPACE")

	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", customDomain),
	})
	if err != nil {
		return fmt.Errorf("error listing pods with label %s=true: %v", customDomain, err)
	}

	if len(podList.Items) == 0 {
		return fmt.Errorf("no pod found with label %s=true", customDomain)
	}

	if len(podList.Items) == 1 {
		return fmt.Errorf("only one pod found with label %s=true, skipping label removal", customDomain)
	}

	// Create JSON patch to remove the label
	patchData := fmt.Sprintf(`[{"op": "remove", "path": "/metadata/labels/%s"}]`, customDomain)

	// Apply the patch
	_, err = clientset.CoreV1().Pods(namespace).Patch(
		context.TODO(),
		podName,
		types.JSONPatchType,
		[]byte(patchData),
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("error patching pod %s in namespace %s: %v", podName, namespace, err)
	}

	return nil
}

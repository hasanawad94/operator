package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildv1 "github.com/openshift/api/build/v1"
	"github.com/redhat-openshift-builds/operator/test/utils"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Entitlements for OpenShift builds operator", Label("e2e"), Label("entitlements"), func() {

	testNamespace := "builds-test"
	It("Should access entitled rhel content", func(ctx SpecContext) {
		By("Waiting for the etc-pki-entitlement secret to exist in openshift-config-managed namespace")
		err := entitlementSecretExists(ctx, "etc-pki-entitlement", 5*time.Minute)
		Expect(err).NotTo(HaveOccurred())

		By("Creating a pod to mount the entitlement secret and verify its contents")
		projectDir, err := utils.GetProjectDir()
		Expect(err).NotTo(HaveOccurred(), "getting project directory")
		resourcePath := projectDir + "/test/data/entitlement-test-pod.yaml"
		err = utils.ApplyResourceFromFile(ctx, kubeClient, resourcePath)
		Expect(err).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")

		By("Waiting for the pod to complete")
		podName := "etc-pki-entitlement-test"
		err = wait.PollUntilContextCancel(ctx, 5*time.Second, true,
			func(ctx context.Context) (done bool, err error) {
				pod := &corev1.Pod{}
				err = kubeClient.Get(ctx, client.ObjectKey{Namespace: "openshift-config-managed",Name: podName}, pod)
				if err != nil {
					// Some other error occurred, stop retrying
					return false, err
				}
				// Pod found, check if it has completed
				if pod.Status.Phase == corev1.PodSucceeded {
					return true, nil
				}
				if pod.Status.Phase == corev1.PodFailed {
					return false, err
				}
				// Pod is still running, retry
				return false, nil
			})
		Expect(err).NotTo(HaveOccurred(), "waiting for pod to complete")
	})

	It("Should access entitled content in build", func(ctx SpecContext) {
        By("Creating a BuildConfig to verify entitled content")
        projectDir, err := utils.GetProjectDir()
        Expect(err).NotTo(HaveOccurred(), "getting project directory")
        resourcePath := projectDir + "/test/data/build-config.yaml"
        err = utils.ApplyResourceFromFile(ctx, kubeClient, resourcePath)
        Expect(err).NotTo(HaveOccurred(), "applying build-config.yaml")
    
        By("Waiting for the BuildConfig to be created")
        resourceName := "my-csi-bc-s2i"
        err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
            buildConfigRef := &buildv1.BuildConfig{}
            err := kubeClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: resourceName}, buildConfigRef)
            if errors.IsNotFound(err) {
                // BuildConfig not found, retry
                return false, nil
            } else if err != nil {
                // Some other error occurred, stop retrying
                return false, err
            }
            // BuildConfig found
            return true, nil
        })
        Expect(err).NotTo(HaveOccurred(), "waiting for BuildConfig to be created")
    
        By("Ensure BuildConfig is complete")
        err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
            buildConfigRef := &buildv1.BuildConfig{}
            err := kubeClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: resourceName}, buildConfigRef)
            if err != nil {
                return false, err
            }
            // Check if the BuildConfig has completed
            if buildConfigRef.Status.LastVersion == 1 { // Assuming 1 indicates completion
                return true, nil
            }
            return false, nil
        })
        Expect(err).NotTo(HaveOccurred(), "waiting for BuildConfig to complete")
    })

})

/*
// getPodLogs retrieves the logs of a given pod
func getPodLogs(ctx context.Context, kubeClient client.Client, namespace, podName string) (string, error) {
    podLogOpts := &corev1.PodLogOptions{}
    req := kubeClient.RESTClient().Get().
        Namespace(namespace).
        Name(podName).
        Resource("pods").
        SubResource("log").
        VersionedParams(podLogOpts, client.ParameterCodec)

    logs, err := req.Do(ctx).Raw()
    if err != nil {
        return "", fmt.Errorf("failed to get logs for pod %s: %w", podName, err)
    }
    return string(logs), nil
}*/

func entitlementSecretExists(ctx context.Context, secretName string, timeout time.Duration) error {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Name:      secretName,
		Namespace: "openshift-config-managed",
	}
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, timeout, true,
		func(ctx context.Context) (done bool, err error) {
			err = kubeClient.Get(ctx, secretKey, secret)
			if errors.IsNotFound(err) {
				// Secret not found, retry
				return false, nil
			} else if err != nil {
				// Some other error occurred, stop retrying
				return false, err
			}
			// Secret found
			return true, nil
		})
	return err
}

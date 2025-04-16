package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/redhat-openshift-builds/operator/test/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Entitlements for OpenShift builds operator", Label("e2e"), Label("entitlements"), func() {

	AfterEach(func() {
		podToDel := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "openshift-config-managed",
				Name:      "etc-pki-entitlement-test",
			},
		}
		By("Cleaning up resources before each test")
		_ = kubeClient.Delete(context.Background(), podToDel)
	})

	It("Should access entitled rhel content", func(ctx SpecContext) {
		By("Ensure etc-pki-entitlement secret exists")
		secret := &corev1.Secret{}
		secretKey := client.ObjectKey{
			Name:      "etc-pki-entitlement",
			Namespace: "openshift-config-managed",
		}
		Eventually(func() error {
			return kubeClient.Get(ctx, secretKey, secret)
		}, 10*time.Second).Should(Succeed())

		By("Creating a pod to use the entitlement secret")
		projectDir, err := utils.GetProjectDir()
		Expect(err).NotTo(HaveOccurred(), "getting project directory")
		resourcePath := projectDir + "/test/data/entitlement-test-pod.yaml"
		err = utils.ApplyResourceFromFile(ctx, kubeClient, resourcePath)
		Expect(err).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")

		By("Waiting for the pod to complete")
		podName := "etc-pki-entitlement-test"
		Eventually(func() error {
			pod := &corev1.Pod{}
			err = kubeClient.Get(ctx, client.ObjectKey{Namespace: "openshift-config-managed", Name: podName}, pod)
			if err != nil {
				return err
			}
			if pod.Status.Phase == corev1.PodSucceeded {
				return nil
			} else if pod.Status.Phase == corev1.PodFailed {
				return fmt.Errorf("pod %s failed", podName)
			} else {
				return fmt.Errorf("pod %s is not yet completed, current phase: %s", podName, pod.Status.Phase)
			}
		}, 5*time.Minute, 15*time.Second).Should(Succeed())

	})
})

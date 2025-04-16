package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/redhat-openshift-builds/operator/test/utils"
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Entitlements for OpenShift builds operator", Label("e2e"), Label("entitlements"), func() {

	Context("When testing entitlement access via a pod", func() {
		var projectDir string
		var podName string

		BeforeEach(func() {
			By("Setting up test environment for pod-based entitlement access")
			var err error
			projectDir, err = utils.GetProjectDir()
			Expect(err).NotTo(HaveOccurred(), "getting project directory")
			podName = "etc-pki-entitlement-test"
		})

		AfterEach(func() {
			By("Cleaning up resources after pod-based entitlement access test")
			podToDel := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "openshift-config-managed",
					Name:      podName,
				},
			}
			Expect(kubeClient.Delete(context.Background(), podToDel)).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")
		})

		It("Should access entitled RHEL content", func(ctx SpecContext) {
			By("Ensuring etc-pki-entitlement secret exists")
			secretKey := client.ObjectKey{
				Name:      "etc-pki-entitlement",
				Namespace: "openshift-config-managed",
			}
			Eventually(func() error {
				return kubeClient.Get(ctx, secretKey, &corev1.Secret{})
			}, 10*time.Second).Should(Succeed())

			By("Creating a pod to use the entitlement secret")
			resourcePath := projectDir + "/test/data/entitlement-test-pod.yaml"
			Expect(utils.ApplyResourceFromFile(ctx, kubeClient, resourcePath)).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")

			By("Waiting for the pod to complete")
			Eventually(func() error {
				pod := &corev1.Pod{}
				err := kubeClient.Get(ctx, client.ObjectKey{Namespace: "openshift-config-managed", Name: podName}, pod)
				if err != nil {
					return err
				}
				switch pod.Status.Phase {
				case corev1.PodSucceeded:
					By("Entitled access in pod completed successfully")
					return nil
				case corev1.PodFailed:
					return fmt.Errorf("pod %s failed", podName)
				default:
					return fmt.Errorf("pod %s is not yet completed, current phase: %s", podName, pod.Status.Phase)
				}
			}, 5*time.Minute, 10*time.Second).Should(Succeed(), fmt.Sprintf("waiting for pod %s to complete", podName))
		})
	})

	Context("When testing entitlement access with shared secret", func() {
		var projectDir string
		var resources []string

		BeforeEach(func() {
			By("Setting up test environment for shared secret-based entitlement access")
			var err error
			projectDir, err = utils.GetProjectDir()
			Expect(err).NotTo(HaveOccurred(), "getting project directory")

			resources = []string{
				projectDir + "/test/data/image-stream.yaml",
				projectDir + "/test/data/shared-secret.yaml",
				projectDir + "/test/data/shared-secret-cluster-role.yaml",
				projectDir + "/test/data/shared-secret-csi-role.yaml",
				projectDir + "/test/data/csi-driver-role-bind.yaml",
				projectDir + "/test/data/pipeline-builder-role-bind.yaml",
				projectDir + "/test/data/entitled-build.yaml",
			}
		})

		AfterEach(func(ctx SpecContext) {
			By("Cleaning up resources after shared secret-based entitlement access test")
			for _, toDel := range resources {
				Expect(utils.DeleteResourceFromFile(ctx, kubeClient, toDel)).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")
			}
			err := utils.DeleteResourceFromFile(ctx, kubeClient, projectDir+"/test/data/entitled-buildrun.yaml")
			Expect(err).NotTo(HaveOccurred(), "applying entitlement-test-pod.yaml")
		})

		It("Should access entitled content using shared secret", func(ctx SpecContext) {
			By("Applying RBAC, shared secret, build resources")
			for _, resource := range resources {
				Expect(utils.ApplyResourceFromFile(ctx, kubeClient, resource)).NotTo(HaveOccurred(), fmt.Sprintf("applying resource: %s", resource))
			}

			By("Applying buildrun")
			entitledBuildRun := projectDir + "/test/data/entitled-buildrun.yaml"
			Expect(utils.ApplyResourceFromFile(ctx, kubeClient, entitledBuildRun)).NotTo(HaveOccurred(), "applying entitled-buildrun.yaml")

			By("Waiting for buildrun to complete")
			Eventually(func() error {
				buildRun := &buildv1beta1.BuildRun{}
				err := kubeClient.Get(ctx, client.ObjectKey{Namespace: "builds-test", Name: "entitled-br"}, buildRun)
				if err != nil {
					return fmt.Errorf("failed to get buildrun entitled-br: %w", err)
				}
				if !buildRun.IsDone() {
					return fmt.Errorf("BuildRun 'entitled-br' is not yet completed")
				}

				if buildRun.IsSuccessful() {
					By("BuildRun 'entitled-br' completed successfully")
					return nil
				}

				return fmt.Errorf("BuildRun 'entitled-br' failed with reason: %s, message: %s",
					buildRun.Status.FailureDetails.Reason, buildRun.Status.FailureDetails.Message)
			}, 5*time.Minute, 15*time.Second).Should(Succeed())
		})
	})
})

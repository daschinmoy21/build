// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

func sampleBetaBuild() *buildapi.Build {
	kind := buildapi.ClusterBuildStrategyKind
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-build",
			Namespace:   "build-ns",
			Labels:      map[string]string{"app": "test"},
			Annotations: map[string]string{"note": "hi"},
		},
		Spec: buildapi.BuildSpec{
			Source: &buildapi.Source{
				Type:       buildapi.GitType,
				ContextDir: ptr.To("src"),
				Git: &buildapi.Git{
					URL:         "https://github.com/shipwright-io/sample",
					Revision:    ptr.To("main"),
					CloneSecret: ptr.To("git-secret"),
				},
			},
			Strategy: buildapi.Strategy{
				Name: "kaniko",
				Kind: &kind,
			},
			ParamValues: []buildapi.ParamValue{{
				Name: "my-param",
				SingleValue: &buildapi.SingleValue{
					Value: ptr.To("my-value"),
				},
			}},
			Output: buildapi.Image{
				Image:       "quay.io/example/app:latest",
				Insecure:    ptr.To(false),
				PushSecret:  ptr.To("push-secret"),
				Annotations: map[string]string{"org.opencontainers.image.source": "git"},
				Labels:      map[string]string{"app": "example"},
				Timestamp:   ptr.To(buildapialpha.OutputImageZeroTimestamp),
			},
			Timeout: &metav1.Duration{Duration: 10 * time.Minute},
			Env:     []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
			Retention: &buildapi.BuildRetention{
				FailedLimit:       ptr.To[uint](3),
				SucceededLimit:    ptr.To[uint](5),
				TTLAfterFailed:    &metav1.Duration{Duration: time.Hour},
				TTLAfterSucceeded: &metav1.Duration{Duration: 2 * time.Hour},
			},
			Volumes: []buildapi.BuildVolume{{
				Name:         "cache",
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			}},
			Trigger: &buildapi.Trigger{
				TriggerSecret: ptr.To("webhook-secret"),
				When: []buildapi.TriggerWhen{{
					Name: "on-push",
					Type: buildapi.GitHubWebHookTrigger,
					GitHub: &buildapi.WhenGitHub{
						Events:   []buildapi.GitHubEventName{buildapi.GitHubPushEvent},
						Branches: []string{"main"},
					},
				}},
			},
		},
		Status: buildapi.BuildStatus{
			Registered: ptr.To(corev1.ConditionTrue),
			Reason:     ptr.To(buildapi.SucceedStatus),
			Message:    ptr.To(buildapi.AllValidationsSucceeded),
		},
	}
}

func alphaBuildFromUnstructured(u *unstructured.Unstructured) buildapialpha.Build {
	var alpha buildapialpha.Build
	Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &alpha)).To(Succeed())
	return alpha
}

var _ = Describe("Build conversion", func() {
	Context("ConvertTo (beta -> alpha)", func() {
		It("preserves meta, git source, strategy, output, and status", func() {
			beta := sampleBetaBuild()
			beta.Kind = "Build"
			beta.APIVersion = "shipwright.io/v1beta1"

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			Expect(u.GetAPIVersion()).To(Equal("shipwright.io/v1alpha1"))
			Expect(u.GetKind()).To(Equal("Build"))

			alpha := alphaBuildFromUnstructured(u)

			Expect(alpha.Name).To(Equal("my-build"))
			Expect(alpha.Namespace).To(Equal("build-ns"))
			Expect(alpha.Labels).To(Equal(map[string]string{"app": "test"}))
			Expect(alpha.Annotations).To(Equal(map[string]string{"note": "hi"}))

			Expect(alpha.Spec.Source.URL).To(Equal(ptr.To("https://github.com/shipwright-io/sample")))
			Expect(alpha.Spec.Source.Revision).To(Equal(ptr.To("main")))
			Expect(alpha.Spec.Source.ContextDir).To(Equal(ptr.To("src")))
			Expect(alpha.Spec.Source.Credentials).To(Equal(&corev1.LocalObjectReference{Name: "git-secret"}))
			Expect(alpha.Spec.Source.BundleContainer).To(BeNil())
			Expect(alpha.Spec.Sources).To(BeEmpty())

			Expect(alpha.Spec.Strategy.Name).To(Equal("kaniko"))
			Expect(alpha.Spec.Strategy.Kind).To(Equal(ptr.To(buildapialpha.ClusterBuildStrategyKind)))

			Expect(alpha.Spec.ParamValues).To(HaveLen(1))
			Expect(alpha.Spec.ParamValues[0].Name).To(Equal("my-param"))
			Expect(alpha.Spec.ParamValues[0].Value).To(Equal(ptr.To("my-value")))

			Expect(alpha.Spec.Output.Image).To(Equal("quay.io/example/app:latest"))
			Expect(alpha.Spec.Output.Insecure).To(Equal(ptr.To(false)))
			Expect(alpha.Spec.Output.Credentials).To(Equal(&corev1.LocalObjectReference{Name: "push-secret"}))
			Expect(alpha.Spec.Output.Annotations).To(Equal(map[string]string{"org.opencontainers.image.source": "git"}))
			Expect(alpha.Spec.Output.Labels).To(Equal(map[string]string{"app": "example"}))
			Expect(alpha.Spec.Output.Timestamp).To(Equal(ptr.To(buildapialpha.OutputImageZeroTimestamp)))

			Expect(alpha.Spec.Timeout).To(Equal(&metav1.Duration{Duration: 10 * time.Minute}))
			Expect(alpha.Spec.Env).To(Equal([]corev1.EnvVar{{Name: "FOO", Value: "bar"}}))
			Expect(alpha.Spec.Retention).ToNot(BeNil())
			Expect(alpha.Spec.Retention.FailedLimit).To(Equal(ptr.To[uint](3)))
			Expect(alpha.Spec.Retention.SucceededLimit).To(Equal(ptr.To[uint](5)))
			Expect(alpha.Spec.Volumes).To(HaveLen(1))
			Expect(alpha.Spec.Volumes[0].Name).To(Equal("cache"))
			Expect(alpha.Spec.Volumes[0].EmptyDir).ToNot(BeNil())

			Expect(alpha.Spec.Trigger).ToNot(BeNil())
			Expect(alpha.Spec.Trigger.SecretRef).To(Equal(&corev1.LocalObjectReference{Name: "webhook-secret"}))
			Expect(alpha.Spec.Trigger.When).To(HaveLen(1))
			Expect(alpha.Spec.Trigger.When[0].Name).To(Equal("on-push"))
			Expect(alpha.Spec.Trigger.When[0].GitHub.Events).To(Equal([]buildapialpha.GitHubEventName{buildapialpha.GitHubPushEvent}))
			Expect(alpha.Spec.Trigger.When[0].GitHub.Branches).To(Equal([]string{"main"}))

			Expect(alpha.Status.Registered).To(Equal(ptr.To(corev1.ConditionTrue)))
			Expect(alpha.Status.Reason).ToNot(BeNil())
			Expect(*alpha.Status.Reason).To(Equal(buildapialpha.SucceedStatus))
			Expect(alpha.Status.Message).To(Equal(ptr.To(buildapi.AllValidationsSucceeded)))
		})

		It("maps OCI source to bundleContainer", func() {
			beta := sampleBetaBuild()
			beta.Kind = "Build"
			beta.APIVersion = "shipwright.io/v1beta1"
			prune := buildapi.PruneAfterPull
			beta.Spec.Source = &buildapi.Source{
				Type:       buildapi.OCIArtifactType,
				ContextDir: ptr.To("app"),
				OCIArtifact: &buildapi.OCIArtifact{
					Image:      "quay.io/example/src:1",
					Prune:      &prune,
					PullSecret: ptr.To("oci-secret"),
				},
			}

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaBuildFromUnstructured(u)
			Expect(alpha.Spec.Source.URL).To(BeNil())
			Expect(alpha.Spec.Source.BundleContainer).ToNot(BeNil())
			Expect(alpha.Spec.Source.BundleContainer.Image).To(Equal("quay.io/example/src:1"))
			Expect(alpha.Spec.Source.BundleContainer.Prune).To(Equal(ptr.To(buildapialpha.PruneAfterPull)))
			Expect(alpha.Spec.Source.Credentials).To(Equal(&corev1.LocalObjectReference{Name: "oci-secret"}))
			Expect(alpha.Spec.Source.ContextDir).To(Equal(ptr.To("app")))
		})

		It("maps Local source to alpha Sources", func() {
			beta := sampleBetaBuild()
			beta.Kind = "Build"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Spec.Source = &buildapi.Source{
				Type:       buildapi.LocalType,
				ContextDir: ptr.To("workdir"),
				Local: &buildapi.Local{
					Name:    "upload",
					Timeout: &metav1.Duration{Duration: 5 * time.Minute},
				},
			}

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaBuildFromUnstructured(u)
			Expect(alpha.Spec.Sources).To(HaveLen(1))
			Expect(alpha.Spec.Sources[0].Name).To(Equal("upload"))
			Expect(alpha.Spec.Sources[0].Type).To(Equal(buildapialpha.LocalCopy))
			Expect(alpha.Spec.Sources[0].Timeout).To(Equal(&metav1.Duration{Duration: 5 * time.Minute}))
		})

		It("maps dockerfile and builder-image params to deprecated fields", func() {
			beta := sampleBetaBuild()
			beta.Kind = "Build"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Spec.ParamValues = append(beta.Spec.ParamValues,
				buildapi.ParamValue{
					Name: "dockerfile",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("Dockerfile.prod"),
					},
				},
				buildapi.ParamValue{
					Name: "builder-image",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("golang:1.22"),
					},
				},
			)

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaBuildFromUnstructured(u)
			Expect(alpha.Spec.Dockerfile).To(Equal(ptr.To("Dockerfile.prod")))
			Expect(alpha.Spec.Builder).ToNot(BeNil())
			Expect(alpha.Spec.Builder.Image).To(Equal("golang:1.22"))
			Expect(alpha.Spec.ParamValues).To(HaveLen(1))
			Expect(alpha.Spec.ParamValues[0].Name).To(Equal("my-param"))
		})

		It("maps AtBuildDeletion to the build-run-deletion annotation", func() {
			beta := sampleBetaBuild()
			beta.Kind = "Build"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Spec.Retention.AtBuildDeletion = ptr.To(true)

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaBuildFromUnstructured(u)
			Expect(alpha.Annotations).To(HaveKeyWithValue(buildapialpha.AnnotationBuildRunDeletion, "true"))
			Expect(alpha.Annotations).To(HaveKeyWithValue("note", "hi"))
		})
	})

	Context("ConvertFrom (alpha -> beta)", func() {
		It("preserves meta, git source, strategy, output, and status", func() {
			kind := buildapialpha.ClusterBuildStrategyKind
			alpha := &buildapialpha.Build{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-build",
					Namespace:   "build-ns",
					Labels:      map[string]string{"app": "test"},
					Annotations: map[string]string{"note": "hi"},
				},
				Spec: buildapialpha.BuildSpec{
					Source: buildapialpha.Source{
						URL:        ptr.To("https://github.com/shipwright-io/sample"),
						Revision:   ptr.To("main"),
						ContextDir: ptr.To("src"),
						Credentials: &corev1.LocalObjectReference{
							Name: "git-secret",
						},
					},
					Strategy: buildapialpha.Strategy{
						Name: "kaniko",
						Kind: &kind,
					},
					ParamValues: []buildapialpha.ParamValue{{
						Name: "my-param",
						SingleValue: &buildapialpha.SingleValue{
							Value: ptr.To("my-value"),
						},
					}},
					Output: buildapialpha.Image{
						Image: "quay.io/example/app:latest",
						Credentials: &corev1.LocalObjectReference{
							Name: "push-secret",
						},
					},
					Timeout: &metav1.Duration{Duration: 10 * time.Minute},
					Env:     []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
					Retention: &buildapialpha.BuildRetention{
						FailedLimit: ptr.To[uint](3),
					},
					Volumes: []buildapialpha.BuildVolume{{
						Name:         "cache",
						VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
					}},
					Trigger: &buildapialpha.Trigger{
						SecretRef: &corev1.LocalObjectReference{Name: "webhook-secret"},
						When: []buildapialpha.TriggerWhen{{
							Name: "on-push",
							Type: buildapialpha.GitHubWebHookTrigger,
							GitHub: &buildapialpha.WhenGitHub{
								Events:   []buildapialpha.GitHubEventName{buildapialpha.GitHubPushEvent},
								Branches: []string{"main"},
							},
						}},
					},
				},
				Status: buildapialpha.BuildStatus{
					Registered: ptr.To(corev1.ConditionTrue),
					Reason:     ptr.To(buildapialpha.SucceedStatus),
					Message:    ptr.To(buildapialpha.AllValidationsSucceeded),
				},
			}
			alpha.Kind = "Build"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.Build{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.APIVersion).To(Equal("shipwright.io/v1beta1"))
			Expect(beta.Name).To(Equal("my-build"))
			Expect(beta.Namespace).To(Equal("build-ns"))
			Expect(beta.Labels).To(Equal(map[string]string{"app": "test"}))

			Expect(beta.Spec.Source).ToNot(BeNil())
			Expect(beta.Spec.Source.Type).To(Equal(buildapi.GitType))
			Expect(beta.Spec.Source.Git.URL).To(Equal("https://github.com/shipwright-io/sample"))
			Expect(beta.Spec.Source.Git.Revision).To(Equal(ptr.To("main")))
			Expect(beta.Spec.Source.Git.CloneSecret).To(Equal(ptr.To("git-secret")))
			Expect(beta.Spec.Source.ContextDir).To(Equal(ptr.To("src")))

			Expect(beta.Spec.Strategy.Name).To(Equal("kaniko"))
			Expect(beta.Spec.Strategy.Kind).To(Equal(ptr.To(buildapi.ClusterBuildStrategyKind)))
			Expect(beta.Spec.ParamValues).To(HaveLen(1))
			Expect(beta.Spec.Output.Image).To(Equal("quay.io/example/app:latest"))
			Expect(beta.Spec.Output.PushSecret).To(Equal(ptr.To("push-secret")))
			Expect(beta.Spec.Timeout).To(Equal(&metav1.Duration{Duration: 10 * time.Minute}))
			Expect(beta.Spec.Retention.FailedLimit).To(Equal(ptr.To[uint](3)))
			Expect(beta.Spec.Volumes).To(HaveLen(1))
			Expect(beta.Spec.Trigger.TriggerSecret).To(Equal(ptr.To("webhook-secret")))
			Expect(beta.Spec.Trigger.When[0].GitHub.Branches).To(Equal([]string{"main"}))

			Expect(beta.Status.Registered).To(Equal(ptr.To(corev1.ConditionTrue)))
			Expect(beta.Status.Reason).To(Equal(ptr.To(buildapi.SucceedStatus)))
			Expect(beta.Status.Message).To(Equal(ptr.To(buildapi.AllValidationsSucceeded)))
		})

		It("maps bundleContainer to OCI source", func() {
			alpha := &buildapialpha.Build{
				ObjectMeta: metav1.ObjectMeta{Name: "my-build", Namespace: "build-ns"},
				Spec: buildapialpha.BuildSpec{
					Source: buildapialpha.Source{
						ContextDir: ptr.To("app"),
						BundleContainer: &buildapialpha.BundleContainer{
							Image: "quay.io/example/src:1",
							Prune: ptr.To(buildapialpha.PruneAfterPull),
						},
						Credentials: &corev1.LocalObjectReference{Name: "oci-secret"},
					},
					Strategy: buildapialpha.Strategy{Name: "kaniko"},
					Output:   buildapialpha.Image{Image: "quay.io/example/app:latest"},
				},
			}
			alpha.Kind = "Build"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.Build{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.Spec.Source.Type).To(Equal(buildapi.OCIArtifactType))
			Expect(beta.Spec.Source.OCIArtifact.Image).To(Equal("quay.io/example/src:1"))
			Expect(beta.Spec.Source.OCIArtifact.Prune).To(Equal(ptr.To(buildapi.PruneAfterPull)))
			Expect(beta.Spec.Source.OCIArtifact.PullSecret).To(Equal(ptr.To("oci-secret")))
			Expect(beta.Spec.Source.ContextDir).To(Equal(ptr.To("app")))
		})

		It("maps LocalCopy Sources to Local source", func() {
			alpha := &buildapialpha.Build{
				ObjectMeta: metav1.ObjectMeta{Name: "my-build", Namespace: "build-ns"},
				Spec: buildapialpha.BuildSpec{
					Source: buildapialpha.Source{ContextDir: ptr.To("workdir")},
					Sources: []buildapialpha.BuildSource{{
						Name:    "upload",
						Type:    buildapialpha.LocalCopy,
						Timeout: &metav1.Duration{Duration: 5 * time.Minute},
					}},
					Strategy: buildapialpha.Strategy{Name: "kaniko"},
					Output:   buildapialpha.Image{Image: "quay.io/example/app:latest"},
				},
			}
			alpha.Kind = "Build"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.Build{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.Spec.Source.Type).To(Equal(buildapi.LocalType))
			Expect(beta.Spec.Source.Local.Name).To(Equal("upload"))
			Expect(beta.Spec.Source.Local.Timeout).To(Equal(&metav1.Duration{Duration: 5 * time.Minute}))
			Expect(beta.Spec.Source.ContextDir).To(Equal(ptr.To("workdir")))
		})

		It("maps deprecated Dockerfile and Builder fields to paramValues", func() {
			alpha := &buildapialpha.Build{
				ObjectMeta: metav1.ObjectMeta{Name: "my-build", Namespace: "build-ns"},
				Spec: buildapialpha.BuildSpec{
					Source:     buildapialpha.Source{URL: ptr.To("https://github.com/shipwright-io/sample")},
					Strategy:   buildapialpha.Strategy{Name: "kaniko"},
					Dockerfile: ptr.To("Dockerfile.prod"),
					Builder:    &buildapialpha.Image{Image: "golang:1.22"},
					Output:     buildapialpha.Image{Image: "quay.io/example/app:latest"},
				},
			}
			alpha.Kind = "Build"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.Build{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.Spec.ParamValues).To(ConsistOf(
				buildapi.ParamValue{
					Name: "dockerfile",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("Dockerfile.prod"),
					},
				},
				buildapi.ParamValue{
					Name: "builder-image",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("golang:1.22"),
					},
				},
			))
		})

		It("maps the build-run-deletion annotation to AtBuildDeletion", func() {
			alpha := &buildapialpha.Build{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-build",
					Namespace: "build-ns",
					Annotations: map[string]string{
						"note":                                   "hi",
						buildapialpha.AnnotationBuildRunDeletion: "true",
					},
				},
				Spec: buildapialpha.BuildSpec{
					Source:   buildapialpha.Source{URL: ptr.To("https://github.com/shipwright-io/sample")},
					Strategy: buildapialpha.Strategy{Name: "kaniko"},
					Output:   buildapialpha.Image{Image: "quay.io/example/app:latest"},
				},
			}
			alpha.Kind = "Build"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.Build{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.Spec.Retention).ToNot(BeNil())
			Expect(beta.Spec.Retention.AtBuildDeletion).To(Equal(ptr.To(true)))
			Expect(beta.Annotations).To(Equal(map[string]string{"note": "hi"}))
			Expect(beta.Annotations).ToNot(HaveKey(buildapialpha.AnnotationBuildRunDeletion))
		})
	})

	It("round-trips beta -> alpha -> beta", func() {
		start := sampleBetaBuild()
		start.Kind = "Build"
		start.APIVersion = "shipwright.io/v1beta1"
		start.Spec.Retention.AtBuildDeletion = ptr.To(true)
		orig := start.DeepCopy()

		u := &unstructured.Unstructured{}
		Expect(start.ConvertTo(context.TODO(), u)).To(Succeed())

		got := &buildapi.Build{}
		Expect(got.ConvertFrom(context.TODO(), u)).To(Succeed())

		Expect(got.Name).To(Equal(orig.Name))
		Expect(got.Namespace).To(Equal(orig.Namespace))
		Expect(got.APIVersion).To(Equal("shipwright.io/v1beta1"))
		Expect(got.Spec.Source).To(Equal(orig.Spec.Source))
		Expect(got.Spec.Strategy).To(Equal(orig.Spec.Strategy))
		Expect(got.Spec.ParamValues).To(Equal(orig.Spec.ParamValues))
		Expect(got.Spec.Output).To(Equal(orig.Spec.Output))
		Expect(got.Spec.Timeout).To(Equal(orig.Spec.Timeout))
		Expect(got.Spec.Env).To(Equal(orig.Spec.Env))
		Expect(got.Spec.Volumes).To(Equal(orig.Spec.Volumes))
		Expect(got.Spec.Trigger).To(Equal(orig.Spec.Trigger))
		Expect(got.Spec.Retention.FailedLimit).To(Equal(orig.Spec.Retention.FailedLimit))
		Expect(got.Spec.Retention.SucceededLimit).To(Equal(orig.Spec.Retention.SucceededLimit))
		Expect(got.Spec.Retention.TTLAfterFailed).To(Equal(orig.Spec.Retention.TTLAfterFailed))
		Expect(got.Spec.Retention.TTLAfterSucceeded).To(Equal(orig.Spec.Retention.TTLAfterSucceeded))
		Expect(got.Spec.Retention.AtBuildDeletion).To(Equal(ptr.To(true)))
		Expect(got.Annotations).ToNot(HaveKey(buildapialpha.AnnotationBuildRunDeletion))
		Expect(got.Status).To(Equal(orig.Status))
	})
})

var _ = Describe("BuildSpec ConvertTo", func() {

	// verifies that converting a BuildSpec whose source type is OCIArtifact but
	// whose OCIArtifact field is unset does not panic, and instead produces an
	// empty BundleContainer, matching the existing nil-safe handling of Git sources
	It("does not panic when the source type is OCIArtifact but OCIArtifact is nil", func() {
		src := buildapi.BuildSpec{
			Source: &buildapi.Source{
				Type: buildapi.OCIArtifactType,
			},
		}

		dest := &buildapialpha.BuildSpec{}

		Expect(func() {
			Expect(src.ConvertTo(dest)).To(Succeed())
		}).ToNot(Panic())

		Expect(dest.Source.BundleContainer).To(BeNil())
	})
})

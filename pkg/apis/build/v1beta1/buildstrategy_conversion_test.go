// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"

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

func sampleBetaBuildStrategy() *buildapi.BuildStrategy {
	return &buildapi.BuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-bs",
			Namespace:   "build-ns",
			Labels:      map[string]string{"app": "test"},
			Annotations: map[string]string{"note": "hi"},
		},
		Spec: buildapi.BuildStrategySpec{
			Steps: []buildapi.Step{{
				Name:            "build",
				Image:           "gcr.io/kaniko:latest",
				Command:         []string{"sh"},
				Args:            []string{"-c", "echo hi"},
				WorkingDir:      "/workspace",
				Env:             []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
				ImagePullPolicy: corev1.PullIfNotPresent,
			}},
			Parameters: []buildapi.Parameter{{
				Name:        "my-param",
				Description: "a plain param",
				Type:        buildapi.ParameterTypeString,
			}},
			SecurityContext: &buildapi.BuildStrategySecurityContext{
				RunAsUser:  1000,
				RunAsGroup: 1000,
			},
			Volumes: []buildapi.BuildStrategyVolume{{
				Overridable:  ptr.To(false),
				Name:         "cache",
				Description:  ptr.To("cache volume"),
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			}},
		},
	}
}

func alphaFromUnstructured(u *unstructured.Unstructured) buildapialpha.BuildStrategy {
	var alpha buildapialpha.BuildStrategy
	Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &alpha)).To(Succeed())
	return alpha
}

var _ = Describe("BuildStrategy conversion", func() {
	Context("ConvertTo (beta -> alpha)", func() {
		It("preserves meta and spec, flips apiVersion", func() {
			beta := sampleBetaBuildStrategy()
			beta.Kind = "BuildStrategy"
			beta.APIVersion = "shipwright.io/v1beta1"

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			Expect(u.GetAPIVersion()).To(Equal("shipwright.io/v1alpha1"))
			Expect(u.GetKind()).To(Equal("BuildStrategy"))

			alpha := alphaFromUnstructured(u)

			Expect(alpha.Name).To(Equal("my-bs"))
			Expect(alpha.Namespace).To(Equal("build-ns"))
			Expect(alpha.Labels).To(Equal(map[string]string{"app": "test"}))
			Expect(alpha.Annotations).To(Equal(map[string]string{"note": "hi"}))

			Expect(alpha.Spec.BuildSteps).To(HaveLen(1))
			Expect(alpha.Spec.BuildSteps[0].Name).To(Equal("build"))
			Expect(alpha.Spec.BuildSteps[0].Image).To(Equal("gcr.io/kaniko:latest"))
			Expect(alpha.Spec.BuildSteps[0].Command).To(Equal([]string{"sh"}))
			Expect(alpha.Spec.BuildSteps[0].Args).To(Equal([]string{"-c", "echo hi"}))
			Expect(alpha.Spec.BuildSteps[0].WorkingDir).To(Equal("/workspace"))
			Expect(alpha.Spec.BuildSteps[0].Env).To(Equal([]corev1.EnvVar{{Name: "FOO", Value: "bar"}}))
			Expect(alpha.Spec.BuildSteps[0].ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))

			Expect(alpha.Spec.Parameters).To(HaveLen(1))
			Expect(alpha.Spec.Parameters[0].Name).To(Equal("my-param"))
			Expect(alpha.Spec.Parameters[0].Description).To(Equal("a plain param"))
			Expect(alpha.Spec.Parameters[0].Type).To(Equal(buildapialpha.ParameterTypeString))

			Expect(alpha.Spec.SecurityContext).ToNot(BeNil())
			Expect(alpha.Spec.SecurityContext.RunAsUser).To(Equal(int64(1000)))
			Expect(alpha.Spec.SecurityContext.RunAsGroup).To(Equal(int64(1000)))

			Expect(alpha.Spec.Volumes).To(HaveLen(1))
			Expect(alpha.Spec.Volumes[0].Name).To(Equal("cache"))
			Expect(alpha.Spec.Volumes[0].Overridable).To(Equal(ptr.To(false)))
			Expect(alpha.Spec.Volumes[0].Description).To(Equal(ptr.To("cache volume")))
			Expect(alpha.Spec.Volumes[0].EmptyDir).ToNot(BeNil())
		})

		It("drops the migrated dockerfile parameter and rewrites $(params.dockerfile)", func() {
			beta := sampleBetaBuildStrategy()
			beta.Kind = "BuildStrategy"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Spec.Parameters = append(beta.Spec.Parameters, buildapi.Parameter{
				Name:        "dockerfile",
				Description: "The Dockerfile to be built.",
				Type:        buildapi.ParameterTypeString,
				Default:     ptr.To("Dockerfile"),
			})
			beta.Spec.Steps[0].Command = []string{"build", "--file=$(params.dockerfile)"}
			beta.Spec.Steps[0].Args = []string{"$(params.dockerfile)"}
			beta.Spec.Steps[0].Env = []corev1.EnvVar{{Name: "DOCKERFILE", Value: "$(params.dockerfile)"}}

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaFromUnstructured(u)

			Expect(alpha.Spec.Parameters).To(HaveLen(1))
			Expect(alpha.Spec.Parameters[0].Name).To(Equal("my-param"))
			Expect(alpha.Spec.BuildSteps[0].Command).To(Equal([]string{"build", "--file=$(params.DOCKERFILE)"}))
			Expect(alpha.Spec.BuildSteps[0].Args).To(Equal([]string{"$(params.DOCKERFILE)"}))
			Expect(alpha.Spec.BuildSteps[0].Env[0].Value).To(Equal("$(params.DOCKERFILE)"))
		})

		It("keeps a dockerfile parameter whose default is not Dockerfile", func() {
			beta := sampleBetaBuildStrategy()
			beta.Kind = "BuildStrategy"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Spec.Parameters = append(beta.Spec.Parameters, buildapi.Parameter{
				Name:        "dockerfile",
				Description: "path to a Containerfile",
				Type:        buildapi.ParameterTypeString,
				Default:     ptr.To("Containerfile"),
			})
			beta.Spec.Steps[0].Args = []string{"$(params.dockerfile)"}

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaFromUnstructured(u)

			Expect(alpha.Spec.Parameters).To(HaveLen(2))
			Expect(alpha.Spec.Parameters[1].Name).To(Equal("dockerfile"))
			Expect(alpha.Spec.Parameters[1].Default).To(Equal(ptr.To("Containerfile")))
			Expect(alpha.Spec.BuildSteps[0].Args).To(Equal([]string{"$(params.dockerfile)"}))
		})

		It("drops the migrated builder-image parameter and rewrites $(params.builder-image)", func() {
			beta := sampleBetaBuildStrategy()
			beta.Kind = "BuildStrategy"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Spec.Parameters = append(beta.Spec.Parameters, buildapi.Parameter{
				Name:        "builder-image",
				Description: "The builder image.",
				Type:        buildapi.ParameterTypeString,
			})
			beta.Spec.Steps[0].Command = []string{"use", "$(params.builder-image)"}
			beta.Spec.Steps[0].Args = []string{"$(params.builder-image)"}
			beta.Spec.Steps[0].Env = []corev1.EnvVar{{Name: "BUILDER", Value: "$(params.builder-image)"}}

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaFromUnstructured(u)

			Expect(alpha.Spec.Parameters).To(HaveLen(1))
			Expect(alpha.Spec.Parameters[0].Name).To(Equal("my-param"))
			Expect(alpha.Spec.BuildSteps[0].Command).To(Equal([]string{"use", "$(build.builder.image)"}))
			Expect(alpha.Spec.BuildSteps[0].Args).To(Equal([]string{"$(build.builder.image)"}))
			Expect(alpha.Spec.BuildSteps[0].Env[0].Value).To(Equal("$(build.builder.image)"))
		})
	})

	Context("ConvertFrom (alpha -> beta)", func() {
		It("preserves meta and spec, flips apiVersion", func() {
			alpha := &buildapialpha.BuildStrategy{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-bs",
					Namespace:   "build-ns",
					Labels:      map[string]string{"app": "test"},
					Annotations: map[string]string{"note": "hi"},
				},
				Spec: buildapialpha.BuildStrategySpec{
					BuildSteps: []buildapialpha.BuildStep{{
						Container: corev1.Container{
							Name:            "build",
							Image:           "gcr.io/kaniko:latest",
							Command:         []string{"sh"},
							Args:            []string{"-c", "echo hi"},
							WorkingDir:      "/workspace",
							Env:             []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					}},
					Parameters: []buildapialpha.Parameter{{
						Name:        "my-param",
						Description: "a plain param",
						Type:        buildapialpha.ParameterTypeString,
					}},
					SecurityContext: &buildapialpha.BuildStrategySecurityContext{
						RunAsUser:  1000,
						RunAsGroup: 1000,
					},
					Volumes: []buildapialpha.BuildStrategyVolume{{
						Overridable:  ptr.To(false),
						Name:         "cache",
						Description:  ptr.To("cache volume"),
						VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
					}},
				},
			}
			alpha.Kind = "BuildStrategy"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.BuildStrategy{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.APIVersion).To(Equal("shipwright.io/v1beta1"))
			Expect(beta.Name).To(Equal("my-bs"))
			Expect(beta.Namespace).To(Equal("build-ns"))
			Expect(beta.Labels).To(Equal(map[string]string{"app": "test"}))

			Expect(beta.Spec.Steps).To(HaveLen(1))
			Expect(beta.Spec.Steps[0].Image).To(Equal("gcr.io/kaniko:latest"))
			Expect(beta.Spec.Steps[0].WorkingDir).To(Equal("/workspace"))
			Expect(beta.Spec.Steps[0].Env).To(Equal([]corev1.EnvVar{{Name: "FOO", Value: "bar"}}))
			Expect(beta.Spec.Steps[0].ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))

			Expect(beta.Spec.Parameters).To(HaveLen(1))
			Expect(beta.Spec.Parameters[0].Name).To(Equal("my-param"))

			Expect(beta.Spec.SecurityContext).ToNot(BeNil())
			Expect(beta.Spec.SecurityContext.RunAsUser).To(Equal(int64(1000)))
			Expect(beta.Spec.Volumes).To(HaveLen(1))
			Expect(beta.Spec.Volumes[0].Name).To(Equal("cache"))
			Expect(beta.Spec.Volumes[0].EmptyDir).ToNot(BeNil())
		})

		It("rewrites $(params.DOCKERFILE) and injects a dockerfile parameter", func() {
			alpha := &buildapialpha.BuildStrategy{
				ObjectMeta: metav1.ObjectMeta{Name: "my-bs", Namespace: "build-ns"},
				Spec: buildapialpha.BuildStrategySpec{
					BuildSteps: []buildapialpha.BuildStep{{
						Container: corev1.Container{
							Name:    "build",
							Image:   "gcr.io/kaniko:latest",
							Command: []string{"build", "--file=$(params.DOCKERFILE)"},
							Args:    []string{"$(params.DOCKERFILE)"},
							Env:     []corev1.EnvVar{{Name: "DOCKERFILE", Value: "$(params.DOCKERFILE)"}},
						},
					}},
					Parameters: []buildapialpha.Parameter{{
						Name:        "my-param",
						Description: "a plain param",
						Type:        buildapialpha.ParameterTypeString,
					}},
				},
			}
			alpha.Kind = "BuildStrategy"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.BuildStrategy{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.Spec.Steps[0].Command).To(Equal([]string{"build", "--file=$(params.dockerfile)"}))
			Expect(beta.Spec.Steps[0].Args).To(Equal([]string{"$(params.dockerfile)"}))
			Expect(beta.Spec.Steps[0].Env[0].Value).To(Equal("$(params.dockerfile)"))

			Expect(beta.Spec.Parameters).To(HaveLen(2))
			Expect(beta.Spec.Parameters[0].Name).To(Equal("my-param"))
			Expect(beta.Spec.Parameters[1].Name).To(Equal("dockerfile"))
			Expect(beta.Spec.Parameters[1].Description).To(Equal("The Dockerfile to be built."))
			Expect(beta.Spec.Parameters[1].Type).To(Equal(buildapi.ParameterTypeString))
			Expect(beta.Spec.Parameters[1].Default).To(Equal(ptr.To("Dockerfile")))
		})

		It("rewrites $(build.dockerfile) and injects a dockerfile parameter", func() {
			alpha := &buildapialpha.BuildStrategy{
				ObjectMeta: metav1.ObjectMeta{Name: "my-bs", Namespace: "build-ns"},
				Spec: buildapialpha.BuildStrategySpec{
					BuildSteps: []buildapialpha.BuildStep{{
						Container: corev1.Container{
							Name:    "build",
							Image:   "gcr.io/kaniko:latest",
							Command: []string{"$(build.dockerfile)"},
							Args:    []string{"--file=$(build.dockerfile)"},
							Env:     []corev1.EnvVar{{Name: "DOCKERFILE", Value: "$(build.dockerfile)"}},
						},
					}},
				},
			}
			alpha.Kind = "BuildStrategy"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.BuildStrategy{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.Spec.Steps[0].Command).To(Equal([]string{"$(params.dockerfile)"}))
			Expect(beta.Spec.Steps[0].Args).To(Equal([]string{"--file=$(params.dockerfile)"}))
			Expect(beta.Spec.Steps[0].Env[0].Value).To(Equal("$(params.dockerfile)"))
			Expect(beta.Spec.Parameters).To(ConsistOf(buildapi.Parameter{
				Name:        "dockerfile",
				Description: "The Dockerfile to be built.",
				Type:        buildapi.ParameterTypeString,
				Default:     ptr.To("Dockerfile"),
			}))
		})

		It("rewrites $(build.builder.image) and injects a builder-image parameter", func() {
			alpha := &buildapialpha.BuildStrategy{
				ObjectMeta: metav1.ObjectMeta{Name: "my-bs", Namespace: "build-ns"},
				Spec: buildapialpha.BuildStrategySpec{
					BuildSteps: []buildapialpha.BuildStep{{
						Container: corev1.Container{
							Name:    "build",
							Image:   "gcr.io/kaniko:latest",
							Command: []string{"use", "$(build.builder.image)"},
							Args:    []string{"$(build.builder.image)"},
							Env:     []corev1.EnvVar{{Name: "BUILDER", Value: "$(build.builder.image)"}},
						},
					}},
				},
			}
			alpha.Kind = "BuildStrategy"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.BuildStrategy{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.Spec.Steps[0].Command).To(Equal([]string{"use", "$(params.builder-image)"}))
			Expect(beta.Spec.Steps[0].Args).To(Equal([]string{"$(params.builder-image)"}))
			Expect(beta.Spec.Steps[0].Env[0].Value).To(Equal("$(params.builder-image)"))
			Expect(beta.Spec.Parameters).To(ConsistOf(buildapi.Parameter{
				Name:        "builder-image",
				Description: "The builder image.",
				Type:        buildapi.ParameterTypeString,
			}))
		})
	})

	It("round-trips beta -> alpha -> beta", func() {
		start := sampleBetaBuildStrategy()
		start.Kind = "BuildStrategy"
		start.APIVersion = "shipwright.io/v1beta1"
		orig := start.DeepCopy()

		u := &unstructured.Unstructured{}
		Expect(start.ConvertTo(context.TODO(), u)).To(Succeed())

		got := &buildapi.BuildStrategy{}
		Expect(got.ConvertFrom(context.TODO(), u)).To(Succeed())

		Expect(got.Spec.Steps).To(Equal(orig.Spec.Steps))
		Expect(got.Spec.Parameters).To(Equal(orig.Spec.Parameters))
		Expect(got.Spec.SecurityContext).To(Equal(orig.Spec.SecurityContext))
		Expect(got.Spec.Volumes).To(Equal(orig.Spec.Volumes))
		Expect(got.Name).To(Equal(orig.Name))
		Expect(got.Namespace).To(Equal(orig.Namespace))
		Expect(got.APIVersion).To(Equal("shipwright.io/v1beta1"))
	})
})

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

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

func sampleBetaCBS() *v1beta1.ClusterBuildStrategy {
	return &v1beta1.ClusterBuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-cbs",
			Labels:      map[string]string{"app": "test"},
			Annotations: map[string]string{"note": "hi"},
		},
		Spec: v1beta1.BuildStrategySpec{
			Steps: []v1beta1.Step{{
				Name:    "build",
				Image:   "gcr.io/kaniko:latest",
				Command: []string{"sh"},
				Args:    []string{"-c", "echo hi"},
			}},
			Parameters: []v1beta1.Parameter{{
				Name:        "my-param",
				Description: "a plain param",
				Type:        v1beta1.ParameterTypeString,
			}},
		},
	}
}

var _ = Describe("ClusterBuildStrategy conversion", func() {
	Context("Convert to (beta -> alpha)", func() {
		It("preserves meta and spec,flips apiversion", func() {
			beta := sampleBetaCBS()
			beta.Kind = "ClusterBuildStrategy"
			beta.APIVersion = "shipwright.io/v1beta1"

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			Expect(u.GetAPIVersion()).To(Equal("shipwright.io/v1alpha1"))
			Expect(u.GetKind()).To(Equal("ClusterBuildStrategy"))

			var alpha v1alpha1.ClusterBuildStrategy
			Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &alpha)).To(Succeed())

			Expect(alpha.Name).To(Equal("my-cbs"))
			Expect(alpha.Labels).To(Equal(map[string]string{"app": "test"}))
			Expect(alpha.Annotations).To(Equal(map[string]string{"note": "hi"}))
			Expect(alpha.Spec.BuildSteps).To(HaveLen(1))
			Expect(alpha.Spec.BuildSteps[0].Image).To(Equal("gcr.io/kaniko:latest"))
			Expect(alpha.Spec.Parameters).To(HaveLen(1))
		})
	})

	Context("ConvertFrom (alpha -> beta)", func() {
		It("preserves meta and spec, flips apiVersion", func() {
			alpha := &v1alpha1.ClusterBuildStrategy{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-cbs",
					Labels:      map[string]string{"app": "test"},
					Annotations: map[string]string{"note": "hi"},
				},
				Spec: v1alpha1.BuildStrategySpec{
					BuildSteps: []v1alpha1.BuildStep{{
						Container: corev1.Container{Name: "build", Image: "gcr.io/kaniko:latest"},
					}},
					Parameters: []v1alpha1.Parameter{{
						Name:        "my-param",
						Description: "a plain param",
						Type:        v1alpha1.ParameterTypeString,
					}},
				},
			}
			alpha.Kind = "ClusterBuildStrategy"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &v1beta1.ClusterBuildStrategy{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.APIVersion).To(Equal("shipwright.io/v1beta1"))
			Expect(beta.Name).To(Equal("my-cbs"))
			Expect(beta.Labels).To(Equal(map[string]string{"app": "test"}))
			Expect(beta.Spec.Steps).To(HaveLen(1))
			Expect(beta.Spec.Steps[0].Image).To(Equal("gcr.io/kaniko:latest"))
			Expect(beta.Spec.Parameters).To(HaveLen(1))
			Expect(beta.Spec.Parameters[0].Name).To(Equal("my-param"))
		})
	})

	It("round-trips beta -> alpha -> beta", func() {
		start := sampleBetaCBS()
		start.Kind = "ClusterBuildStrategy"
		start.APIVersion = "shipwright.io/v1beta1"

		u := &unstructured.Unstructured{}
		Expect(start.ConvertTo(context.TODO(), u)).To(Succeed())

		got := &v1beta1.ClusterBuildStrategy{}
		Expect(got.ConvertFrom(context.TODO(), u)).To(Succeed())

		Expect(got.Spec.Steps).To(Equal(start.Spec.Steps))
		Expect(got.Spec.Parameters).To(Equal(start.Spec.Parameters))
		Expect(got.Spec.SecurityContext).To(Equal(start.Spec.SecurityContext))
		// ConvertTo/ConvertFrom normalize nil slices to empty ones
		// (they init `Volumes = []...`), so assert emptiness, not nil-ness.
		Expect(got.Spec.Volumes).To(BeEmpty())
		Expect(got.Name).To(Equal(start.Name))
		Expect(got.APIVersion).To(Equal("shipwright.io/v1beta1"))
	})
})

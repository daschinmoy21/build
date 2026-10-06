// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

func sampleBetaCBS() *buildapi.ClusterBuildStrategy {
	return &buildapi.ClusterBuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-cbs",
			Labels:      map[string]string{"app": "test"},
			Annotations: map[string]string{"note": "hi"},
		},
		Spec: sampleBetaBuildStrategy().Spec,
	}
}

var _ = Describe("ClusterBuildStrategy conversion", func() {
	Context("ConvertTo (beta -> alpha)", func() {
		It("preserves meta and spec, flips apiVersion", func() {
			beta := sampleBetaCBS()
			beta.Kind = "ClusterBuildStrategy"
			beta.APIVersion = "shipwright.io/v1beta1"

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			Expect(u.GetAPIVersion()).To(Equal("shipwright.io/v1alpha1"))
			Expect(u.GetKind()).To(Equal("ClusterBuildStrategy"))

			var alpha buildapialpha.ClusterBuildStrategy
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
			alpha := &buildapialpha.ClusterBuildStrategy{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-cbs",
					Labels:      map[string]string{"app": "test"},
					Annotations: map[string]string{"note": "hi"},
				},
				Spec: buildapialpha.BuildStrategySpec{
					BuildSteps: []buildapialpha.BuildStep{{
						Container: corev1.Container{Name: "build", Image: "gcr.io/kaniko:latest"},
					}},
					Parameters: []buildapialpha.Parameter{{
						Name:        "my-param",
						Description: "a plain param",
						Type:        buildapialpha.ParameterTypeString,
					}},
				},
			}
			alpha.Kind = "ClusterBuildStrategy"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.ClusterBuildStrategy{}
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

		got := &buildapi.ClusterBuildStrategy{}
		Expect(got.ConvertFrom(context.TODO(), u)).To(Succeed())

		Expect(got.Spec.Steps).To(Equal(start.Spec.Steps))
		Expect(got.Spec.Parameters).To(Equal(start.Spec.Parameters))
		Expect(got.Spec.SecurityContext).To(Equal(start.Spec.SecurityContext))
		Expect(got.Spec.Volumes).To(Equal(start.Spec.Volumes))
		Expect(got.Name).To(Equal(start.Name))
		Expect(got.APIVersion).To(Equal("shipwright.io/v1beta1"))
	})
})

var _ = Describe("ClusterBuildStrategy conversion field coverage", func() {
	DescribeTable("preserves parameter defaults and all step fields in both directions", func(raw string) {
		beta := sampleBetaCBS()
		beta.Kind, beta.APIVersion = "ClusterBuildStrategy", "shipwright.io/v1beta1"
		alpha := &buildapialpha.ClusterBuildStrategy{TypeMeta: metav1.TypeMeta{Kind: "ClusterBuildStrategy", APIVersion: "shipwright.io/v1alpha1"}, ObjectMeta: beta.ObjectMeta, Spec: sampleAlphaStrategySpec()}
		Expect(json.Unmarshal([]byte(raw), &alpha.Spec.Parameters)).To(Succeed())
		Expect(json.Unmarshal([]byte(raw), &beta.Spec.Parameters)).To(Succeed())
		origBeta, origAlpha := beta.DeepCopy(), alpha.DeepCopy()
		u := &unstructured.Unstructured{}
		Expect(beta.ConvertTo(context.Background(), u)).To(Succeed())
		var gotAlpha buildapialpha.ClusterBuildStrategy
		Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &gotAlpha)).To(Succeed())
		expectSameJSON(gotAlpha, origAlpha)
		got := &buildapi.ClusterBuildStrategy{}
		Expect(got.ConvertFrom(context.Background(), asUnstructured(alpha))).To(Succeed())
		expectSameJSON(got, origBeta)
		Expect(got.ConvertTo(context.Background(), u)).To(Succeed())
		Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &gotAlpha)).To(Succeed())
		expectSameJSON(gotAlpha, origAlpha)
		expectSameJSON(beta, origBeta)
	}, strategyParameterEntries())

	It("round-trips absent optional fields", func() {
		start := &buildapialpha.ClusterBuildStrategy{TypeMeta: metav1.TypeMeta{Kind: "ClusterBuildStrategy", APIVersion: "shipwright.io/v1alpha1"}, Spec: buildapialpha.BuildStrategySpec{BuildSteps: []buildapialpha.BuildStep{{Container: corev1.Container{Name: "build", Image: "builder"}}}}}
		beta := &buildapi.ClusterBuildStrategy{}
		Expect(beta.ConvertFrom(context.Background(), asUnstructured(start))).To(Succeed())
		u := &unstructured.Unstructured{}
		Expect(beta.ConvertTo(context.Background(), u)).To(Succeed())
		var got buildapialpha.ClusterBuildStrategy
		Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &got)).To(Succeed())
		expectSameJSON(got, start)
	})
})

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

func sampleBetaBuildRun() *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-buildrun",
			Namespace:   "build-ns",
			Labels:      map[string]string{"app": "test"},
			Annotations: map[string]string{"note": "hi"},
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: ptr.To("my-build"),
			},
			ServiceAccount: ptr.To("build-sa"),
			Timeout:        &metav1.Duration{Duration: 15 * time.Minute},
			ParamValues: []buildapi.ParamValue{{
				Name: "my-param",
				SingleValue: &buildapi.SingleValue{
					Value: ptr.To("my-value"),
				},
			}},
			Output: &buildapi.Image{
				Image:       "quay.io/example/app:run",
				Insecure:    ptr.To(true),
				PushSecret:  ptr.To("push-secret"),
				Labels:      map[string]string{"run": "yes"},
				Annotations: map[string]string{"org.opencontainers.image.url": "https://example.com"},
				Timestamp:   ptr.To(buildapi.OutputImageSourceTimestamp),
			},
			State: ptr.To(buildapi.BuildRunRequestedState(buildapi.BuildRunStateCancel)),
			Env:   []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
			Retention: &buildapi.BuildRunRetention{
				TTLAfterFailed:    &metav1.Duration{Duration: time.Hour},
				TTLAfterSucceeded: &metav1.Duration{Duration: 2 * time.Hour},
			},
			Volumes: []buildapi.BuildVolume{{
				Name:         "cache",
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			}},
		},
		Status: buildapi.BuildRunStatus{
			Source: &buildapi.SourceResult{
				Git: &buildapi.GitSourceResult{
					CommitSha:    "abc123",
					CommitAuthor: "dev",
					BranchName:   "main",
				},
				Timestamp: &metav1.Time{Time: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)},
			},
			Output: &buildapi.Output{
				Digest: "sha256:deadbeef",
				Size:   42,
			},
			Conditions: []buildapi.Condition{{
				Type:               buildapi.Succeeded,
				Status:             corev1.ConditionFalse,
				LastTransitionTime: metav1.Time{Time: time.Date(2024, 1, 2, 4, 0, 0, 0, time.UTC)},
				Reason:             "Failed",
				Message:            "step failed",
			}},
			Executor: &buildapi.BuildExecutor{
				Name: "my-buildrun-pod",
				Kind: "TaskRun",
			},
			StartTime:      &metav1.Time{Time: time.Date(2024, 1, 2, 3, 0, 0, 0, time.UTC)},
			CompletionTime: &metav1.Time{Time: time.Date(2024, 1, 2, 4, 0, 0, 0, time.UTC)},
			FailureDetails: &buildapi.FailureDetails{
				Reason:  "StepFailed",
				Message: "exit 1",
				Location: &buildapi.Location{
					Pod:       "my-buildrun-pod",
					Container: "step-build",
				},
			},
		},
	}
}

func alphaBuildRunFromUnstructured(u *unstructured.Unstructured) buildapialpha.BuildRun {
	var alpha buildapialpha.BuildRun
	Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &alpha)).To(Succeed())
	return alpha
}

var _ = Describe("BuildRun conversion", func() {
	Context("ConvertTo (beta -> alpha)", func() {
		It("preserves meta, build ref, output, and status", func() {
			beta := sampleBetaBuildRun()
			beta.Kind = "BuildRun"
			beta.APIVersion = "shipwright.io/v1beta1"

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			Expect(u.GetAPIVersion()).To(Equal("shipwright.io/v1alpha1"))
			Expect(u.GetKind()).To(Equal("BuildRun"))

			alpha := alphaBuildRunFromUnstructured(u)

			Expect(alpha.Name).To(Equal("my-buildrun"))
			Expect(alpha.Namespace).To(Equal("build-ns"))
			Expect(alpha.Labels).To(Equal(map[string]string{"app": "test"}))
			Expect(alpha.Annotations).To(Equal(map[string]string{"note": "hi"}))

			Expect(alpha.Spec.BuildRef).ToNot(BeNil())
			Expect(alpha.Spec.BuildRef.Name).To(Equal("my-build"))
			Expect(alpha.Spec.BuildSpec).To(BeNil())
			Expect(alpha.Spec.ServiceAccount).ToNot(BeNil())
			Expect(alpha.Spec.ServiceAccount.Name).To(Equal(ptr.To("build-sa")))
			Expect(alpha.Spec.ServiceAccount.Generate).To(BeNil())
			Expect(alpha.Spec.Timeout).To(Equal(&metav1.Duration{Duration: 15 * time.Minute}))
			Expect(alpha.Spec.ParamValues).To(HaveLen(1))
			Expect(alpha.Spec.ParamValues[0].Name).To(Equal("my-param"))
			Expect(alpha.Spec.Output.Image).To(Equal("quay.io/example/app:run"))
			Expect(alpha.Spec.Output.Credentials).To(Equal(&corev1.LocalObjectReference{Name: "push-secret"}))
			Expect(alpha.Spec.State).To(Equal(ptr.To(buildapialpha.BuildRunRequestedState(buildapialpha.BuildRunStateCancel))))
			Expect(alpha.Spec.Env).To(Equal([]corev1.EnvVar{{Name: "FOO", Value: "bar"}}))
			Expect(alpha.Spec.Retention).ToNot(BeNil())
			Expect(alpha.Spec.Volumes).To(HaveLen(1))
			Expect(alpha.Spec.Volumes[0].Name).To(Equal("cache"))

			Expect(alpha.Status.Sources).To(HaveLen(1))
			Expect(alpha.Status.Sources[0].Name).To(Equal("default"))
			Expect(alpha.Status.Sources[0].Git.CommitSha).To(Equal("abc123"))
			Expect(alpha.Status.Sources[0].Git.CommitAuthor).To(Equal("dev"))
			Expect(alpha.Status.Output.Digest).To(Equal("sha256:deadbeef"))
			Expect(alpha.Status.Output.Size).To(Equal(int64(42)))
			Expect(alpha.Status.Conditions).To(HaveLen(1))
			Expect(alpha.Status.Conditions[0].Type).To(Equal(buildapialpha.Succeeded))
			Expect(alpha.Status.LatestTaskRunRef).To(Equal(ptr.To("my-buildrun-pod")))
			Expect(alpha.Status.FailureDetails).ToNot(BeNil())
			Expect(alpha.Status.FailureDetails.Reason).To(Equal("StepFailed"))
			Expect(alpha.Status.FailureDetails.Location.Pod).To(Equal("my-buildrun-pod"))
			Expect(alpha.Status.FailureDetails.Location.Container).To(Equal("step-build"))
			// ConvertTo mirrors FailureDetails.Location into the deprecated FailedAt field.
			//nolint:staticcheck // Verify compatibility with the deprecated v1alpha1 status field.
			Expect(alpha.Status.FailedAt).To(Equal(alpha.Status.FailureDetails.Location))
		})

		It("maps embedded Build.Spec to alpha BuildSpec", func() {
			beta := sampleBetaBuildRun()
			beta.Kind = "BuildRun"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Spec.Build = buildapi.ReferencedBuild{
				Spec: &buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type: buildapi.GitType,
						Git: &buildapi.Git{
							URL: "https://github.com/shipwright-io/sample",
						},
					},
					Strategy: buildapi.Strategy{Name: "kaniko"},
					Output:   buildapi.Image{Image: "quay.io/example/app:embedded"},
				},
			}

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaBuildRunFromUnstructured(u)
			Expect(alpha.Spec.BuildRef).To(BeNil())
			Expect(alpha.Spec.BuildSpec).ToNot(BeNil())
			Expect(alpha.Spec.BuildSpec.Source.URL).To(Equal(ptr.To("https://github.com/shipwright-io/sample")))
			Expect(alpha.Spec.BuildSpec.Strategy.Name).To(Equal("kaniko"))
			Expect(alpha.Spec.BuildSpec.Output.Image).To(Equal("quay.io/example/app:embedded"))
		})

		It("maps ServiceAccount .generate to Generate=true", func() {
			beta := sampleBetaBuildRun()
			beta.Kind = "BuildRun"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Spec.ServiceAccount = ptr.To(".generate")

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaBuildRunFromUnstructured(u)
			Expect(alpha.Spec.ServiceAccount.Generate).To(Equal(ptr.To(true)))
			Expect(alpha.Spec.ServiceAccount.Name).To(Equal(ptr.To("my-buildrun")))
		})

		It("maps Local BuildRun source to alpha Sources", func() {
			beta := sampleBetaBuildRun()
			beta.Kind = "BuildRun"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Spec.Source = &buildapi.BuildRunSource{
				Type: buildapi.LocalType,
				Local: &buildapi.Local{
					Name:    "upload",
					Timeout: &metav1.Duration{Duration: 3 * time.Minute},
				},
			}

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaBuildRunFromUnstructured(u)
			Expect(alpha.Spec.Sources).To(HaveLen(1))
			Expect(alpha.Spec.Sources[0].Name).To(Equal("upload"))
			Expect(alpha.Spec.Sources[0].Type).To(Equal(buildapialpha.LocalCopy))
			Expect(alpha.Spec.Sources[0].Timeout).To(Equal(&metav1.Duration{Duration: 3 * time.Minute}))
		})

		It("maps OCI status source to alpha Bundle SourceResult", func() {
			beta := sampleBetaBuildRun()
			beta.Kind = "BuildRun"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Status.Source = &buildapi.SourceResult{
				OciArtifact: &buildapi.OciArtifactSourceResult{Digest: "sha256:bundle"},
			}

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaBuildRunFromUnstructured(u)
			Expect(alpha.Status.Sources).To(HaveLen(1))
			Expect(alpha.Status.Sources[0].Bundle.Digest).To(Equal("sha256:bundle"))
			Expect(alpha.Status.Sources[0].Git).To(BeNil())
		})

		It("falls back to deprecated TaskRunName when Executor is unset", func() {
			beta := sampleBetaBuildRun()
			beta.Kind = "BuildRun"
			beta.APIVersion = "shipwright.io/v1beta1"
			beta.Status.Executor = nil
			beta.Status.TaskRunName = ptr.To("legacy-taskrun")

			u := &unstructured.Unstructured{}
			Expect(beta.ConvertTo(context.TODO(), u)).To(Succeed())

			alpha := alphaBuildRunFromUnstructured(u)
			Expect(alpha.Status.LatestTaskRunRef).To(Equal(ptr.To("legacy-taskrun")))
		})
	})

	Context("ConvertFrom (alpha -> beta)", func() {
		It("preserves meta, build ref, output, and status", func() {
			alpha := &buildapialpha.BuildRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-buildrun",
					Namespace:   "build-ns",
					Labels:      map[string]string{"app": "test"},
					Annotations: map[string]string{"note": "hi"},
				},
				Spec: buildapialpha.BuildRunSpec{
					BuildRef: &buildapialpha.BuildRef{Name: "my-build"},
					ServiceAccount: &buildapialpha.ServiceAccount{
						Name: ptr.To("build-sa"),
					},
					Timeout: &metav1.Duration{Duration: 15 * time.Minute},
					ParamValues: []buildapialpha.ParamValue{{
						Name: "my-param",
						SingleValue: &buildapialpha.SingleValue{
							Value: ptr.To("my-value"),
						},
					}},
					Output: &buildapialpha.Image{
						Image: "quay.io/example/app:run",
						Credentials: &corev1.LocalObjectReference{
							Name: "push-secret",
						},
					},
					State: ptr.To(buildapialpha.BuildRunRequestedState(buildapialpha.BuildRunStateCancel)),
					Env:   []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
					Retention: &buildapialpha.BuildRunRetention{
						TTLAfterFailed: &metav1.Duration{Duration: time.Hour},
					},
					Volumes: []buildapialpha.BuildVolume{{
						Name:         "cache",
						VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
					}},
				},
				Status: buildapialpha.BuildRunStatus{
					Sources: []buildapialpha.SourceResult{{
						Name: "default",
						Git: &buildapialpha.GitSourceResult{
							CommitSha:    "abc123",
							CommitAuthor: "dev",
							BranchName:   "main",
						},
					}},
					Output: &buildapialpha.Output{
						Digest: "sha256:deadbeef",
						Size:   42,
					},
					Conditions: []buildapialpha.Condition{{
						Type:               buildapialpha.Succeeded,
						Status:             corev1.ConditionFalse,
						LastTransitionTime: metav1.Time{Time: time.Date(2024, 1, 2, 4, 0, 0, 0, time.UTC)},
						Reason:             "Failed",
						Message:            "step failed",
					}},
					LatestTaskRunRef: ptr.To("my-buildrun-pod"),
					StartTime:        &metav1.Time{Time: time.Date(2024, 1, 2, 3, 0, 0, 0, time.UTC)},
					CompletionTime:   &metav1.Time{Time: time.Date(2024, 1, 2, 4, 0, 0, 0, time.UTC)},
					FailureDetails: &buildapialpha.FailureDetails{
						Reason:  "StepFailed",
						Message: "exit 1",
						Location: &buildapialpha.FailedAt{
							Pod:       "my-buildrun-pod",
							Container: "step-build",
						},
					},
				},
			}
			alpha.Kind = "BuildRun"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.BuildRun{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())

			Expect(beta.APIVersion).To(Equal("shipwright.io/v1beta1"))
			Expect(beta.Name).To(Equal("my-buildrun"))
			Expect(beta.Namespace).To(Equal("build-ns"))
			Expect(beta.Spec.Build.Name).To(Equal(ptr.To("my-build")))
			Expect(beta.Spec.Build.Spec).To(BeNil())
			Expect(beta.Spec.ServiceAccount).To(Equal(ptr.To("build-sa")))
			Expect(beta.Spec.Timeout).To(Equal(&metav1.Duration{Duration: 15 * time.Minute}))
			Expect(beta.Spec.ParamValues).To(HaveLen(1))
			Expect(beta.Spec.Output.Image).To(Equal("quay.io/example/app:run"))
			Expect(beta.Spec.Output.PushSecret).To(Equal(ptr.To("push-secret")))
			Expect(beta.Spec.State).To(Equal(ptr.To(buildapi.BuildRunRequestedState(buildapi.BuildRunStateCancel))))
			Expect(beta.Spec.Volumes).To(HaveLen(1))

			Expect(beta.Status.Source.Git.CommitSha).To(Equal("abc123"))
			Expect(beta.Status.Output.Digest).To(Equal("sha256:deadbeef"))
			Expect(beta.Status.Conditions).To(HaveLen(1))
			Expect(beta.Status.Executor).To(Equal(&buildapi.BuildExecutor{
				Name: "my-buildrun-pod",
				Kind: "TaskRun",
			}))
			Expect(beta.Status.TaskRunName).To(Equal(ptr.To("my-buildrun-pod")))
			Expect(beta.Status.FailureDetails.Reason).To(Equal("StepFailed"))
			Expect(beta.Status.FailureDetails.Location.Pod).To(Equal("my-buildrun-pod"))
			Expect(beta.Status.FailureDetails.Location.Container).To(Equal("step-build"))
		})

		It("maps Generate=true to ServiceAccount .generate", func() {
			alpha := &buildapialpha.BuildRun{
				ObjectMeta: metav1.ObjectMeta{Name: "my-buildrun", Namespace: "build-ns"},
				Spec: buildapialpha.BuildRunSpec{
					BuildRef: &buildapialpha.BuildRef{Name: "my-build"},
					ServiceAccount: &buildapialpha.ServiceAccount{
						Name:     ptr.To("ignored-when-generate"),
						Generate: ptr.To(true),
					},
				},
			}
			alpha.Kind = "BuildRun"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.BuildRun{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())
			Expect(beta.Spec.ServiceAccount).To(Equal(ptr.To(".generate")))
		})

		It("maps LocalCopy Sources to BuildRun Local source", func() {
			alpha := &buildapialpha.BuildRun{
				ObjectMeta: metav1.ObjectMeta{Name: "my-buildrun", Namespace: "build-ns"},
				Spec: buildapialpha.BuildRunSpec{
					BuildRef: &buildapialpha.BuildRef{Name: "my-build"},
					Sources: []buildapialpha.BuildSource{{
						Name:    "upload",
						Type:    buildapialpha.LocalCopy,
						Timeout: &metav1.Duration{Duration: 3 * time.Minute},
					}},
				},
			}
			alpha.Kind = "BuildRun"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.BuildRun{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())
			Expect(beta.Spec.Source.Type).To(Equal(buildapi.LocalType))
			Expect(beta.Spec.Source.Local.Name).To(Equal("upload"))
			Expect(beta.Spec.Source.Local.Timeout).To(Equal(&metav1.Duration{Duration: 3 * time.Minute}))
		})

		It("maps embedded BuildSpec and status BuildSpec", func() {
			alpha := &buildapialpha.BuildRun{
				ObjectMeta: metav1.ObjectMeta{Name: "my-buildrun", Namespace: "build-ns"},
				Spec: buildapialpha.BuildRunSpec{
					BuildSpec: &buildapialpha.BuildSpec{
						Source:   buildapialpha.Source{URL: ptr.To("https://github.com/shipwright-io/sample")},
						Strategy: buildapialpha.Strategy{Name: "kaniko"},
						Output:   buildapialpha.Image{Image: "quay.io/example/app:embedded"},
					},
				},
				Status: buildapialpha.BuildRunStatus{
					BuildSpec: &buildapialpha.BuildSpec{
						Source:   buildapialpha.Source{URL: ptr.To("https://github.com/shipwright-io/sample")},
						Strategy: buildapialpha.Strategy{Name: "kaniko"},
						Output:   buildapialpha.Image{Image: "quay.io/example/app:embedded"},
					},
				},
			}
			alpha.Kind = "BuildRun"
			alpha.APIVersion = "shipwright.io/v1alpha1"

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alpha)
			Expect(err).ToNot(HaveOccurred())

			beta := &buildapi.BuildRun{}
			Expect(beta.ConvertFrom(context.TODO(), &unstructured.Unstructured{Object: raw})).To(Succeed())
			Expect(beta.Spec.Build.Name).To(BeNil())
			Expect(beta.Spec.Build.Spec).ToNot(BeNil())
			Expect(beta.Spec.Build.Spec.Source.Git.URL).To(Equal("https://github.com/shipwright-io/sample"))
			Expect(beta.Status.BuildSpec).ToNot(BeNil())
			Expect(beta.Status.BuildSpec.Strategy.Name).To(Equal("kaniko"))
		})

		// TODO: Add coverage for deprecated status.failedAt without FailureDetails
		// with the fix for https://github.com/shipwright-io/build/issues/2304.
	})

	It("round-trips beta -> alpha -> beta", func() {
		start := sampleBetaBuildRun()
		start.Kind = "BuildRun"
		start.APIVersion = "shipwright.io/v1beta1"
		orig := start.DeepCopy()

		u := &unstructured.Unstructured{}
		Expect(start.ConvertTo(context.TODO(), u)).To(Succeed())

		got := &buildapi.BuildRun{}
		Expect(got.ConvertFrom(context.TODO(), u)).To(Succeed())

		Expect(got.Name).To(Equal(orig.Name))
		Expect(got.Namespace).To(Equal(orig.Namespace))
		Expect(got.APIVersion).To(Equal("shipwright.io/v1beta1"))
		Expect(got.Spec.Build).To(Equal(orig.Spec.Build))
		Expect(got.Spec.ServiceAccount).To(Equal(orig.Spec.ServiceAccount))
		Expect(got.Spec.Timeout).To(Equal(orig.Spec.Timeout))
		Expect(got.Spec.ParamValues).To(Equal(orig.Spec.ParamValues))
		Expect(got.Spec.Output).To(Equal(orig.Spec.Output))
		Expect(got.Spec.State).To(Equal(orig.Spec.State))
		Expect(got.Spec.Env).To(Equal(orig.Spec.Env))
		Expect(got.Spec.Retention).To(Equal(orig.Spec.Retention))
		Expect(got.Spec.Volumes).To(Equal(orig.Spec.Volumes))

		Expect(got.Status.Source.Git).To(Equal(orig.Status.Source.Git))
		Expect(got.Status.Source.Timestamp.Unix()).To(Equal(orig.Status.Source.Timestamp.Unix()))
		Expect(got.Status.Output.Digest).To(Equal(orig.Status.Output.Digest))
		Expect(got.Status.Output.Size).To(Equal(orig.Status.Output.Size))
		Expect(got.Status.Conditions).To(HaveLen(1))
		Expect(got.Status.Conditions[0].Type).To(Equal(orig.Status.Conditions[0].Type))
		Expect(got.Status.Conditions[0].Status).To(Equal(orig.Status.Conditions[0].Status))
		Expect(got.Status.Conditions[0].Reason).To(Equal(orig.Status.Conditions[0].Reason))
		Expect(got.Status.Conditions[0].Message).To(Equal(orig.Status.Conditions[0].Message))
		Expect(got.Status.Conditions[0].LastTransitionTime.Unix()).To(Equal(orig.Status.Conditions[0].LastTransitionTime.Unix()))
		Expect(got.Status.Executor).To(Equal(orig.Status.Executor))
		Expect(got.Status.TaskRunName).To(Equal(ptr.To(orig.Status.Executor.Name)))
		Expect(got.Status.StartTime.Unix()).To(Equal(orig.Status.StartTime.Unix()))
		Expect(got.Status.CompletionTime.Unix()).To(Equal(orig.Status.CompletionTime.Unix()))
		Expect(got.Status.FailureDetails).To(Equal(orig.Status.FailureDetails))
	})
})

var _ = Describe("BuildRun conversion field coverage", func() {
	DescribeTable("preserves parameter variants in both directions", func(raw string) {
		betaParam, alphaParam := parameterPair(raw)
		beta := sampleBetaBuildRun()
		beta.Spec.ParamValues = []buildapi.ParamValue{betaParam}
		u := &unstructured.Unstructured{}
		Expect(beta.ConvertTo(context.Background(), u)).To(Succeed())
		expectSameJSON(alphaBuildRunFromUnstructured(u).Spec.ParamValues, []buildapialpha.ParamValue{alphaParam})
		alpha := sampleAlphaBuildRun()
		alpha.Spec.ParamValues = []buildapialpha.ParamValue{alphaParam}
		got := &buildapi.BuildRun{}
		Expect(got.ConvertFrom(context.Background(), asUnstructured(alpha))).To(Succeed())
		expectSameJSON(got.Spec.ParamValues, []buildapi.ParamValue{betaParam})
		Expect(got.ConvertTo(context.Background(), u)).To(Succeed())
		expectSameJSON(alphaBuildRunFromUnstructured(u), alpha)
	}, parameterEntries())

	DescribeTable("round-trips all shared spec and status fields from alpha", func(embedded bool, bundle bool) {
		start := sampleAlphaBuildRun()
		if embedded {
			start.Spec.BuildRef = nil
			start.Spec.BuildSpec = ptr.To(sampleAlphaBuild().Spec)
		}
		if bundle {
			start.Status.Sources[0].Git = nil
			start.Status.Sources[0].Bundle = &buildapialpha.BundleSourceResult{Digest: "sha256:bundle"}
		}
		orig := start.DeepCopy()
		beta := &buildapi.BuildRun{}
		Expect(beta.ConvertFrom(context.Background(), asUnstructured(start))).To(Succeed())
		want := sampleBetaBuildRun()
		want.Kind, want.APIVersion = "BuildRun", "shipwright.io/v1beta1"
		want.Spec.Source = &buildapi.BuildRunSource{Type: buildapi.LocalType, Local: &buildapi.Local{Name: "upload", Timeout: &metav1.Duration{Duration: 3 * time.Minute}}}
		want.Spec.ParamValues = betaParamValues()
		buildSpec := sampleBetaBuild().Spec
		buildSpec.ParamValues = append(betaParamValues(),
			buildapi.ParamValue{Name: "dockerfile", SingleValue: &buildapi.SingleValue{Value: ptr.To("Dockerfile.prod")}},
			buildapi.ParamValue{Name: "builder-image", SingleValue: &buildapi.SingleValue{Value: ptr.To("golang:1.22")}},
		)
		want.Status.BuildSpec = &buildSpec
		want.Status.Source.Timestamp = want.Status.StartTime
		want.Status.TaskRunName = ptr.To("my-buildrun-pod")
		if embedded {
			want.Spec.Build = buildapi.ReferencedBuild{Spec: &buildSpec}
		}
		if bundle {
			want.Status.Source.Git = nil
			want.Status.Source.OciArtifact = &buildapi.OciArtifactSourceResult{Digest: "sha256:bundle"}
		}
		expectSameJSON(beta, want)
		// Also exercise beta -> alpha using the independent beta fixture.
		direct := &unstructured.Unstructured{}
		Expect(want.ConvertTo(context.Background(), direct)).To(Succeed())
		expectSameJSON(alphaBuildRunFromUnstructured(direct), orig)
		u := &unstructured.Unstructured{}
		Expect(beta.ConvertTo(context.Background(), u)).To(Succeed())
		expectSameJSON(alphaBuildRunFromUnstructured(u), orig)
	}, Entry("build reference and Git status", false, false), Entry("embedded build and Git status", true, false), Entry("build reference and OCI status", false, true), Entry("embedded build and OCI status", true, true))

	DescribeTable("round-trips failure details with optional locations", func(location bool) {
		start := sampleBetaBuildRun()
		if !location {
			start.Status.FailureDetails.Location = nil
		}
		u := &unstructured.Unstructured{}
		Expect(start.ConvertTo(context.Background(), u)).To(Succeed())
		alpha := alphaBuildRunFromUnstructured(u)
		Expect(alpha.Status.FailureDetails.Message).To(Equal("exit 1"))
		got := &buildapi.BuildRun{}
		Expect(got.ConvertFrom(context.Background(), u)).To(Succeed())
		expectSameJSON(got.Status.FailureDetails, start.Status.FailureDetails)
	}, Entry("with location", true), Entry("without location", false))

	It("prefers FailureDetails when the deprecated FailedAt disagrees", func() {
		alpha := sampleAlphaBuildRun()
		//nolint:staticcheck // Verify the newer field remains authoritative over deprecated status.
		alpha.Status.FailedAt = &buildapialpha.FailedAt{Pod: "legacy-pod", Container: "legacy-step"}
		got := &buildapi.BuildRun{}
		Expect(got.ConvertFrom(context.Background(), asUnstructured(alpha))).To(Succeed())
		expectSameJSON(got.Status.FailureDetails, sampleBetaBuildRun().Status.FailureDetails)
	})

	It("prefers Executor over the deprecated TaskRunName", func() {
		beta := sampleBetaBuildRun()
		beta.Status.TaskRunName = ptr.To("legacy-taskrun")
		u := &unstructured.Unstructured{}
		Expect(beta.ConvertTo(context.Background(), u)).To(Succeed())
		Expect(alphaBuildRunFromUnstructured(u).Status.LatestTaskRunRef).To(Equal(ptr.To(beta.Status.Executor.Name)))
	})

	It("round-trips absent optional fields", func() {
		start := &buildapialpha.BuildRun{TypeMeta: metav1.TypeMeta{Kind: "BuildRun", APIVersion: "shipwright.io/v1alpha1"}, ObjectMeta: metav1.ObjectMeta{Name: "minimal"}, Spec: buildapialpha.BuildRunSpec{BuildRef: &buildapialpha.BuildRef{Name: "build"}}}
		beta := &buildapi.BuildRun{}
		Expect(beta.ConvertFrom(context.Background(), asUnstructured(start))).To(Succeed())
		Expect(beta.Spec.ServiceAccount).To(BeNil())
		u := &unstructured.Unstructured{}
		Expect(beta.ConvertTo(context.Background(), u)).To(Succeed())
		// Existing webhook compatibility tests specify an empty alpha
		// serviceAccount for an unset beta service account (default semantics).
		start.Spec.ServiceAccount = &buildapialpha.ServiceAccount{}
		expectSameJSON(alphaBuildRunFromUnstructured(u), start)
	})
})

var _ = Describe("BuildRun conversion API differences", func() {
	It("omits beta-only spec and status fields from alpha", func() {
		// These fields do not exist in the alpha schema. This is distinct from
		// losing shared fields such as output.insecure or FailureDetails.
		beta := sampleBetaBuildRun()
		beta.Spec.NodeSelector = map[string]string{"pool": "build"}
		beta.Spec.Tolerations = []corev1.Toleration{{Key: "build", Operator: corev1.TolerationOpExists}}
		beta.Spec.SchedulerName, beta.Spec.RuntimeClassName = ptr.To("scheduler"), ptr.To("runtime")
		beta.Spec.StepResources = []buildapi.StepResourceOverride{{Name: "build", Resources: conversionStepResources()}}
		beta.Spec.Output.VulnerabilityScan = &buildapi.VulnerabilityScanOptions{Enabled: true}
		beta.Spec.Output.Platforms = []buildapi.ImagePlatform{{OS: "linux", Arch: "arm64"}}
		beta.Status.Output.Vulnerabilities = []buildapi.Vulnerability{{ID: "CVE-TEST", Severity: buildapi.High}}
		u := &unstructured.Unstructured{}
		Expect(beta.ConvertTo(context.Background(), u)).To(Succeed())
		for _, field := range [][]string{{"spec", "nodeSelector"}, {"spec", "tolerations"}, {"spec", "schedulerName"}, {"spec", "runtimeClassName"}, {"spec", "stepResources"}, {"spec", "output", "vulnerabilityScan"}, {"spec", "output", "platforms"}, {"status", "output", "vulnerabilities"}, {"status", "executor"}} {
			_, found, err := unstructured.NestedFieldNoCopy(u.Object, field...)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse(), "alpha has no %v field", field)
		}
		Expect(alphaBuildRunFromUnstructured(u).Status.LatestTaskRunRef).To(Equal(ptr.To(beta.Status.Executor.Name)))
	})
})

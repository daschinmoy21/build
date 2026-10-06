// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// Compare populated API fields. The unstructured converter can allocate an
// empty inline SingleValue and emit optional null fields that were omitted on
// input. Nil/empty omitted slices and time locations can also differ in Go.
func expectSameJSON(actual, expected any) {
	GinkgoHelper()
	Expect(normalizedJSON(actual)).To(MatchJSON(normalizedJSON(expected)))
}

func normalizedJSON(object any) []byte {
	GinkgoHelper()
	raw, err := json.Marshal(object)
	Expect(err).NotTo(HaveOccurred())
	var value any
	Expect(json.Unmarshal(raw, &value)).To(Succeed())
	removeNullFields(value)
	raw, err = json.Marshal(value)
	Expect(err).NotTo(HaveOccurred())
	return raw
}

func removeNullFields(value any) {
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			if child == nil {
				delete(item, key)
			} else {
				removeNullFields(child)
			}
		}
	case []any:
		for _, child := range item {
			removeNullFields(child)
		}
	}
}

func asUnstructured(object any) *unstructured.Unstructured {
	GinkgoHelper()
	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	Expect(err).NotTo(HaveOccurred())
	return &unstructured.Unstructured{Object: raw}
}

// Parameter names and value fields map directly between both versions. Use
// independent JSON fixtures instead of deriving expectations with converters.
const conversionParamValues = `[
 {"name":"literal","value":"hello"},
 {"name":"config","configMapValue":{"name":"settings","key":"flag","format":"--flag=${CONFIGMAP_VALUE}"}},
 {"name":"secret","secretValue":{"name":"credentials","key":"token","format":"Bearer ${SECRET_VALUE}"}},
 {"name":"array","values":[{"value":"first"},{"configMapValue":{"name":"settings","key":"second"}},{"secretValue":{"name":"credentials","key":"third"}}]}
]`

func betaParamValues() []buildapi.ParamValue {
	GinkgoHelper()
	var values []buildapi.ParamValue
	Expect(json.Unmarshal([]byte(conversionParamValues), &values)).To(Succeed())
	return values
}

func alphaParamValues() []buildapialpha.ParamValue {
	GinkgoHelper()
	var values []buildapialpha.ParamValue
	Expect(json.Unmarshal([]byte(conversionParamValues), &values)).To(Succeed())
	return values
}

func parameterEntries() []TableEntry {
	return []TableEntry{
		Entry("literal", `{"name":"param","value":"hello"}`),
		Entry("empty literal", `{"name":"param","value":""}`),
		Entry("ConfigMap reference", `{"name":"param","configMapValue":{"name":"settings","key":"flag","format":"--flag=${CONFIGMAP_VALUE}"}}`),
		Entry("Secret reference", `{"name":"param","secretValue":{"name":"credentials","key":"token","format":"Bearer ${SECRET_VALUE}"}}`),
		Entry("array with all value variants", `{"name":"param","values":[{"value":"hello"},{"configMapValue":{"name":"settings","key":"flag"}},{"secretValue":{"name":"credentials","key":"token"}}]}`),
		Entry("dockerfile ConfigMap reference", `{"name":"dockerfile","configMapValue":{"name":"settings","key":"dockerfile"}}`),
		Entry("builder-image Secret reference", `{"name":"builder-image","secretValue":{"name":"credentials","key":"builder"}}`),
	}
}

func parameterPair(raw string) (buildapi.ParamValue, buildapialpha.ParamValue) {
	GinkgoHelper()
	var beta buildapi.ParamValue
	var alpha buildapialpha.ParamValue
	Expect(json.Unmarshal([]byte(raw), &beta)).To(Succeed())
	Expect(json.Unmarshal([]byte(raw), &alpha)).To(Succeed())
	return beta, alpha
}

func strategyParameterEntries() []TableEntry {
	return []TableEntry{
		Entry("string default", `[{"name":"param","description":"string","type":"string","default":"hello"}]`),
		Entry("empty string default", `[{"name":"param","description":"string","type":"string","default":""}]`),
		Entry("required string", `[{"name":"param","description":"string","type":"string"}]`),
		Entry("array defaults", `[{"name":"param","description":"array","type":"array","defaults":["first","second"]}]`),
		Entry("empty array defaults", `[{"name":"param","description":"array","type":"array","defaults":[]}]`),
		Entry("required array", `[{"name":"param","description":"array","type":"array"}]`),
		Entry("builder-image with a custom default", `[{"name":"builder-image","description":"custom builder","type":"string","default":"golang:1.22"}]`),
		Entry("dockerfile as an array", `[{"name":"dockerfile","description":"array","type":"array","defaults":["Dockerfile"]}]`),
	}
}

func conversionStepResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("64Mi")},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("256Mi")},
	}
}

func sampleAlphaStrategySpec() buildapialpha.BuildStrategySpec {
	return buildapialpha.BuildStrategySpec{
		BuildSteps: []buildapialpha.BuildStep{{Container: corev1.Container{
			Name: "build", Image: "gcr.io/kaniko:latest", Command: []string{"sh"}, Args: []string{"-c", "echo hi"},
			WorkingDir: "/workspace", Env: []corev1.EnvVar{{Name: "FOO", Value: "bar"}}, ImagePullPolicy: corev1.PullIfNotPresent,
			Resources: conversionStepResources(), VolumeMounts: []corev1.VolumeMount{{Name: "cache", MountPath: "/cache", ReadOnly: true}},
			SecurityContext: &corev1.SecurityContext{RunAsUser: ptr.To[int64](1001), RunAsGroup: ptr.To[int64](1002), AllowPrivilegeEscalation: ptr.To(false)},
		}}},
		Parameters:      []buildapialpha.Parameter{{Name: "my-param", Description: "a plain param", Type: buildapialpha.ParameterTypeString, Default: ptr.To("default-value")}},
		SecurityContext: &buildapialpha.BuildStrategySecurityContext{RunAsUser: 1000, RunAsGroup: 1000},
		Volumes:         []buildapialpha.BuildStrategyVolume{{Name: "cache", Description: ptr.To("cache volume"), Overridable: ptr.To(false), VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
	}
}

func sampleAlphaBuild() *buildapialpha.Build {
	return &buildapialpha.Build{
		TypeMeta:   metav1.TypeMeta{Kind: "Build", APIVersion: "shipwright.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "my-build", Namespace: "build-ns", Labels: map[string]string{"app": "test"}, Annotations: map[string]string{"note": "hi"}},
		Spec: buildapialpha.BuildSpec{
			Source:      buildapialpha.Source{URL: ptr.To("https://github.com/shipwright-io/sample"), Revision: ptr.To("main"), ContextDir: ptr.To("src"), Credentials: &corev1.LocalObjectReference{Name: "git-secret"}},
			Strategy:    buildapialpha.Strategy{Name: "kaniko", Kind: ptr.To(buildapialpha.ClusterBuildStrategyKind)},
			ParamValues: alphaParamValues(), Dockerfile: ptr.To("Dockerfile.prod"), Builder: &buildapialpha.Image{Image: "golang:1.22"},
			Output:  buildapialpha.Image{Image: "quay.io/example/app:latest", Insecure: ptr.To(false), Credentials: &corev1.LocalObjectReference{Name: "push-secret"}, Annotations: map[string]string{"org.opencontainers.image.source": "git"}, Labels: map[string]string{"app": "example"}, Timestamp: ptr.To(buildapialpha.OutputImageZeroTimestamp)},
			Timeout: &metav1.Duration{Duration: 10 * time.Minute}, Env: []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
			Retention: &buildapialpha.BuildRetention{FailedLimit: ptr.To[uint](3), SucceededLimit: ptr.To[uint](5), TTLAfterFailed: &metav1.Duration{Duration: time.Hour}, TTLAfterSucceeded: &metav1.Duration{Duration: 2 * time.Hour}},
			Volumes:   []buildapialpha.BuildVolume{{Name: "cache", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
			Trigger:   &buildapialpha.Trigger{SecretRef: &corev1.LocalObjectReference{Name: "webhook-secret"}, When: []buildapialpha.TriggerWhen{{Name: "on-push", Type: buildapialpha.GitHubWebHookTrigger, GitHub: &buildapialpha.WhenGitHub{Events: []buildapialpha.GitHubEventName{buildapialpha.GitHubPushEvent}, Branches: []string{"main"}}}}},
		},
		Status: buildapialpha.BuildStatus{Registered: ptr.To(corev1.ConditionTrue), Reason: ptr.To(buildapialpha.SucceedStatus), Message: ptr.To(buildapialpha.AllValidationsSucceeded)},
	}
}

func sampleAlphaBuildRun() *buildapialpha.BuildRun {
	start := metav1.NewTime(time.Date(2024, 1, 2, 3, 0, 0, 0, time.UTC))
	end := metav1.NewTime(time.Date(2024, 1, 2, 4, 0, 0, 0, time.UTC))
	return &buildapialpha.BuildRun{
		TypeMeta:   metav1.TypeMeta{Kind: "BuildRun", APIVersion: "shipwright.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "my-buildrun", Namespace: "build-ns", Labels: map[string]string{"app": "test"}, Annotations: map[string]string{"note": "hi"}},
		Spec: buildapialpha.BuildRunSpec{
			BuildRef: &buildapialpha.BuildRef{Name: "my-build"}, ServiceAccount: &buildapialpha.ServiceAccount{Name: ptr.To("build-sa")},
			Sources: []buildapialpha.BuildSource{{Name: "upload", Type: buildapialpha.LocalCopy, Timeout: &metav1.Duration{Duration: 3 * time.Minute}}},
			Timeout: &metav1.Duration{Duration: 15 * time.Minute}, ParamValues: alphaParamValues(),
			Output: &buildapialpha.Image{Image: "quay.io/example/app:run", Insecure: ptr.To(true), Credentials: &corev1.LocalObjectReference{Name: "push-secret"}, Labels: map[string]string{"run": "yes"}, Annotations: map[string]string{"org.opencontainers.image.url": "https://example.com"}, Timestamp: ptr.To(buildapialpha.OutputImageSourceTimestamp)},
			State:  ptr.To(buildapialpha.BuildRunRequestedState(buildapialpha.BuildRunStateCancel)), Env: []corev1.EnvVar{{Name: "FOO", Value: "bar"}},
			Retention: &buildapialpha.BuildRunRetention{TTLAfterFailed: &metav1.Duration{Duration: time.Hour}, TTLAfterSucceeded: &metav1.Duration{Duration: 2 * time.Hour}},
			Volumes:   []buildapialpha.BuildVolume{{Name: "cache", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
		},
		Status: buildapialpha.BuildRunStatus{
			Sources:          []buildapialpha.SourceResult{{Name: "default", Git: &buildapialpha.GitSourceResult{CommitSha: "abc123", CommitAuthor: "dev", BranchName: "main"}, Timestamp: &start}},
			Output:           &buildapialpha.Output{Digest: "sha256:deadbeef", Size: 42},
			Conditions:       []buildapialpha.Condition{{Type: buildapialpha.Succeeded, Status: corev1.ConditionFalse, LastTransitionTime: end, Reason: "Failed", Message: "step failed"}},
			LatestTaskRunRef: ptr.To("my-buildrun-pod"), StartTime: &start, CompletionTime: &end, BuildSpec: ptr.To(sampleAlphaBuild().Spec),
			//nolint:staticcheck // This fixture verifies the deprecated alpha status field alongside its replacement.
			FailedAt:       &buildapialpha.FailedAt{Pod: "my-buildrun-pod", Container: "step-build"},
			FailureDetails: &buildapialpha.FailureDetails{Reason: "StepFailed", Message: "exit 1", Location: &buildapialpha.FailedAt{Pod: "my-buildrun-pod", Container: "step-build"}},
		},
	}
}

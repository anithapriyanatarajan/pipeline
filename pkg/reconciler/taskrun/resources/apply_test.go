/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	podtpl "github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
	"github.com/tektoncd/pipeline/pkg/workspace"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

var (
	simpleTaskSpec = &v1.TaskSpec{
		Sidecars: []v1.Sidecar{{
			Name:  "foo",
			Image: `$(params["myimage"])`,
			Env: []corev1.EnvVar{{
				Name:  "foo",
				Value: "$(params['FOO'])",
			}},
		}},
		StepTemplate: &v1.StepTemplate{
			Env: []corev1.EnvVar{{
				Name:  "template-var",
				Value: `$(params["FOO"])`,
			}},
			Image: "$(params.myimage)",
		},
		Steps: []v1.Step{{
			Name:  "foo",
			Image: "$(params.myimage)",
		}, {
			Name:       "baz",
			Image:      "bat",
			WorkingDir: "$(inputs.resources.workspace.path)",
			Args:       []string{"$(inputs.resources.workspace.url)"},
		}, {
			Name:  "foo",
			Image: `$(params["myimage"])`,
		}, {
			Name:       "baz",
			Image:      "$(params.somethingelse)",
			WorkingDir: "$(inputs.resources.workspace.path)",
			Args:       []string{"$(inputs.resources.workspace.url)"},
		}, {
			Name:  "foo",
			Image: "busybox:$(params.FOO)",
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "$(params.FOO)",
				MountPath: "path/to/$(params.FOO)",
				SubPath:   "sub/$(params.FOO)/path",
			}},
		}, {
			Name:  "foo",
			Image: "busybox:$(params.FOO)",
			Env: []corev1.EnvVar{{
				Name:  "foo",
				Value: "value-$(params.FOO)",
			}, {
				Name: "bar",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "config-$(params.FOO)"},
						Key:                  "config-key-$(params.FOO)",
					},
				},
			}, {
				Name: "baz",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "secret-$(params.FOO)"},
						Key:                  "secret-key-$(params.FOO)",
					},
				},
			}},
			EnvFrom: []corev1.EnvFromSource{{
				Prefix: "prefix-0-$(params.FOO)",
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "config-$(params.FOO)"},
				},
			}, {
				Prefix: "prefix-1-$(params.FOO)",
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "secret-$(params.FOO)"},
				},
			}},
		}},
		Volumes: []corev1.Volume{{
			Name: "$(params.FOO)",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "$(params.FOO)",
					},
					Items: []corev1.KeyToPath{{
						Key:  "$(params.FOO)",
						Path: "$(params.FOO)",
					}},
				},
			},
		}, {
			Name: "some-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "$(params.FOO)",
					Items: []corev1.KeyToPath{{
						Key:  "$(params.FOO)",
						Path: "$(params.FOO)",
					}},
				},
			},
		}, {
			Name: "some-pvc",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "$(params.FOO)",
				},
			},
		}, {
			Name: "some-projected-volumes",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{{
						ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "$(params.FOO)",
							},
						},
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "$(params.FOO)",
							},
						},
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Audience: "$(params.FOO)",
						},
					}},
				},
			},
		}, {
			Name: "some-csi",
			VolumeSource: corev1.VolumeSource{
				CSI: &corev1.CSIVolumeSource{
					VolumeAttributes: map[string]string{
						"secretProviderClass": "$(params.FOO)",
					},
					NodePublishSecretRef: &corev1.LocalObjectReference{
						Name: "$(params.FOO)",
					},
				},
			},
		}},
	}

	stepParamTaskSpec = &v1.TaskSpec{
		Params: v1.ParamSpecs{{
			Name: "myObject",
			Default: &v1.ParamValue{
				Type:      v1.ParamTypeObject,
				ObjectVal: map[string]string{"key1": "key1"},
			},
		}, {
			Name: "myString",
			Default: &v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "string-value",
			},
		}, {
			Name: "myArray",
			Default: &v1.ParamValue{
				Type:     v1.ParamTypeArray,
				ArrayVal: []string{"array", "value"},
			},
		}},
		Steps: []v1.Step{{
			Name: "foo",
			Ref: &v1.Ref{
				Name: "stepAction",
			},
			Params: v1.Params{{
				Name:  "myObject",
				Value: *v1.NewStructuredValues("$(params.myObject[*])"),
			}, {
				Name:  "myString",
				Value: *v1.NewStructuredValues("$(params.myString)"),
			}, {
				Name:  "myArray",
				Value: *v1.NewStructuredValues("$(params.myArray[*])"),
			}},
		}},
	}

	// a taskspec for testing object var in all places i.e. Sidecars, StepTemplate, Steps and Volumns
	objectParamTaskSpec = &v1.TaskSpec{
		Sidecars: []v1.Sidecar{{
			Name:  "foo",
			Image: `$(params.myObject.key1)`,
			Env: []corev1.EnvVar{{
				Name:  "foo",
				Value: "$(params.myObject.key2)",
			}},
		}},
		StepTemplate: &v1.StepTemplate{
			Image: "$(params.myObject.key1)",
			Env: []corev1.EnvVar{{
				Name:  "template-var",
				Value: `$(params.myObject.key2)`,
			}},
		},
		Steps: []v1.Step{{
			Name:       "foo",
			Image:      "$(params.myObject.key1)",
			WorkingDir: "path/to/$(params.myObject.key2)",
			Args:       []string{"first $(params.myObject.key1)", "second $(params.myObject.key2)"},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "$(params.myObject.key1)",
				MountPath: "path/to/$(params.myObject.key2)",
				SubPath:   "sub/$(params.myObject.key2)/path",
			}},
			Env: []corev1.EnvVar{{
				Name:  "foo",
				Value: "value-$(params.myObject.key1)",
			}, {
				Name: "bar",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "config-$(params.myObject.key1)"},
						Key:                  "config-key-$(params.myObject.key2)",
					},
				},
			}, {
				Name: "baz",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "secret-$(params.myObject.key1)"},
						Key:                  "secret-key-$(params.myObject.key2)",
					},
				},
			}},
			EnvFrom: []corev1.EnvFromSource{{
				Prefix: "prefix-0-$(params.myObject.key1)",
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "config-$(params.myObject.key1)"},
				},
			}, {
				Prefix: "prefix-1-$(params.myObject.key1)",
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "secret-$(params.myObject.key1)"},
				},
			}},
		}},
		Volumes: []corev1.Volume{{
			Name: "$(params.myObject.key1)",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "$(params.myObject.key1)",
					},
					Items: []corev1.KeyToPath{{
						Key:  "$(params.myObject.key1)",
						Path: "$(params.myObject.key2)",
					}},
				},
				Secret: &corev1.SecretVolumeSource{
					SecretName: "$(params.myObject.key1)",
					Items: []corev1.KeyToPath{{
						Key:  "$(params.myObject.key1)",
						Path: "$(params.myObject.key2)",
					}},
				},
			},
		}, {
			Name: "some-pvc",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "$(params.myObject.key1)",
				},
			},
		}, {
			Name: "some-projected-volumes",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{{
						ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "$(params.myObject.key1)",
							},
						},
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "$(params.myObject.key1)",
							},
						},
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Audience: "$(params.myObject.key2)",
						},
					}},
				},
			},
		}, {
			Name: "some-csi",
			VolumeSource: corev1.VolumeSource{
				CSI: &corev1.CSIVolumeSource{
					VolumeAttributes: map[string]string{
						"secretProviderClass": "$(params.myObject.key1)",
					},
					NodePublishSecretRef: &corev1.LocalObjectReference{
						Name: "$(params.myObject.key1)",
					},
				},
			},
		}},
	}

	simpleTaskSpecArrayIndexing = &v1.TaskSpec{
		Sidecars: []v1.Sidecar{{
			Name:  "foo",
			Image: `$(params["myimage"][0])`,
			Env: []corev1.EnvVar{{
				Name:  "foo",
				Value: "$(params['FOO'][1])",
			}},
		}},
		StepTemplate: &v1.StepTemplate{
			Env: []corev1.EnvVar{{
				Name:  "template-var",
				Value: `$(params["FOO"][1])`,
			}},
			Image: "$(params.myimage[0])",
		},
		Steps: []v1.Step{{
			Name:  "foo",
			Image: "$(params.myimage[0])",
		}, {
			Name:       "baz",
			Image:      "bat",
			WorkingDir: "$(inputs.resources.workspace.path)",
			Args:       []string{"$(inputs.resources.workspace.url)"},
		}, {
			Name:  "foo",
			Image: `$(params["myimage"][0])`,
		}, {
			Name:       "baz",
			Image:      "$(params.somethingelse)",
			WorkingDir: "$(inputs.resources.workspace.path)",
			Args:       []string{"$(inputs.resources.workspace.url)"},
		}, {
			Name:  "foo",
			Image: "busybox:$(params.FOO[1])",
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "$(params.FOO[1])",
				MountPath: "path/to/$(params.FOO[1])",
				SubPath:   "sub/$(params.FOO[1])/path",
			}},
		}, {
			Name:  "foo",
			Image: "busybox:$(params.FOO[1])",
			Env: []corev1.EnvVar{{
				Name:  "foo",
				Value: "value-$(params.FOO[1])",
			}, {
				Name: "bar",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "config-$(params.FOO[1])"},
						Key:                  "config-key-$(params.FOO[1])",
					},
				},
			}, {
				Name: "baz",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "secret-$(params.FOO[1])"},
						Key:                  "secret-key-$(params.FOO[1])",
					},
				},
			}},
			EnvFrom: []corev1.EnvFromSource{{
				Prefix: "prefix-0-$(params.FOO[1])",
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "config-$(params.FOO[1])"},
				},
			}, {
				Prefix: "prefix-1-$(params.FOO[1])",
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "secret-$(params.FOO[1])"},
				},
			}},
		}},
		Volumes: []corev1.Volume{{
			Name: "$(params.FOO[1])",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "$(params.FOO[1])",
					},
					Items: []corev1.KeyToPath{{
						Key:  "$(params.FOO[1])",
						Path: "$(params.FOO[1])",
					}},
				},
			},
		}, {
			Name: "some-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "$(params.FOO[1])",
					Items: []corev1.KeyToPath{{
						Key:  "$(params.FOO[1])",
						Path: "$(params.FOO[1])",
					}},
				},
			},
		}, {
			Name: "some-pvc",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "$(params.FOO[1])",
				},
			},
		}, {
			Name: "some-projected-volumes",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{{
						ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "$(params.FOO[1])",
							},
						},
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "$(params.FOO[1])",
							},
						},
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Audience: "$(params.FOO[1])",
						},
					}},
				},
			},
		}, {
			Name: "some-csi",
			VolumeSource: corev1.VolumeSource{
				CSI: &corev1.CSIVolumeSource{
					VolumeAttributes: map[string]string{
						"secretProviderClass": "$(params.FOO[1])",
					},
					NodePublishSecretRef: &corev1.LocalObjectReference{
						Name: "$(params.FOO[1])",
					},
				},
			},
		}},
	}

	arrayParamTaskSpec = &v1.TaskSpec{
		Steps: []v1.Step{{
			Name:  "simple-image",
			Image: "some-image",
		}, {
			Name:    "image-with-c-specified",
			Image:   "some-other-image",
			Command: []string{"echo"},
			Args:    []string{"first", "second", "$(params.array-param)", "last"},
		}},
	}

	arrayAndStringParamTaskSpec = &v1.TaskSpec{
		Steps: []v1.Step{{
			Name:  "simple-image",
			Image: "some-image",
		}, {
			Name:    "image-with-c-specified",
			Image:   "some-other-image",
			Command: []string{"echo"},
			Args:    []string{"$(params.normal-param)", "second", "$(params.array-param)", "last"},
		}},
	}

	arrayAndObjectParamTaskSpec = &v1.TaskSpec{
		Steps: []v1.Step{{
			Name:  "simple-image",
			Image: "some-image",
		}, {
			Name:    "image-with-c-specified",
			Image:   "some-other-image",
			Command: []string{"echo"},
			Args:    []string{"$(params.myObject.key1)", "$(params.myObject.key2)", "$(params.array-param)", "last"},
		}},
	}

	multipleArrayParamsTaskSpec = &v1.TaskSpec{
		Steps: []v1.Step{{
			Name:  "simple-image",
			Image: "some-image",
		}, {
			Name:    "image-with-c-specified",
			Image:   "some-other-image",
			Command: []string{"cmd", "$(params.another-array-param)"},
			Args:    []string{"first", "second", "$(params.array-param)", "last"},
		}},
	}

	multipleArrayAndStringsParamsTaskSpec = &v1.TaskSpec{
		Steps: []v1.Step{{
			Name:  "simple-image",
			Image: "image-$(params.string-param2)",
		}, {
			Name:    "image-with-c-specified",
			Image:   "some-other-image",
			Command: []string{"cmd", "$(params.array-param1)"},
			Args:    []string{"$(params.array-param2)", "second", "$(params.array-param1)", "$(params.string-param1)", "last"},
		}},
	}

	multipleArrayAndObjectParamsTaskSpec = &v1.TaskSpec{
		Steps: []v1.Step{{
			Name:  "simple-image",
			Image: "image-$(params.myObject.key1)",
		}, {
			Name:    "image-with-c-specified",
			Image:   "some-other-image",
			Command: []string{"cmd", "$(params.array-param1)"},
			Args:    []string{"$(params.array-param2)", "second", "$(params.array-param1)", "$(params.myObject.key2)", "last"},
		}},
	}

	arrayTaskRun0Elements = &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{
				{
					Name: "array-param",
					Value: v1.ParamValue{
						Type:     v1.ParamTypeArray,
						ArrayVal: []string{},
					},
				},
			},
		},
	}

	arrayTaskRun1Elements = &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "array-param",
				Value: *v1.NewStructuredValues("foo"),
			}},
		},
	}

	arrayTaskRun3Elements = &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "array-param",
				Value: *v1.NewStructuredValues("foo", "bar", "third"),
			}},
		},
	}

	arrayTaskRunMultipleArrays = &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "array-param",
				Value: *v1.NewStructuredValues("foo", "bar", "third"),
			}, {
				Name:  "another-array-param",
				Value: *v1.NewStructuredValues("part1", "part2"),
			}},
		},
	}

	arrayTaskRunWith1StringParam = &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "array-param",
				Value: *v1.NewStructuredValues("middlefirst", "middlesecond"),
			}, {
				Name:  "normal-param",
				Value: *v1.NewStructuredValues("foo"),
			}},
		},
	}

	arrayTaskRunWith1ObjectParam = &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "array-param",
				Value: *v1.NewStructuredValues("middlefirst", "middlesecond"),
			}, {
				Name: "myObject",
				Value: *v1.NewObject(map[string]string{
					"key1": "object value1",
					"key2": "object value2",
				}),
			}},
		},
	}

	arrayTaskRunMultipleArraysAndStrings = &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "array-param1",
				Value: *v1.NewStructuredValues("1-param1", "2-param1", "3-param1", "4-param1"),
			}, {
				Name:  "array-param2",
				Value: *v1.NewStructuredValues("1-param2", "2-param2", "2-param3"),
			}, {
				Name:  "string-param1",
				Value: *v1.NewStructuredValues("foo"),
			}, {
				Name:  "string-param2",
				Value: *v1.NewStructuredValues("bar"),
			}},
		},
	}

	arrayTaskRunMultipleArraysAndObject = &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "array-param1",
				Value: *v1.NewStructuredValues("1-param1", "2-param1", "3-param1", "4-param1"),
			}, {
				Name:  "array-param2",
				Value: *v1.NewStructuredValues("1-param2", "2-param2", "3-param3"),
			}, {
				Name: "myObject",
				Value: *v1.NewObject(map[string]string{
					"key1": "value1",
					"key2": "value2",
				}),
			}},
		},
	}
)

func applyMutation(ts *v1.TaskSpec, f func(*v1.TaskSpec)) *v1.TaskSpec {
	ts = ts.DeepCopy()
	f(ts)
	return ts
}

func TestApplyArrayParameters(t *testing.T) {
	type args struct {
		ts *v1.TaskSpec
		tr *v1.TaskRun
		dp []v1.ParamSpec
	}
	tests := []struct {
		name string
		args args
		want *v1.TaskSpec
	}{{
		name: "array parameter with 0 elements",
		args: args{
			ts: arrayParamTaskSpec,
			tr: arrayTaskRun0Elements,
		},
		want: applyMutation(arrayParamTaskSpec, func(spec *v1.TaskSpec) {
			spec.Steps[1].Args = []string{"first", "second", "last"}
		}),
	}, {
		name: "array parameter with 1 element",
		args: args{
			ts: arrayParamTaskSpec,
			tr: arrayTaskRun1Elements,
		},
		want: applyMutation(arrayParamTaskSpec, func(spec *v1.TaskSpec) {
			spec.Steps[1].Args = []string{"first", "second", "foo", "last"}
		}),
	}, {
		name: "array parameter with 3 elements",
		args: args{
			ts: arrayParamTaskSpec,
			tr: arrayTaskRun3Elements,
		},
		want: applyMutation(arrayParamTaskSpec, func(spec *v1.TaskSpec) {
			spec.Steps[1].Args = []string{"first", "second", "foo", "bar", "third", "last"}
		}),
	}, {
		name: "multiple arrays",
		args: args{
			ts: multipleArrayParamsTaskSpec,
			tr: arrayTaskRunMultipleArrays,
		},
		want: applyMutation(multipleArrayParamsTaskSpec, func(spec *v1.TaskSpec) {
			spec.Steps[1].Command = []string{"cmd", "part1", "part2"}
			spec.Steps[1].Args = []string{"first", "second", "foo", "bar", "third", "last"}
		}),
	}, {
		name: "array and normal string parameter",
		args: args{
			ts: arrayAndStringParamTaskSpec,
			tr: arrayTaskRunWith1StringParam,
		},
		want: applyMutation(arrayAndStringParamTaskSpec, func(spec *v1.TaskSpec) {
			spec.Steps[1].Args = []string{"foo", "second", "middlefirst", "middlesecond", "last"}
		}),
	}, {
		name: "several arrays and strings",
		args: args{
			ts: multipleArrayAndStringsParamsTaskSpec,
			tr: arrayTaskRunMultipleArraysAndStrings,
		},
		want: applyMutation(multipleArrayAndStringsParamsTaskSpec, func(spec *v1.TaskSpec) {
			spec.Steps[0].Image = "image-bar"
			spec.Steps[1].Command = []string{"cmd", "1-param1", "2-param1", "3-param1", "4-param1"}
			spec.Steps[1].Args = []string{"1-param2", "2-param2", "2-param3", "second", "1-param1", "2-param1", "3-param1", "4-param1", "foo", "last"}
		}),
	}, {
		name: "array and object parameter",
		args: args{
			ts: arrayAndObjectParamTaskSpec,
			tr: arrayTaskRunWith1ObjectParam,
		},
		want: applyMutation(arrayAndObjectParamTaskSpec, func(spec *v1.TaskSpec) {
			spec.Steps[1].Args = []string{"object value1", "object value2", "middlefirst", "middlesecond", "last"}
		}),
	}, {
		name: "several arrays and objects",
		args: args{
			ts: multipleArrayAndObjectParamsTaskSpec,
			tr: arrayTaskRunMultipleArraysAndObject,
		},
		want: applyMutation(multipleArrayAndObjectParamsTaskSpec, func(spec *v1.TaskSpec) {
			spec.Steps[0].Image = "image-value1"
			spec.Steps[1].Command = []string{"cmd", "1-param1", "2-param1", "3-param1", "4-param1"}
			spec.Steps[1].Args = []string{"1-param2", "2-param2", "3-param3", "second", "1-param1", "2-param1", "3-param1", "4-param1", "value2", "last"}
		}),
	}, {
		name: "default array parameter",
		args: args{
			ts: arrayParamTaskSpec,
			tr: &v1.TaskRun{},
			dp: []v1.ParamSpec{{
				Name:    "array-param",
				Default: v1.NewStructuredValues("defaulted", "value!"),
			}},
		},
		want: applyMutation(arrayParamTaskSpec, func(spec *v1.TaskSpec) {
			spec.Steps[1].Args = []string{"first", "second", "defaulted", "value!", "last"}
		}),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resources.ApplyParameters(tt.args.ts, tt.args.tr, tt.args.dp...)
			if d := cmp.Diff(tt.want, got); d != "" {
				t.Errorf("ApplyParameters() got diff %s", diff.PrintWantGot(d))
			}
		})
	}
}

func TestApplyParameters(t *testing.T) {
	tr := &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "myimage",
				Value: *v1.NewStructuredValues("bar"),
			}, {
				Name:  "FOO",
				Value: *v1.NewStructuredValues("world"),
			}},
			PodTemplate: &podtpl.Template{
				NodeSelector: map[string]string{
					"kubernetes.io/arch": "$(params.myimage)",
					"disktype":           "$(params.FOO)",
					"static":             "value",
				},
				Tolerations: []corev1.Toleration{{
					Key:      "$(params.myimage)",
					Value:    "$(params.FOO)",
					Operator: corev1.TolerationOpEqual,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				RuntimeClassName:  ptr.To("$(params.myimage)"),
				SchedulerName:     "$(params.FOO)",
				PriorityClassName: ptr.To("$(params.myimage)"),
				ImagePullSecrets: []corev1.LocalObjectReference{
					{Name: "$(params.myimage)"},
					{Name: "$(params.FOO)"},
				},
				HostAliases: []corev1.HostAlias{{
					IP:        "$(params.myimage)",
					Hostnames: []string{"$(params.FOO).example.com", "host2.example.com"},
				}},
				Env: []corev1.EnvVar{
					{Name: "$(params.myimage)", Value: "$(params.FOO)"},
					{Name: "STATIC_VAR", Value: "static_value"},
					{Name: "FROM_CONFIG", ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "config-$(params.myimage)"},
							Key:                  "config-key-$(params.FOO)",
						},
					}},
					{Name: "FROM_SECRET", ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "secret-$(params.myimage)"},
							Key:                  "secret-key-$(params.FOO)",
						},
					}},
					{Name: "FROM_FIELD", ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName.$(params.myimage)",
						},
					}},
					{Name: "FROM_RESOURCE", ValueFrom: &corev1.EnvVarSource{
						ResourceFieldRef: &corev1.ResourceFieldSelector{
							Resource:      "limits.memory.$(params.FOO)",
							ContainerName: "container-$(params.myimage)",
						},
					}},
				},
				DNSConfig: &corev1.PodDNSConfig{
					Nameservers: []string{"$(params.myimage)", "$(params.FOO)"},
					Searches:    []string{"$(params.FOO).local"},
					Options: []corev1.PodDNSConfigOption{
						{Name: "$(params.myimage)", Value: ptr.To("$(params.FOO)")},
					},
				},
				DNSPolicy: &[]corev1.DNSPolicy{corev1.DNSClusterFirst}[0],
				SecurityContext: &corev1.PodSecurityContext{
					SELinuxOptions: &corev1.SELinuxOptions{
						User:  "$(params.myimage)",
						Role:  "$(params.FOO)",
						Type:  "container_t",
						Level: "s0:c123,c456",
					},
					WindowsOptions: &corev1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: ptr.To("$(params.myimage)"),
						GMSACredentialSpec:     ptr.To("$(params.FOO)"),
						RunAsUserName:          ptr.To("$(params.myimage)"),
					},
					AppArmorProfile: &corev1.AppArmorProfile{
						Type:             corev1.AppArmorProfileTypeLocalhost,
						LocalhostProfile: ptr.To("$(params.myimage)"),
					},
					Sysctls: []corev1.Sysctl{{
						Name:  "$(params.myimage)",
						Value: "$(params.FOO)",
					}},
					FSGroupChangePolicy:      &[]corev1.PodFSGroupChangePolicy{corev1.FSGroupChangeOnRootMismatch}[0],
					SupplementalGroupsPolicy: &[]corev1.SupplementalGroupsPolicy{corev1.SupplementalGroupsPolicyMerge}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type:             corev1.SeccompProfileTypeLocalhost,
						LocalhostProfile: ptr.To("$(params.FOO)"),
					},
					SELinuxChangePolicy: &[]corev1.PodSELinuxChangePolicy{corev1.SELinuxChangePolicyMountOption}[0],
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "$(params.myimage)",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"$(params.FOO)"},
								}},
								MatchFields: []corev1.NodeSelectorRequirement{{
									Key:      "$(params.myimage)",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"$(params.FOO)"},
								}},
							}},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
							Weight: 1,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "$(params.myimage)",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"$(params.FOO)"},
								}},
							},
						}},
					},
					PodAffinity: &corev1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"$(params.myimage)": "$(params.FOO)",
								},
								MatchExpressions: []metav1.LabelSelectorRequirement{{
									Key:      "$(params.myimage)",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"$(params.FOO)"},
								}},
							},
							TopologyKey: "$(params.myimage)",
							Namespaces:  []string{"$(params.FOO)"},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"$(params.myimage)": "$(params.FOO)",
								},
							},
						}},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
							Weight: 1,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"$(params.myimage)": "$(params.FOO)",
									},
								},
								TopologyKey: "$(params.myimage)",
							},
						}},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"$(params.myimage)": "$(params.FOO)",
								},
							},
							TopologyKey: "$(params.myimage)",
						}},
					},
				},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       "$(params.myimage)",
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "$(params.FOO)",
						},
					},
				}},
				Volumes: []corev1.Volume{{
					Name: "$(params.myimage)",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "$(params.myimage)",
							},
							Items: []corev1.KeyToPath{{
								Key:  "$(params.myimage)",
								Path: "$(params.FOO)",
							}},
						},
					},
				}, {
					Name: "secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "$(params.myimage)",
							Items: []corev1.KeyToPath{{
								Key:  "$(params.myimage)",
								Path: "$(params.FOO)",
							}},
						},
					},
				}, {
					Name: "pvc-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "$(params.myimage)",
						},
					},
				}, {
					Name: "projected-volume",
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{{
								ConfigMap: &corev1.ConfigMapProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "$(params.myimage)",
									},
								},
								Secret: &corev1.SecretProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "$(params.myimage)",
									},
								},
								ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
									Audience: "$(params.FOO)",
								},
							}},
						},
					},
				}, {
					Name: "csi-volume",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							VolumeAttributes: map[string]string{
								"secretProviderClass": "$(params.myimage)",
							},
							NodePublishSecretRef: &corev1.LocalObjectReference{
								Name: "$(params.myimage)",
							},
						},
					},
				}},
			},
		},
	}
	dp := []v1.ParamSpec{{
		Name:    "something",
		Default: v1.NewStructuredValues("mydefault"),
	}, {
		Name:    "somethingelse",
		Default: v1.NewStructuredValues(""),
	}}
	want := applyMutation(simpleTaskSpec, func(spec *v1.TaskSpec) {
		spec.StepTemplate.Env[0].Value = "world"
		spec.StepTemplate.Image = "bar"

		spec.Steps[0].Image = "bar"
		spec.Steps[2].Image = "mydefault"
		spec.Steps[2].Image = "bar"
		spec.Steps[3].Image = ""

		spec.Steps[4].VolumeMounts[0].Name = "world"
		spec.Steps[4].VolumeMounts[0].SubPath = "sub/world/path"
		spec.Steps[4].VolumeMounts[0].MountPath = "path/to/world"
		spec.Steps[4].Image = "busybox:world"

		spec.Steps[5].Env[0].Value = "value-world"
		spec.Steps[5].Env[1].ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name = "config-world"
		spec.Steps[5].Env[1].ValueFrom.ConfigMapKeyRef.Key = "config-key-world"
		spec.Steps[5].Env[2].ValueFrom.SecretKeyRef.LocalObjectReference.Name = "secret-world"
		spec.Steps[5].Env[2].ValueFrom.SecretKeyRef.Key = "secret-key-world"
		spec.Steps[5].EnvFrom[0].Prefix = "prefix-0-world"
		spec.Steps[5].EnvFrom[0].ConfigMapRef.LocalObjectReference.Name = "config-world"
		spec.Steps[5].EnvFrom[1].Prefix = "prefix-1-world"
		spec.Steps[5].EnvFrom[1].SecretRef.LocalObjectReference.Name = "secret-world"
		spec.Steps[5].Image = "busybox:world"

		spec.Volumes[0].Name = "world"
		spec.Volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name = "world"
		spec.Volumes[0].VolumeSource.ConfigMap.Items[0].Key = "world"
		spec.Volumes[0].VolumeSource.ConfigMap.Items[0].Path = "world"
		spec.Volumes[1].VolumeSource.Secret.SecretName = "world"
		spec.Volumes[1].VolumeSource.Secret.Items[0].Key = "world"
		spec.Volumes[1].VolumeSource.Secret.Items[0].Path = "world"
		spec.Volumes[2].VolumeSource.PersistentVolumeClaim.ClaimName = "world"
		spec.Volumes[3].VolumeSource.Projected.Sources[0].ConfigMap.Name = "world"
		spec.Volumes[3].VolumeSource.Projected.Sources[0].Secret.Name = "world"
		spec.Volumes[3].VolumeSource.Projected.Sources[0].ServiceAccountToken.Audience = "world"
		spec.Volumes[4].VolumeSource.CSI.VolumeAttributes["secretProviderClass"] = "world"
		spec.Volumes[4].VolumeSource.CSI.NodePublishSecretRef.Name = "world"

		spec.Sidecars[0].Image = "bar"
		spec.Sidecars[0].Env[0].Value = "world"
	})
	wantTr := &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "myimage",
				Value: *v1.NewStructuredValues("bar"),
			}, {
				Name:  "FOO",
				Value: *v1.NewStructuredValues("world"),
			}},
			PodTemplate: &podtpl.Template{
				NodeSelector: map[string]string{
					"kubernetes.io/arch": "bar",
					"disktype":           "world",
					"static":             "value",
				},
				Tolerations: []corev1.Toleration{{
					Key:      "bar",
					Value:    "world",
					Operator: corev1.TolerationOpEqual,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				RuntimeClassName:  ptr.To("bar"),
				SchedulerName:     "world",
				PriorityClassName: ptr.To("bar"),
				ImagePullSecrets: []corev1.LocalObjectReference{
					{Name: "bar"},
					{Name: "world"},
				},
				HostAliases: []corev1.HostAlias{{
					IP:        "bar",
					Hostnames: []string{"world.example.com", "host2.example.com"},
				}},
				Env: []corev1.EnvVar{
					{Name: "bar", Value: "world"},
					{Name: "STATIC_VAR", Value: "static_value"},
					{Name: "FROM_CONFIG", ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "config-bar"},
							Key:                  "config-key-world",
						},
					}},
					{Name: "FROM_SECRET", ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "secret-bar"},
							Key:                  "secret-key-world",
						},
					}},
					{Name: "FROM_FIELD", ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName.bar",
						},
					}},
					{Name: "FROM_RESOURCE", ValueFrom: &corev1.EnvVarSource{
						ResourceFieldRef: &corev1.ResourceFieldSelector{
							Resource:      "limits.memory.world",
							ContainerName: "container-bar",
						},
					}},
				},
				DNSConfig: &corev1.PodDNSConfig{
					Nameservers: []string{"bar", "world"},
					Searches:    []string{"world.local"},
					Options: []corev1.PodDNSConfigOption{
						{Name: "bar", Value: ptr.To("world")},
					},
				},
				DNSPolicy: &[]corev1.DNSPolicy{corev1.DNSClusterFirst}[0],
				SecurityContext: &corev1.PodSecurityContext{
					SELinuxOptions: &corev1.SELinuxOptions{
						User:  "bar",
						Role:  "world",
						Type:  "container_t",
						Level: "s0:c123,c456",
					},
					WindowsOptions: &corev1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: ptr.To("bar"),
						GMSACredentialSpec:     ptr.To("world"),
						RunAsUserName:          ptr.To("bar"),
					},
					AppArmorProfile: &corev1.AppArmorProfile{
						Type:             corev1.AppArmorProfileTypeLocalhost,
						LocalhostProfile: ptr.To("bar"),
					},
					Sysctls: []corev1.Sysctl{{
						Name:  "bar",
						Value: "world",
					}},
					FSGroupChangePolicy:      &[]corev1.PodFSGroupChangePolicy{corev1.FSGroupChangeOnRootMismatch}[0],
					SupplementalGroupsPolicy: &[]corev1.SupplementalGroupsPolicy{corev1.SupplementalGroupsPolicyMerge}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type:             corev1.SeccompProfileTypeLocalhost,
						LocalhostProfile: ptr.To("world"),
					},
					SELinuxChangePolicy: &[]corev1.PodSELinuxChangePolicy{corev1.SELinuxChangePolicyMountOption}[0],
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "bar",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"world"},
								}},
								MatchFields: []corev1.NodeSelectorRequirement{{
									Key:      "bar",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"world"},
								}},
							}},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
							Weight: 1,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "bar",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"world"},
								}},
							},
						}},
					},
					PodAffinity: &corev1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"bar": "world",
								},
								MatchExpressions: []metav1.LabelSelectorRequirement{{
									Key:      "bar",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"world"},
								}},
							},
							TopologyKey: "bar",
							Namespaces:  []string{"world"},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"bar": "world",
								},
							},
						}},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
							Weight: 1,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"bar": "world",
									},
								},
								TopologyKey: "bar",
							},
						}},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"bar": "world",
								},
							},
							TopologyKey: "bar",
						}},
					},
				},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       "bar",
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "world",
						},
					},
				}},
				Volumes: []corev1.Volume{{
					Name: "bar",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "bar",
							},
							Items: []corev1.KeyToPath{{
								Key:  "bar",
								Path: "world",
							}},
						},
					},
				}, {
					Name: "secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "bar",
							Items: []corev1.KeyToPath{{
								Key:  "bar",
								Path: "world",
							}},
						},
					},
				}, {
					Name: "pvc-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "bar",
						},
					},
				}, {
					Name: "projected-volume",
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{{
								ConfigMap: &corev1.ConfigMapProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "bar",
									},
								},
								Secret: &corev1.SecretProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "bar",
									},
								},
								ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
									Audience: "world",
								},
							}},
						},
					},
				}, {
					Name: "csi-volume",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							VolumeAttributes: map[string]string{
								"secretProviderClass": "bar",
							},
							NodePublishSecretRef: &corev1.LocalObjectReference{
								Name: "bar",
							},
						},
					},
				}},
			},
		},
	}
	got := resources.ApplyParameters(simpleTaskSpec, tr, dp...)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("ApplyParameters() got diff %s", diff.PrintWantGot(d))
	}

	gotTr := tr.DeepCopy()
	gotTr.Spec.PodTemplate = resources.ApplyPodTemplateReplacements(tr.Spec.PodTemplate, tr)
	if d := cmp.Diff(wantTr, gotTr); d != "" {
		t.Errorf("ApplyPodTemplateParameters() got diff %s", diff.PrintWantGot(d))
	}
}

func TestApplyParameters_ArrayIndexing(t *testing.T) {
	tr := &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "myimage",
				Value: *v1.NewStructuredValues("bar", "foo"),
			}, {
				Name:  "FOO",
				Value: *v1.NewStructuredValues("hello", "world"),
			}},
			PodTemplate: &podtpl.Template{
				NodeSelector: map[string]string{
					"kubernetes.io/arch": "$(params.myimage[0])",
					"disktype":           "$(params.FOO[1])",
				},
				ImagePullSecrets: []corev1.LocalObjectReference{
					{Name: "$(params.myimage[0])"},
					{Name: "$(params.FOO[1])"},
				},
				Tolerations: []corev1.Toleration{{
					Key:      "$(params.myimage[0])",
					Value:    "$(params.FOO[1])",
					Operator: corev1.TolerationOpEqual,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				RuntimeClassName:  ptr.To("$(params.myimage[0])"),
				SchedulerName:     "$(params.FOO[1])",
				PriorityClassName: ptr.To("$(params.myimage[0])"),
				HostAliases: []corev1.HostAlias{{
					IP:        "$(params.myimage[0])",
					Hostnames: []string{"$(params.FOO[1]).example.com", "host2.example.com"},
				}},
				Env: []corev1.EnvVar{
					{Name: "$(params.myimage[0])", Value: "$(params.FOO[1])"},
					{Name: "STATIC_VAR", Value: "static_value"},
					{Name: "FROM_CONFIG", ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "config-$(params.myimage[0])"},
							Key:                  "config-key-$(params.FOO[1])",
						},
					}},
					{Name: "FROM_SECRET", ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "secret-$(params.myimage[0])"},
							Key:                  "secret-key-$(params.FOO[1])",
						},
					}},
					{Name: "FROM_FIELD", ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName.$(params.myimage[0])",
						},
					}},
					{Name: "FROM_RESOURCE", ValueFrom: &corev1.EnvVarSource{
						ResourceFieldRef: &corev1.ResourceFieldSelector{
							Resource:      "limits.memory.$(params.FOO[1])",
							ContainerName: "container-$(params.myimage[0])",
						},
					}},
				},
				DNSConfig: &corev1.PodDNSConfig{
					Nameservers: []string{"$(params.myimage[0])", "$(params.FOO[1])"},
					Searches:    []string{"$(params.FOO[1]).local"},
					Options: []corev1.PodDNSConfigOption{
						{Name: "$(params.myimage[0])", Value: ptr.To("$(params.FOO[1])")},
					},
				},
				DNSPolicy: &[]corev1.DNSPolicy{corev1.DNSClusterFirst}[0],
				SecurityContext: &corev1.PodSecurityContext{
					SELinuxOptions: &corev1.SELinuxOptions{
						User:  "$(params.myimage[0])",
						Role:  "$(params.FOO[1])",
						Type:  "container_t",
						Level: "s0:c123,c456",
					},
					WindowsOptions: &corev1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: ptr.To("$(params.myimage[0])"),
						GMSACredentialSpec:     ptr.To("$(params.FOO[1])"),
						RunAsUserName:          ptr.To("$(params.myimage[0])"),
					},
					AppArmorProfile: &corev1.AppArmorProfile{
						Type:             corev1.AppArmorProfileTypeLocalhost,
						LocalhostProfile: ptr.To("$(params.myimage[0])"),
					},
					Sysctls: []corev1.Sysctl{{
						Name:  "$(params.myimage[0])",
						Value: "$(params.FOO[1])",
					}},
					FSGroupChangePolicy:      &[]corev1.PodFSGroupChangePolicy{corev1.FSGroupChangeOnRootMismatch}[0],
					SupplementalGroupsPolicy: &[]corev1.SupplementalGroupsPolicy{corev1.SupplementalGroupsPolicyMerge}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type:             corev1.SeccompProfileTypeLocalhost,
						LocalhostProfile: ptr.To("$(params.FOO[1])"),
					},
					SELinuxChangePolicy: &[]corev1.PodSELinuxChangePolicy{corev1.SELinuxChangePolicyMountOption}[0],
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "$(params.myimage[0])",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"$(params.FOO[1])"},
								}},
								MatchFields: []corev1.NodeSelectorRequirement{{
									Key:      "$(params.myimage[0])",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"$(params.FOO[1])"},
								}},
							}},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
							Weight: 1,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "$(params.myimage[0])",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"$(params.FOO[1])"},
								}},
							},
						}},
					},
					PodAffinity: &corev1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"$(params.myimage[0])": "$(params.FOO[1])",
								},
								MatchExpressions: []metav1.LabelSelectorRequirement{{
									Key:      "$(params.myimage[0])",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"$(params.FOO[1])"},
								}},
							},
							TopologyKey: "$(params.myimage[0])",
							Namespaces:  []string{"$(params.FOO[1])"},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"$(params.myimage[0])": "$(params.FOO[1])",
								},
							},
						}},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
							Weight: 1,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"$(params.myimage[0])": "$(params.FOO[1])",
									},
								},
								TopologyKey: "$(params.myimage[0])",
							},
						}},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"$(params.myimage[0])": "$(params.FOO[1])",
								},
							},
							TopologyKey: "$(params.myimage[0])",
						}},
					},
				},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       "$(params.myimage[0])",
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "$(params.FOO[1])",
						},
					},
				}},
				Volumes: []corev1.Volume{{
					Name: "$(params.myimage[0])",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "$(params.myimage[0])",
							},
							Items: []corev1.KeyToPath{{
								Key:  "$(params.myimage[0])",
								Path: "$(params.FOO[1])",
							}},
						},
					},
				}, {
					Name: "secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "$(params.myimage[0])",
							Items: []corev1.KeyToPath{{
								Key:  "$(params.myimage[0])",
								Path: "$(params.FOO[1])",
							}},
						},
					},
				}, {
					Name: "pvc-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "$(params.myimage[0])",
						},
					},
				}, {
					Name: "projected-volume",
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{{
								ConfigMap: &corev1.ConfigMapProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "$(params.myimage[0])",
									},
								},
								Secret: &corev1.SecretProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "$(params.myimage[0])",
									},
								},
								ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
									Audience: "$(params.FOO[1])",
								},
							}},
						},
					},
				}, {
					Name: "csi-volume",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							VolumeAttributes: map[string]string{
								"secretProviderClass": "$(params.myimage[0])",
							},
							NodePublishSecretRef: &corev1.LocalObjectReference{
								Name: "$(params.myimage[0])",
							},
						},
					},
				}},
			},
		},
	}
	dp := []v1.ParamSpec{{
		Name:    "something",
		Default: v1.NewStructuredValues("mydefault", "mydefault2"),
	}, {
		Name:    "somethingelse",
		Default: v1.NewStructuredValues(""),
	}}
	want := applyMutation(simpleTaskSpec, func(spec *v1.TaskSpec) {
		spec.StepTemplate.Env[0].Value = "world"
		spec.StepTemplate.Image = "bar"

		spec.Steps[0].Image = "bar"
		spec.Steps[2].Image = "bar"
		spec.Steps[3].Image = ""

		spec.Steps[4].VolumeMounts[0].Name = "world"
		spec.Steps[4].VolumeMounts[0].SubPath = "sub/world/path"
		spec.Steps[4].VolumeMounts[0].MountPath = "path/to/world"
		spec.Steps[4].Image = "busybox:world"

		spec.Steps[5].Env[0].Value = "value-world"
		spec.Steps[5].Env[1].ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name = "config-world"
		spec.Steps[5].Env[1].ValueFrom.ConfigMapKeyRef.Key = "config-key-world"
		spec.Steps[5].Env[2].ValueFrom.SecretKeyRef.LocalObjectReference.Name = "secret-world"
		spec.Steps[5].Env[2].ValueFrom.SecretKeyRef.Key = "secret-key-world"
		spec.Steps[5].EnvFrom[0].Prefix = "prefix-0-world"
		spec.Steps[5].EnvFrom[0].ConfigMapRef.LocalObjectReference.Name = "config-world"
		spec.Steps[5].EnvFrom[1].Prefix = "prefix-1-world"
		spec.Steps[5].EnvFrom[1].SecretRef.LocalObjectReference.Name = "secret-world"
		spec.Steps[5].Image = "busybox:world"

		spec.Volumes[0].Name = "world"
		spec.Volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name = "world"
		spec.Volumes[0].VolumeSource.ConfigMap.Items[0].Key = "world"
		spec.Volumes[0].VolumeSource.ConfigMap.Items[0].Path = "world"
		spec.Volumes[1].VolumeSource.Secret.SecretName = "world"
		spec.Volumes[1].VolumeSource.Secret.Items[0].Key = "world"
		spec.Volumes[1].VolumeSource.Secret.Items[0].Path = "world"
		spec.Volumes[2].VolumeSource.PersistentVolumeClaim.ClaimName = "world"
		spec.Volumes[3].VolumeSource.Projected.Sources[0].ConfigMap.Name = "world"
		spec.Volumes[3].VolumeSource.Projected.Sources[0].Secret.Name = "world"
		spec.Volumes[3].VolumeSource.Projected.Sources[0].ServiceAccountToken.Audience = "world"
		spec.Volumes[4].VolumeSource.CSI.VolumeAttributes["secretProviderClass"] = "world"
		spec.Volumes[4].VolumeSource.CSI.NodePublishSecretRef.Name = "world"

		spec.Sidecars[0].Image = "bar"
		spec.Sidecars[0].Env[0].Value = "world"
	})
	wantTr := &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name:  "myimage",
				Value: *v1.NewStructuredValues("bar", "foo"),
			}, {
				Name:  "FOO",
				Value: *v1.NewStructuredValues("hello", "world"),
			}},
			PodTemplate: &podtpl.Template{
				NodeSelector: map[string]string{
					"kubernetes.io/arch": "bar",
					"disktype":           "world",
				},
				ImagePullSecrets: []corev1.LocalObjectReference{
					{Name: "bar"},
					{Name: "world"},
				},
				Tolerations: []corev1.Toleration{{
					Key:      "bar",
					Value:    "world",
					Operator: corev1.TolerationOpEqual,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				RuntimeClassName:  ptr.To("bar"),
				SchedulerName:     "world",
				PriorityClassName: ptr.To("bar"),
				HostAliases: []corev1.HostAlias{{
					IP:        "bar",
					Hostnames: []string{"world.example.com", "host2.example.com"},
				}},
				Env: []corev1.EnvVar{
					{Name: "bar", Value: "world"},
					{Name: "STATIC_VAR", Value: "static_value"},
					{Name: "FROM_CONFIG", ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "config-bar"},
							Key:                  "config-key-world",
						},
					}},
					{Name: "FROM_SECRET", ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "secret-bar"},
							Key:                  "secret-key-world",
						},
					}},
					{Name: "FROM_FIELD", ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName.bar",
						},
					}},
					{Name: "FROM_RESOURCE", ValueFrom: &corev1.EnvVarSource{
						ResourceFieldRef: &corev1.ResourceFieldSelector{
							Resource:      "limits.memory.world",
							ContainerName: "container-bar",
						},
					}},
				},
				DNSConfig: &corev1.PodDNSConfig{
					Nameservers: []string{"bar", "world"},
					Searches:    []string{"world.local"},
					Options: []corev1.PodDNSConfigOption{
						{Name: "bar", Value: ptr.To("world")},
					},
				},
				DNSPolicy: &[]corev1.DNSPolicy{corev1.DNSClusterFirst}[0],
				SecurityContext: &corev1.PodSecurityContext{
					SELinuxOptions: &corev1.SELinuxOptions{
						User:  "bar",
						Role:  "world",
						Type:  "container_t",
						Level: "s0:c123,c456",
					},
					WindowsOptions: &corev1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: ptr.To("bar"),
						GMSACredentialSpec:     ptr.To("world"),
						RunAsUserName:          ptr.To("bar"),
					},
					AppArmorProfile: &corev1.AppArmorProfile{
						Type:             corev1.AppArmorProfileTypeLocalhost,
						LocalhostProfile: ptr.To("bar"),
					},
					Sysctls: []corev1.Sysctl{{
						Name:  "bar",
						Value: "world",
					}},
					FSGroupChangePolicy:      &[]corev1.PodFSGroupChangePolicy{corev1.FSGroupChangeOnRootMismatch}[0],
					SupplementalGroupsPolicy: &[]corev1.SupplementalGroupsPolicy{corev1.SupplementalGroupsPolicyMerge}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type:             corev1.SeccompProfileTypeLocalhost,
						LocalhostProfile: ptr.To("world"),
					},
					SELinuxChangePolicy: &[]corev1.PodSELinuxChangePolicy{corev1.SELinuxChangePolicyMountOption}[0],
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "bar",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"world"},
								}},
								MatchFields: []corev1.NodeSelectorRequirement{{
									Key:      "bar",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"world"},
								}},
							}},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
							Weight: 1,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "bar",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"world"},
								}},
							},
						}},
					},
					PodAffinity: &corev1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"bar": "world",
								},
								MatchExpressions: []metav1.LabelSelectorRequirement{{
									Key:      "bar",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"world"},
								}},
							},
							TopologyKey: "bar",
							Namespaces:  []string{"world"},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"bar": "world",
								},
							},
						}},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
							Weight: 1,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"bar": "world",
									},
								},
								TopologyKey: "bar",
							},
						}},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"bar": "world",
								},
							},
							TopologyKey: "bar",
						}},
					},
				},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       "bar",
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "world",
						},
					},
				}},
				Volumes: []corev1.Volume{{
					Name: "bar",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "bar",
							},
							Items: []corev1.KeyToPath{{
								Key:  "bar",
								Path: "world",
							}},
						},
					},
				}, {
					Name: "secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "bar",
							Items: []corev1.KeyToPath{{
								Key:  "bar",
								Path: "world",
							}},
						},
					},
				}, {
					Name: "pvc-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "bar",
						},
					},
				}, {
					Name: "projected-volume",
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{{
								ConfigMap: &corev1.ConfigMapProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "bar",
									},
								},
								Secret: &corev1.SecretProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "bar",
									},
								},
								ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
									Audience: "world",
								},
							}},
						},
					},
				}, {
					Name: "csi-volume",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							VolumeAttributes: map[string]string{
								"secretProviderClass": "bar",
							},
							NodePublishSecretRef: &corev1.LocalObjectReference{
								Name: "bar",
							},
						},
					},
				}},
			},
		},
	}
	got := resources.ApplyParameters(simpleTaskSpecArrayIndexing, tr, dp...)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("ApplyParameters() got diff %s", diff.PrintWantGot(d))
	}

	// Test PodTemplate parameter substitution
	gotTr := tr.DeepCopy()
	gotTr.Spec.PodTemplate = resources.ApplyPodTemplateReplacements(tr.Spec.PodTemplate, tr)
	if d := cmp.Diff(wantTr, gotTr); d != "" {
		t.Errorf("ApplyPodTemplateParameters() got diff %s", diff.PrintWantGot(d))
	}
}

func TestApplyObjectParameters(t *testing.T) {
	// define the taskrun to test values provided by taskrun can overwrite the values provided in spec's default
	tr := &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name: "myObject",
				Value: *v1.NewObject(map[string]string{
					"key1": "taskrun-value-for-key1",
					"key2": "taskrun-value-for-key2",
				}),
			}},
			PodTemplate: &podtpl.Template{
				NodeSelector: map[string]string{
					"kubernetes.io/arch": "$(params.myObject.key1)",
					"zone":               "$(params.myObject.key2)",
					"static":             "value",
				},
				Tolerations: []corev1.Toleration{{
					Key:      "$(params.myObject.key1)",
					Value:    "$(params.myObject.key2)",
					Operator: corev1.TolerationOpEqual,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				RuntimeClassName:  ptr.To("$(params.myObject.key1)"),
				SchedulerName:     "$(params.myObject.key2)",
				PriorityClassName: ptr.To("$(params.myObject.key1)"),
				ImagePullSecrets: []corev1.LocalObjectReference{
					{Name: "$(params.myObject.key1)"},
					{Name: "$(params.myObject.key2)"},
				},
				HostAliases: []corev1.HostAlias{{
					IP:        "$(params.myObject.key1)",
					Hostnames: []string{"$(params.myObject.key2).example.com", "host2.example.com"},
				}},
				Env: []corev1.EnvVar{
					{Name: "DEPLOYMENT_ENV", Value: "$(params.myObject.key1)"},
					{Name: "VERSION", Value: "$(params.myObject.key2)"},
				},
				DNSConfig: &corev1.PodDNSConfig{
					Nameservers: []string{"$(params.myObject.key1)", "$(params.myObject.key2)"},
					Searches:    []string{"$(params.myObject.key2).local"},
					Options: []corev1.PodDNSConfigOption{
						{Name: "$(params.myObject.key1)", Value: ptr.To("$(params.myObject.key2)")},
					},
				},
				DNSPolicy: &[]corev1.DNSPolicy{corev1.DNSClusterFirst}[0],
				SecurityContext: &corev1.PodSecurityContext{
					SELinuxOptions: &corev1.SELinuxOptions{
						User:  "$(params.myObject.key1)",
						Role:  "$(params.myObject.key2)",
						Type:  "container_t",
						Level: "s0:c123,c456",
					},
					WindowsOptions: &corev1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: ptr.To("$(params.myObject.key1)"),
						RunAsUserName:          ptr.To("$(params.myObject.key2)"),
					},
					AppArmorProfile: &corev1.AppArmorProfile{
						Type:             corev1.AppArmorProfileTypeLocalhost,
						LocalhostProfile: ptr.To("$(params.myObject.key1)"),
					},
					Sysctls: []corev1.Sysctl{{
						Name:  "$(params.myObject.key1)",
						Value: "$(params.myObject.key2)",
					}},
					FSGroupChangePolicy:      &[]corev1.PodFSGroupChangePolicy{corev1.FSGroupChangeOnRootMismatch}[0],
					SupplementalGroupsPolicy: &[]corev1.SupplementalGroupsPolicy{corev1.SupplementalGroupsPolicyMerge}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type:             corev1.SeccompProfileTypeLocalhost,
						LocalhostProfile: ptr.To("$(params.myObject.key2)"),
					},
					SELinuxChangePolicy: &[]corev1.PodSELinuxChangePolicy{corev1.SELinuxChangePolicyMountOption}[0],
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "$(params.myObject.key1)",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"$(params.myObject.key2)"},
								}},
								MatchFields: []corev1.NodeSelectorRequirement{{
									Key:      "$(params.myObject.key1)",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"$(params.myObject.key2)"},
								}},
							}},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
							Weight: 1,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "$(params.myObject.key1)",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"$(params.myObject.key2)"},
								}},
							},
						}},
					},
					PodAffinity: &corev1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"$(params.myObject.key1)": "$(params.myObject.key2)",
								},
								MatchExpressions: []metav1.LabelSelectorRequirement{{
									Key:      "$(params.myObject.key1)",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"$(params.myObject.key2)"},
								}},
							},
							TopologyKey: "$(params.myObject.key1)",
							Namespaces:  []string{"$(params.myObject.key2)"},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"$(params.myObject.key1)": "$(params.myObject.key2)",
								},
							},
						}},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
							Weight: 1,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"$(params.myObject.key1)": "$(params.myObject.key2)",
									},
								},
								TopologyKey: "$(params.myObject.key1)",
							},
						}},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"$(params.myObject.key1)": "$(params.myObject.key2)",
								},
							},
							TopologyKey: "$(params.myObject.key1)",
						}},
					},
				},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       "$(params.myObject.key1)",
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "$(params.myObject.key2)",
						},
					},
				}},
			},
		},
	}
	dp := []v1.ParamSpec{{
		Name: "myObject",
		Default: v1.NewObject(map[string]string{
			"key1": "default-value-for-key1",
			"key2": "default-value-for-key2",
		}),
	}}

	want := applyMutation(objectParamTaskSpec, func(spec *v1.TaskSpec) {
		spec.Sidecars[0].Image = "taskrun-value-for-key1"
		spec.Sidecars[0].Env[0].Value = "taskrun-value-for-key2"

		spec.StepTemplate.Image = "taskrun-value-for-key1"
		spec.StepTemplate.Env[0].Value = "taskrun-value-for-key2"

		spec.Steps[0].Image = "taskrun-value-for-key1"
		spec.Steps[0].WorkingDir = "path/to/taskrun-value-for-key2"
		spec.Steps[0].Args = []string{"first taskrun-value-for-key1", "second taskrun-value-for-key2"}

		spec.Steps[0].VolumeMounts[0].Name = "taskrun-value-for-key1"
		spec.Steps[0].VolumeMounts[0].SubPath = "sub/taskrun-value-for-key2/path"
		spec.Steps[0].VolumeMounts[0].MountPath = "path/to/taskrun-value-for-key2"

		spec.Steps[0].Env[0].Value = "value-taskrun-value-for-key1"
		spec.Steps[0].Env[1].ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name = "config-taskrun-value-for-key1"
		spec.Steps[0].Env[1].ValueFrom.ConfigMapKeyRef.Key = "config-key-taskrun-value-for-key2"
		spec.Steps[0].Env[2].ValueFrom.SecretKeyRef.LocalObjectReference.Name = "secret-taskrun-value-for-key1"
		spec.Steps[0].Env[2].ValueFrom.SecretKeyRef.Key = "secret-key-taskrun-value-for-key2"
		spec.Steps[0].EnvFrom[0].Prefix = "prefix-0-taskrun-value-for-key1"
		spec.Steps[0].EnvFrom[0].ConfigMapRef.LocalObjectReference.Name = "config-taskrun-value-for-key1"
		spec.Steps[0].EnvFrom[1].Prefix = "prefix-1-taskrun-value-for-key1"
		spec.Steps[0].EnvFrom[1].SecretRef.LocalObjectReference.Name = "secret-taskrun-value-for-key1"

		spec.Volumes[0].Name = "taskrun-value-for-key1"
		spec.Volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name = "taskrun-value-for-key1"
		spec.Volumes[0].VolumeSource.ConfigMap.Items[0].Key = "taskrun-value-for-key1"
		spec.Volumes[0].VolumeSource.ConfigMap.Items[0].Path = "taskrun-value-for-key2"
		spec.Volumes[0].VolumeSource.Secret.SecretName = "taskrun-value-for-key1"
		spec.Volumes[0].VolumeSource.Secret.Items[0].Key = "taskrun-value-for-key1"
		spec.Volumes[0].VolumeSource.Secret.Items[0].Path = "taskrun-value-for-key2"
		spec.Volumes[1].VolumeSource.PersistentVolumeClaim.ClaimName = "taskrun-value-for-key1"
		spec.Volumes[2].VolumeSource.Projected.Sources[0].ConfigMap.Name = "taskrun-value-for-key1"
		spec.Volumes[2].VolumeSource.Projected.Sources[0].Secret.Name = "taskrun-value-for-key1"
		spec.Volumes[2].VolumeSource.Projected.Sources[0].ServiceAccountToken.Audience = "taskrun-value-for-key2"
		spec.Volumes[3].VolumeSource.CSI.VolumeAttributes["secretProviderClass"] = "taskrun-value-for-key1"
		spec.Volumes[3].VolumeSource.CSI.NodePublishSecretRef.Name = "taskrun-value-for-key1"
	})
	wantTr := &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name: "myObject",
				Value: *v1.NewObject(map[string]string{
					"key1": "taskrun-value-for-key1",
					"key2": "taskrun-value-for-key2",
				}),
			}},
			PodTemplate: &podtpl.Template{
				NodeSelector: map[string]string{
					"kubernetes.io/arch": "taskrun-value-for-key1",
					"zone":               "taskrun-value-for-key2",
					"static":             "value",
				},
				Tolerations: []corev1.Toleration{{
					Key:      "taskrun-value-for-key1",
					Value:    "taskrun-value-for-key2",
					Operator: corev1.TolerationOpEqual,
					Effect:   corev1.TaintEffectNoSchedule,
				}},
				RuntimeClassName:  ptr.To("taskrun-value-for-key1"),
				SchedulerName:     "taskrun-value-for-key2",
				PriorityClassName: ptr.To("taskrun-value-for-key1"),
				ImagePullSecrets: []corev1.LocalObjectReference{
					{Name: "taskrun-value-for-key1"},
					{Name: "taskrun-value-for-key2"},
				},
				HostAliases: []corev1.HostAlias{{
					IP:        "taskrun-value-for-key1",
					Hostnames: []string{"taskrun-value-for-key2.example.com", "host2.example.com"},
				}},
				Env: []corev1.EnvVar{
					{Name: "DEPLOYMENT_ENV", Value: "taskrun-value-for-key1"},
					{Name: "VERSION", Value: "taskrun-value-for-key2"},
				},
				DNSConfig: &corev1.PodDNSConfig{
					Nameservers: []string{"taskrun-value-for-key1", "taskrun-value-for-key2"},
					Searches:    []string{"taskrun-value-for-key2.local"},
					Options: []corev1.PodDNSConfigOption{
						{Name: "taskrun-value-for-key1", Value: ptr.To("taskrun-value-for-key2")},
					},
				},
				DNSPolicy: &[]corev1.DNSPolicy{corev1.DNSClusterFirst}[0],
				SecurityContext: &corev1.PodSecurityContext{
					SELinuxOptions: &corev1.SELinuxOptions{
						User:  "taskrun-value-for-key1",
						Role:  "taskrun-value-for-key2",
						Type:  "container_t",
						Level: "s0:c123,c456",
					},
					WindowsOptions: &corev1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: ptr.To("taskrun-value-for-key1"),
						RunAsUserName:          ptr.To("taskrun-value-for-key2"),
					},
					AppArmorProfile: &corev1.AppArmorProfile{
						Type:             corev1.AppArmorProfileTypeLocalhost,
						LocalhostProfile: ptr.To("taskrun-value-for-key1"),
					},
					Sysctls: []corev1.Sysctl{{
						Name:  "taskrun-value-for-key1",
						Value: "taskrun-value-for-key2",
					}},
					FSGroupChangePolicy:      &[]corev1.PodFSGroupChangePolicy{corev1.FSGroupChangeOnRootMismatch}[0],
					SupplementalGroupsPolicy: &[]corev1.SupplementalGroupsPolicy{corev1.SupplementalGroupsPolicyMerge}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type:             corev1.SeccompProfileTypeLocalhost,
						LocalhostProfile: ptr.To("taskrun-value-for-key2"),
					},
					SELinuxChangePolicy: &[]corev1.PodSELinuxChangePolicy{corev1.SELinuxChangePolicyMountOption}[0],
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "taskrun-value-for-key1",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"taskrun-value-for-key2"},
								}},
								MatchFields: []corev1.NodeSelectorRequirement{{
									Key:      "taskrun-value-for-key1",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"taskrun-value-for-key2"},
								}},
							}},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
							Weight: 1,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "taskrun-value-for-key1",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"taskrun-value-for-key2"},
								}},
							},
						}},
					},
					PodAffinity: &corev1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"taskrun-value-for-key1": "taskrun-value-for-key2",
								},
								MatchExpressions: []metav1.LabelSelectorRequirement{{
									Key:      "taskrun-value-for-key1",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"taskrun-value-for-key2"},
								}},
							},
							TopologyKey: "taskrun-value-for-key1",
							Namespaces:  []string{"taskrun-value-for-key2"},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"taskrun-value-for-key1": "taskrun-value-for-key2",
								},
							},
						}},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
							Weight: 1,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"taskrun-value-for-key1": "taskrun-value-for-key2",
									},
								},
								TopologyKey: "taskrun-value-for-key1",
							},
						}},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"taskrun-value-for-key1": "taskrun-value-for-key2",
								},
							},
							TopologyKey: "taskrun-value-for-key1",
						}},
					},
				},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       "taskrun-value-for-key1",
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "taskrun-value-for-key2",
						},
					},
				}},
			},
		},
	}
	got := resources.ApplyParameters(objectParamTaskSpec, tr, dp...)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("ApplyParameters() got diff %s", diff.PrintWantGot(d))
	}

	// Test PodTemplate parameter substitution
	gotTr := tr.DeepCopy()
	gotTr.Spec.PodTemplate = resources.ApplyPodTemplateReplacements(tr.Spec.PodTemplate, tr)
	if d := cmp.Diff(wantTr, gotTr); d != "" {
		t.Errorf("ApplyPodTemplateParameters() got diff %s", diff.PrintWantGot(d))
	}
}

func TestApplyStepParameters(t *testing.T) {
	// define the taskrun to test values provided by taskrun can overwrite the values provided in spec's default
	tr := &v1.TaskRun{
		Spec: v1.TaskRunSpec{
			Params: []v1.Param{{
				Name: "myObject",
				Value: *v1.NewObject(map[string]string{
					"key1": "taskrun-value-for-key1",
				}),
			}, {
				Name: "myString",
				Value: v1.ParamValue{
					Type:      v1.ParamTypeString,
					StringVal: "taskrun-string-value",
				},
			}, {
				Name: "myArray",
				Value: v1.ParamValue{
					Type:     v1.ParamTypeArray,
					ArrayVal: []string{"taskrun", "array", "value"},
				},
			}},
			TaskSpec: stepParamTaskSpec,
		},
	}
	dp := []v1.ParamSpec{{
		Name: "myObject",
		Default: &v1.ParamValue{
			Type:      v1.ParamTypeObject,
			ObjectVal: map[string]string{"key1": "key1"},
		},
	}, {
		Name: "myString",
		Default: &v1.ParamValue{
			Type:      v1.ParamTypeString,
			StringVal: "default-string-value",
		},
	}, {
		Name: "myArray",
		Default: &v1.ParamValue{
			Type:     v1.ParamTypeArray,
			ArrayVal: []string{"default", "array", "value"},
		},
	}}

	want := applyMutation(stepParamTaskSpec, func(spec *v1.TaskSpec) {
		spec.Steps[0].Params = []v1.Param{{
			Name: "myObject",
			Value: v1.ParamValue{
				Type:      v1.ParamTypeObject,
				ObjectVal: map[string]string{"key1": "taskrun-value-for-key1"},
			},
		}, {
			Name: "myString",
			Value: v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "taskrun-string-value",
			},
		}, {
			Name: "myArray",
			Value: v1.ParamValue{
				Type:     v1.ParamTypeArray,
				ArrayVal: []string{"taskrun", "array", "value"},
			},
		}}
	})
	got := resources.ApplyParameters(stepParamTaskSpec, tr, dp...)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("ApplyParameters() got diff %s", diff.PrintWantGot(d))
	}
}

func TestApplyWorkspaces(t *testing.T) {
	names.TestingSeed()
	ts := &v1.TaskSpec{
		StepTemplate: &v1.StepTemplate{
			Env: []corev1.EnvVar{{
				Name:  "template-var",
				Value: "$(workspaces.myws.volume)",
			}, {
				Name:  "pvc-name",
				Value: "$(workspaces.myws.claim)",
			}, {
				Name:  "non-pvc-name",
				Value: "$(workspaces.otherws.claim)",
			}},
		},
		Steps: []v1.Step{{
			Name:       "$(workspaces.myws.volume)",
			Image:      "$(workspaces.otherws.volume)",
			WorkingDir: "$(workspaces.otherws.volume)",
			Args:       []string{"$(workspaces.myws.path)"},
		}, {
			Name:  "foo",
			Image: "bar",
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "$(workspaces.myws.volume)",
				MountPath: "path/to/$(workspaces.otherws.path)",
				SubPath:   "$(workspaces.myws.volume)",
			}},
		}, {
			Name:  "foo",
			Image: "bar",
			Env: []corev1.EnvVar{{
				Name:  "foo",
				Value: "$(workspaces.myws.volume)",
			}, {
				Name: "baz",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "$(workspaces.myws.volume)"},
						Key:                  "$(workspaces.myws.volume)",
					},
				},
			}},
			EnvFrom: []corev1.EnvFromSource{{
				Prefix: "$(workspaces.myws.volume)",
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "$(workspaces.myws.volume)"},
				},
			}},
		}},
		Volumes: []corev1.Volume{{
			Name: "$(workspaces.myws.volume)",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "$(workspaces.myws.volume)",
					},
				},
			},
		}, {
			Name: "some-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "$(workspaces.myws.volume)",
				},
			},
		}, {
			Name: "some-pvc",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "$(workspaces.myws.volume)",
				},
			},
		}},
	}
	for _, tc := range []struct {
		name  string
		spec  *v1.TaskSpec
		decls []v1.WorkspaceDeclaration
		binds []v1.WorkspaceBinding
		want  *v1.TaskSpec
	}{{
		name: "workspace-variable-replacement",
		spec: ts.DeepCopy(),
		decls: []v1.WorkspaceDeclaration{{
			Name: "myws",
		}, {
			Name:      "otherws",
			MountPath: "/foo",
		}},
		binds: []v1.WorkspaceBinding{{
			Name: "myws",
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: "foo",
			},
		}, {
			Name:     "otherws",
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		}},
		want: applyMutation(ts, func(spec *v1.TaskSpec) {
			spec.StepTemplate.Env[0].Value = "ws-b31db"
			spec.StepTemplate.Env[1].Value = "foo"
			spec.StepTemplate.Env[2].Value = ""

			spec.Steps[0].Name = "ws-b31db"
			spec.Steps[0].Image = "ws-a6f34"
			spec.Steps[0].WorkingDir = "ws-a6f34"
			spec.Steps[0].Args = []string{"/workspace/myws"}

			spec.Steps[1].VolumeMounts[0].Name = "ws-b31db"
			spec.Steps[1].VolumeMounts[0].MountPath = "path/to//foo"
			spec.Steps[1].VolumeMounts[0].SubPath = "ws-b31db"

			spec.Steps[2].Env[0].Value = "ws-b31db"
			spec.Steps[2].Env[1].ValueFrom.SecretKeyRef.LocalObjectReference.Name = "ws-b31db"
			spec.Steps[2].Env[1].ValueFrom.SecretKeyRef.Key = "ws-b31db"
			spec.Steps[2].EnvFrom[0].Prefix = "ws-b31db"
			spec.Steps[2].EnvFrom[0].ConfigMapRef.LocalObjectReference.Name = "ws-b31db"

			spec.Volumes[0].Name = "ws-b31db"
			spec.Volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name = "ws-b31db"
			spec.Volumes[1].VolumeSource.Secret.SecretName = "ws-b31db"
			spec.Volumes[2].VolumeSource.PersistentVolumeClaim.ClaimName = "ws-b31db"
		}),
	}, {
		name: "optional-workspace-provided-variable-replacement",
		spec: &v1.TaskSpec{Steps: []v1.Step{{
			Script: `test "$(workspaces.ows.bound)" = "true" && echo "$(workspaces.ows.path)"`,
		}}},
		decls: []v1.WorkspaceDeclaration{{
			Name:     "ows",
			Optional: true,
		}},
		binds: []v1.WorkspaceBinding{{
			Name:     "ows",
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		}},
		want: &v1.TaskSpec{Steps: []v1.Step{{
			Script: `test "true" = "true" && echo "/workspace/ows"`,
		}}},
	}, {
		name: "optional-workspace-omitted-variable-replacement",
		spec: &v1.TaskSpec{Steps: []v1.Step{{
			Script: `test "$(workspaces.ows.bound)" = "true" && echo "$(workspaces.ows.path)"`,
		}}},
		decls: []v1.WorkspaceDeclaration{{
			Name:     "ows",
			Optional: true,
		}},
		binds: []v1.WorkspaceBinding{}, // intentionally omitted ows binding
		want: &v1.TaskSpec{Steps: []v1.Step{{
			Script: `test "false" = "true" && echo ""`,
		}}},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			vols := workspace.CreateVolumes(tc.binds)
			got := resources.ApplyWorkspaces(t.Context(), tc.spec, tc.decls, tc.binds, vols)
			if d := cmp.Diff(tc.want, got); d != "" {
				t.Errorf("TestApplyWorkspaces() got diff %s", diff.PrintWantGot(d))
			}
		})
	}
}

func TestApplyWorkspaces_IsolatedWorkspaces(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spec  *v1.TaskSpec
		decls []v1.WorkspaceDeclaration
		binds []v1.WorkspaceBinding
		want  *v1.TaskSpec
	}{{
		name: "step-workspace-with-custom-mountpath",
		spec: &v1.TaskSpec{Steps: []v1.Step{{
			Script: `echo "$(workspaces.ws.path)"`,
			Workspaces: []v1.WorkspaceUsage{{
				Name:      "ws",
				MountPath: "/foo",
			}},
		}, {
			Script: `echo "$(workspaces.ws.path)"`,
		}}, Sidecars: []v1.Sidecar{{
			Script: `echo "$(workspaces.ws.path)"`,
		}}},
		decls: []v1.WorkspaceDeclaration{{
			Name: "ws",
		}},
		want: &v1.TaskSpec{Steps: []v1.Step{{
			Script: `echo "/foo"`,
			Workspaces: []v1.WorkspaceUsage{{
				Name:      "ws",
				MountPath: "/foo",
			}},
		}, {
			Script: `echo "/workspace/ws"`,
		}}, Sidecars: []v1.Sidecar{{
			Script: `echo "/workspace/ws"`,
		}}},
	}, {
		name: "sidecar-workspace-with-custom-mountpath",
		spec: &v1.TaskSpec{Steps: []v1.Step{{
			Script: `echo "$(workspaces.ws.path)"`,
		}}, Sidecars: []v1.Sidecar{{
			Script: `echo "$(workspaces.ws.path)"`,
			Workspaces: []v1.WorkspaceUsage{{
				Name:      "ws",
				MountPath: "/bar",
			}},
		}}},
		decls: []v1.WorkspaceDeclaration{{
			Name: "ws",
		}},
		want: &v1.TaskSpec{Steps: []v1.Step{{
			Script: `echo "/workspace/ws"`,
		}}, Sidecars: []v1.Sidecar{{
			Script: `echo "/bar"`,
			Workspaces: []v1.WorkspaceUsage{{
				Name:      "ws",
				MountPath: "/bar",
			}},
		}}},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			vols := workspace.CreateVolumes(tc.binds)
			got := resources.ApplyWorkspaces(ctx, tc.spec, tc.decls, tc.binds, vols)
			if d := cmp.Diff(tc.want, got); d != "" {
				t.Errorf("TestApplyWorkspaces() got diff %s", diff.PrintWantGot(d))
			}
		})
	}
}

func TestContext(t *testing.T) {
	for _, tc := range []struct {
		description string
		taskName    string
		tr          v1.TaskRun
		spec        v1.TaskSpec
		want        v1.TaskSpec
	}{{
		description: "context taskName replacement without taskRun in spec container",
		taskName:    "Task1",
		tr:          v1.TaskRun{},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.task.name)-1",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "Task1-1",
			}},
		},
	}, {
		description: "context taskName replacement with taskRun in spec container",
		taskName:    "Task1",
		tr: v1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: "taskrunName",
			},
		},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.task.name)-1",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "Task1-1",
			}},
		},
	}, {
		description: "context taskRunName replacement with defined taskRun in spec container",
		taskName:    "Task1",
		tr: v1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: "taskrunName",
			},
		},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.taskRun.name)-1",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "taskrunName-1",
			}},
		},
	}, {
		description: "context taskRunName replacement with no defined taskRun name in spec container",
		taskName:    "Task1",
		tr:          v1.TaskRun{},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.taskRun.name)-1",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "-1",
			}},
		},
	}, {
		description: "context taskRun namespace replacement with no defined namepsace in spec container",
		taskName:    "Task1",
		tr:          v1.TaskRun{},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.taskRun.namespace)-1",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "-1",
			}},
		},
	}, {
		description: "context taskRun namespace replacement with defined namepsace in spec container",
		taskName:    "Task1",
		tr: v1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrunName",
				Namespace: "trNamespace",
			},
		},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.taskRun.namespace)-1",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "trNamespace-1",
			}},
		},
	}, {
		description: "context taskRunName replacement with no defined taskName in spec container",
		tr:          v1.TaskRun{},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.task.name)-1",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "-1",
			}},
		},
	}, {
		description: "context UID replacement",
		taskName:    "Task1",
		tr: v1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				UID: "UID-1",
			},
		},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.taskRun.uid)",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "UID-1",
			}},
		},
	}, {
		description: "context retry count replacement",
		tr: v1.TaskRun{
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					RetriesStatus: []v1.TaskRunStatus{{
						Status: duckv1.Status{
							Conditions: []apis.Condition{{
								Type:   apis.ConditionSucceeded,
								Status: corev1.ConditionFalse,
							}},
						},
					}, {
						Status: duckv1.Status{
							Conditions: []apis.Condition{{
								Type:   apis.ConditionSucceeded,
								Status: corev1.ConditionFalse,
							}},
						},
					}},
				},
			},
		},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.task.retry-count)-1",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "2-1",
			}},
		},
	}, {
		description: "context retry count replacement with task that never retries",
		tr:          v1.TaskRun{},
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "$(context.task.retry-count)-1",
			}},
		},
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Name:  "ImageName",
				Image: "0-1",
			}},
		},
	}} {
		t.Run(tc.description, func(t *testing.T) {
			got := resources.ApplyContexts(&tc.spec, tc.taskName, &tc.tr)
			if d := cmp.Diff(&tc.want, got); d != "" {
				t.Error(diff.PrintWantGot(d))
			}
		})
	}
}

func TestTaskResults(t *testing.T) {
	names.TestingSeed()
	ts := &v1.TaskSpec{
		Results: []v1.TaskResult{
			{
				Name:        "current.date.unix.timestamp",
				Description: "The current date in unix timestamp format",
			}, {
				Name:        "current-date-human-readable",
				Description: "The current date in humand readable format",
			},
		},
		Steps: []v1.Step{{
			Name:   "print-date-unix-timestamp",
			Image:  "bash:latest",
			Args:   []string{"$(results[\"current.date.unix.timestamp\"].path)"},
			Script: "#!/usr/bin/env bash\ndate +%s | tee $(results[\"current.date.unix.timestamp\"].path)",
		}, {
			Name:   "print-date-human-readable",
			Image:  "bash:latest",
			Script: "#!/usr/bin/env bash\ndate | tee $(results.current-date-human-readable.path)",
		}, {
			Name:   "print-date-human-readable-again",
			Image:  "bash:latest",
			Script: "#!/usr/bin/env bash\ndate | tee $(results['current-date-human-readable'].path)",
		}},
	}
	want := applyMutation(ts, func(spec *v1.TaskSpec) {
		spec.Steps[0].Script = "#!/usr/bin/env bash\ndate +%s | tee /tekton/results/current.date.unix.timestamp"
		spec.Steps[0].Args[0] = "/tekton/results/current.date.unix.timestamp"
		spec.Steps[1].Script = "#!/usr/bin/env bash\ndate | tee /tekton/results/current-date-human-readable"
		spec.Steps[2].Script = "#!/usr/bin/env bash\ndate | tee /tekton/results/current-date-human-readable"
	})
	got := resources.ApplyResults(ts)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("ApplyTaskResults() got diff %s", diff.PrintWantGot(d))
	}
}

func TestStepResults(t *testing.T) {
	names.TestingSeed()
	ts := &v1.TaskSpec{
		Steps: []v1.Step{{
			Name: "print-date-unix-timestamp",
			Results: []v1.StepResult{{
				Name:        "current.date.unix.timestamp",
				Description: "The current date in unix timestamp format",
			}},
			Image: "bash:latest",
			Args: []string{
				"$(step.results[\"current.date.unix.timestamp\"].path)",
			},
			Script: "#!/usr/bin/env bash\ndate +%s | tee $(step.results[\"current.date.unix.timestamp\"].path)",
		}, {
			Name: "print-date-human-readable",
			Results: []v1.StepResult{{
				Name:        "current-date-human-readable",
				Description: "The current date in humand readable format",
			}},
			Image:  "bash:latest",
			Script: "#!/usr/bin/env bash\ndate | tee $(step.results.current-date-human-readable.path)",
		}, {
			Name:  "print-date-human-readable-again",
			Image: "bash:latest",
			Results: []v1.StepResult{{
				Name:        "current-date-human-readable",
				Description: "The current date in humand readable format",
			}},
			Script: "#!/usr/bin/env bash\ndate | tee $(step.results['current-date-human-readable'].path)",
		}},
	}
	want := applyMutation(ts, func(spec *v1.TaskSpec) {
		spec.Steps[0].Script = "#!/usr/bin/env bash\ndate +%s | tee /tekton/steps/step-print-date-unix-timestamp/results/current.date.unix.timestamp"
		spec.Steps[0].Args[0] = "/tekton/steps/step-print-date-unix-timestamp/results/current.date.unix.timestamp"
		spec.Steps[1].Script = "#!/usr/bin/env bash\ndate | tee /tekton/steps/step-print-date-human-readable/results/current-date-human-readable"
		spec.Steps[2].Script = "#!/usr/bin/env bash\ndate | tee /tekton/steps/step-print-date-human-readable-again/results/current-date-human-readable"
	})
	got := resources.ApplyResults(ts)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("ApplyTaskResults() got diff %s", diff.PrintWantGot(d))
	}
}

func TestApplyStepExitCodePath(t *testing.T) {
	names.TestingSeed()
	ts := &v1.TaskSpec{
		Steps: []v1.Step{{
			Image:  "bash:latest",
			Script: "#!/usr/bin/env bash\nexit 11",
		}, {
			Name:   "failing-step",
			Image:  "bash:latest",
			Script: "#!/usr/bin/env bash\ncat $(steps.step-unnamed-0.exitCode.path)",
		}, {
			Name:   "check-failing-step",
			Image:  "bash:latest",
			Script: "#!/usr/bin/env bash\ncat $(steps.step-failing-step.exitCode.path)",
		}},
	}
	expected := applyMutation(ts, func(spec *v1.TaskSpec) {
		spec.Steps[1].Script = "#!/usr/bin/env bash\ncat /tekton/steps/step-unnamed-0/exitCode"
		spec.Steps[2].Script = "#!/usr/bin/env bash\ncat /tekton/steps/step-failing-step/exitCode"
	})
	got := resources.ApplyStepExitCodePath(ts)
	if d := cmp.Diff(expected, got); d != "" {
		t.Errorf("ApplyStepExitCodePath() got diff %s", diff.PrintWantGot(d))
	}
}

func TestApplyCredentialsPath(t *testing.T) {
	for _, tc := range []struct {
		description string
		spec        v1.TaskSpec
		path        string
		want        v1.TaskSpec
	}{{
		description: "replacement in spec container",
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Command: []string{"cp"},
				Args:    []string{"-R", "$(credentials.path)/", "$HOME"},
			}},
		},
		path: "/tekton/creds",
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Command: []string{"cp"},
				Args:    []string{"-R", "/tekton/creds/", "$HOME"},
			}},
		},
	}, {
		description: "replacement in spec Script",
		spec: v1.TaskSpec{
			Steps: []v1.Step{{
				Script: `cp -R "$(credentials.path)/" $HOME`,
			}},
		},
		path: "/tekton/home",
		want: v1.TaskSpec{
			Steps: []v1.Step{{
				Script: `cp -R "/tekton/home/" $HOME`,
			}},
		},
	}} {
		t.Run(tc.description, func(t *testing.T) {
			got := resources.ApplyCredentialsPath(&tc.spec, tc.path)
			if d := cmp.Diff(&tc.want, got); d != "" {
				t.Error(diff.PrintWantGot(d))
			}
		})
	}
}

func TestApplyParametersToWorkspaceBindings(t *testing.T) {
	tests := []struct {
		name string
		ts   *v1.TaskSpec
		tr   *v1.TaskRun
		want *v1.TaskRun
	}{
		{
			name: "pvc",
			ts: &v1.TaskSpec{
				Params: []v1.ParamSpec{
					{Name: "claim-name", Type: v1.ParamTypeString},
				},
			},
			tr: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "$(params.claim-name)",
							},
						},
					},
					Params: v1.Params{{Name: "claim-name", Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "claim-value",
					}}},
				},
			},
			want: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "claim-value",
							},
						},
					},
					Params: v1.Params{{Name: "claim-name", Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "claim-value",
					}}},
				},
			},
		},
		{
			name: "subPath",
			ts: &v1.TaskSpec{
				Params: []v1.ParamSpec{
					{Name: "subPath-name", Type: v1.ParamTypeString},
				},
			},
			tr: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							SubPath: "$(params.subPath-name)",
						},
					},
					Params: v1.Params{{Name: "subPath-name", Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "subPath-value",
					}}},
				},
			},
			want: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							SubPath: "subPath-value",
						},
					},
					Params: v1.Params{{Name: "subPath-name", Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "subPath-value",
					}}},
				},
			},
		},
		{
			name: "configMap",
			ts: &v1.TaskSpec{
				Params: []v1.ParamSpec{
					{Name: "configMap-name", Type: v1.ParamTypeString},
				},
			},
			tr: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "$(params.configMap-name)",
								},
							},
						},
					},
					Params: v1.Params{{Name: "configMap-name", Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "configMap-value",
					}}},
				},
			},
			want: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "configMap-value",
								},
							},
						},
					},
					Params: v1.Params{{Name: "configMap-name", Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "configMap-value",
					}}},
				},
			},
		},
		{
			name: "secret",
			ts: &v1.TaskSpec{
				Params: []v1.ParamSpec{
					{Name: "secret-name", Type: v1.ParamTypeString},
				},
			},
			tr: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "$(params.secret-name)",
							},
						},
					},
					Params: v1.Params{{Name: "secret-name", Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "secret-value",
					}}},
				},
			},
			want: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "secret-value",
							},
						},
					},
					Params: v1.Params{
						{Name: "secret-name", Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "secret-value",
						}},
					},
				},
			},
		},
		{
			name: "projected-sources-configMap",
			ts: &v1.TaskSpec{
				Params: []v1.ParamSpec{
					{Name: "proj-configMap-name", Type: v1.ParamTypeString},
				},
			},
			tr: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{
										ConfigMap: &corev1.ConfigMapProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "$(params.proj-configMap-name)",
											},
										},
									},
								},
							},
						},
					},
					Params: v1.Params{
						{
							Name: "proj-configMap-name", Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "proj-configMap-value",
							},
						},
					},
				},
			},
			want: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{
										ConfigMap: &corev1.ConfigMapProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "proj-configMap-value",
											},
										},
									},
								},
							},
						},
					},
					Params: v1.Params{
						{
							Name: "proj-configMap-name", Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "proj-configMap-value",
							},
						},
					},
				},
			},
		},
		{
			name: "projected-sources-secret",
			ts: &v1.TaskSpec{
				Params: []v1.ParamSpec{
					{Name: "proj-secret-name", Type: v1.ParamTypeString},
				},
			},
			tr: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{
										Secret: &corev1.SecretProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "$(params.proj-secret-name)",
											},
										},
									},
								},
							},
						},
					},
					Params: v1.Params{
						{
							Name: "proj-secret-name", Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "proj-secret-value",
							},
						},
					},
				},
			},
			want: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{
										Secret: &corev1.SecretProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "proj-secret-value",
											},
										},
									},
								},
							},
						},
					},
					Params: v1.Params{
						{
							Name: "proj-secret-name", Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "proj-secret-value",
							},
						},
					},
				},
			},
		},
		{
			name: "csi-driver",
			ts: &v1.TaskSpec{
				Params: []v1.ParamSpec{
					{Name: "csi-driver-name", Type: v1.ParamTypeString},
				},
			},
			tr: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							CSI: &corev1.CSIVolumeSource{Driver: "$(params.csi-driver-name)"},
						},
					},
					Params: v1.Params{
						{
							Name: "csi-driver-name", Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "csi-driver-value",
							},
						},
					},
				},
			},
			want: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							CSI: &corev1.CSIVolumeSource{Driver: "csi-driver-value"},
						},
					},
					Params: v1.Params{
						{
							Name: "csi-driver-name", Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "csi-driver-value",
							},
						},
					},
				},
			},
		},
		{
			name: "csi-nodePublishSecretRef-name",
			ts: &v1.TaskSpec{
				Params: []v1.ParamSpec{
					{Name: "csi-nodePublishSecretRef-name", Type: v1.ParamTypeString},
				},
			},
			tr: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							CSI: &corev1.CSIVolumeSource{NodePublishSecretRef: &corev1.LocalObjectReference{
								Name: "$(params.csi-nodePublishSecretRef-name)",
							}},
						},
					},
					Params: v1.Params{
						{
							Name: "csi-nodePublishSecretRef-name", Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "csi-nodePublishSecretRef-value",
							},
						},
					},
				},
			},
			want: &v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Workspaces: []v1.WorkspaceBinding{
						{
							CSI: &corev1.CSIVolumeSource{NodePublishSecretRef: &corev1.LocalObjectReference{
								Name: "csi-nodePublishSecretRef-value",
							}},
						},
					},
					Params: v1.Params{
						{
							Name: "csi-nodePublishSecretRef-name", Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "csi-nodePublishSecretRef-value",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resources.ApplyParametersToWorkspaceBindings(tt.ts, tt.tr)
			if d := cmp.Diff(got, tt.want); d != "" {
				t.Errorf("ApplyParametersToWorkspaceBindings() %v, diff %v", tt.name, d)
			}
		})
	}
}

func TestArtifacts(t *testing.T) {
	ts := &v1.TaskSpec{
		Steps: []v1.Step{
			{
				Name:  "name1",
				Image: "bash:latest",
				Args: []string{
					"$(step.artifacts.path)",
				},
				Script: "#!/usr/bin/env bash\n echo -n $(step.artifacts.path)",
			},
		},
	}

	want := applyMutation(ts, func(spec *v1.TaskSpec) {
		spec.Steps[0].Args[0] = "/tekton/steps/step-name1/artifacts/provenance.json"
		spec.Steps[0].Script = "#!/usr/bin/env bash\n echo -n /tekton/steps/step-name1/artifacts/provenance.json"
	})
	got := resources.ApplyArtifacts(ts)
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("ApplyArtifacts() got diff %s", diff.PrintWantGot(d))
	}
}

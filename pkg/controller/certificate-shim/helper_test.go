/*
Copyright 2020 The cert-manager Authors.

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

package shimhelper

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/cert-manager/cert-manager/test/unit/gen"
)

func Test_translateAnnotations(t *testing.T) {
	type testCase struct {
		crt           *cmapi.Certificate
		annotations   map[string]string
		mutate        func(*testCase)
		check         func(*assert.Assertions, *cmapi.Certificate)
		expectedError error
	}

	validAnnotations := func() map[string]string {
		return map[string]string{
			cmapi.CommonNameAnnotationKey:           "www.example.com",
			cmapi.DurationAnnotationKey:             "168h", // 1 week
			cmapi.RenewBeforeAnnotationKey:          "24h",
			cmapi.UsagesAnnotationKey:               "server auth,signing",
			cmapi.RevisionHistoryLimitAnnotationKey: "7",
		}
	}

	tests := map[string]testCase{
		"success": {
			crt:         gen.Certificate("example-cert"),
			annotations: validAnnotations(),
			check: func(a *assert.Assertions, crt *cmapi.Certificate) {
				a.Equal("www.example.com", crt.Spec.CommonName)
				a.Equal(&metav1.Duration{Duration: time.Hour * 24 * 7}, crt.Spec.Duration)
				a.Equal(&metav1.Duration{Duration: time.Hour * 24}, crt.Spec.RenewBefore)
				a.Equal([]cmapi.KeyUsage{cmapi.UsageServerAuth, cmapi.UsageSigning}, crt.Spec.Usages)
				a.Equal(pointer.Int32(7), crt.Spec.RevisionHistoryLimit)
			},
		},
		"nil annotations": {
			crt:         gen.Certificate("example-cert"),
			annotations: nil,
		},
		"empty annotations": {
			crt:         gen.Certificate("example-cert"),
			annotations: map[string]string{},
		},
		"nil certificate": {
			crt:           nil,
			annotations:   validAnnotations(),
			expectedError: errNilCertificate,
		},
		"bad duration": {
			crt:         gen.Certificate("example-cert"),
			annotations: validAnnotations(),
			mutate: func(tc *testCase) {
				tc.annotations[cmapi.DurationAnnotationKey] = "an un-parsable duration string"
			},
			expectedError: errInvalidIngressAnnotation,
		},
		"bad renewBefore": {
			crt:         gen.Certificate("example-cert"),
			annotations: validAnnotations(),
			mutate: func(tc *testCase) {
				tc.annotations[cmapi.RenewBeforeAnnotationKey] = "an un-parsable duration string"
			},
			expectedError: errInvalidIngressAnnotation,
		},
		"bad usages": {
			crt:         gen.Certificate("example-cert"),
			annotations: validAnnotations(),
			mutate: func(tc *testCase) {
				tc.annotations[cmapi.UsagesAnnotationKey] = "playing ping pong"
			},
			expectedError: errInvalidIngressAnnotation,
		},
		"bad usage list": {
			crt:         gen.Certificate("example-cert"),
			annotations: validAnnotations(),
			mutate: func(tc *testCase) {
				tc.annotations[cmapi.UsagesAnnotationKey] = "server auth,,signing"
			},
			expectedError: errInvalidIngressAnnotation,
		},
		"bad revision history limit": {
			crt:         gen.Certificate("example-cert"),
			annotations: validAnnotations(),
			mutate: func(tc *testCase) {
				tc.annotations[cmapi.RevisionHistoryLimitAnnotationKey] = "invalid revision history limit"
			},
			expectedError: errInvalidIngressAnnotation,
		},
		"zero revision history limit": {
			crt:         gen.Certificate("example-cert"),
			annotations: validAnnotations(),
			mutate: func(tc *testCase) {
				tc.annotations[cmapi.RevisionHistoryLimitAnnotationKey] = "0"
			},
			expectedError: errInvalidIngressAnnotation,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.mutate != nil {
				tc.mutate(&tc)
			}
			crt := tc.crt.DeepCopy()

			err := translateAnnotations(crt, tc.annotations)

			if tc.expectedError != nil {
				assertErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
			if tc.check != nil {
				tc.check(assert.New(t), crt)
			}
		})
	}
}

// assertErrorIs checks that the supplied error has the target error in its chain.
// TODO Upgrade to next release of testify package which has this built in.
func assertErrorIs(t *testing.T, err, target error) {
	if assert.Error(t, err) {
		assert.Truef(t, errors.Is(err, target), "unexpected error type. err: %v, target: %v", err, target)
	}
}

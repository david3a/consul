// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package types

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// update allows golden files to be updated based on the current output.
var update = flag.Bool("update", false, "update golden files")

func goldenError(t *testing.T, name string, actual string) {
	t.Helper()

	fpath := filepath.Join("testdata", name+".golden")

	if *update {
		require.NoError(t, os.WriteFile(fpath, []byte(actual), 0644))
	} else {
		expected, err := os.ReadFile(fpath)
		require.NoError(t, err)
		require.Equal(t, string(expected), actual)
	}
}

func TestErrorStrings(t *testing.T) {
	type testCase struct {
		err      error
		expected string
	}

	fakeWrappedErr := fmt.Errorf("fake test error")

	cases := map[string]error{
		"ErrDataParse": ErrDataParse{
			TypeName: "hashicorp.consul.catalog.v1alpha1.Service",
			Wrapped:  fakeWrappedErr,
		},
		"ErrInvalidField": ErrInvalidField{
			Name:    "host",
			Wrapped: fakeWrappedErr,
		},
		"ErrInvalidListElement": ErrInvalidListElement{
			Name:    "addresses",
			Index:   42,
			Wrapped: fakeWrappedErr,
		},
		"ErrInvalidMapKey": ErrInvalidMapKey{
			Map:     "ports",
			Key:     "http",
			Wrapped: fakeWrappedErr,
		},
		"ErrInvalidMapValue": ErrInvalidMapValue{
			Map:     "ports",
			Key:     "http",
			Wrapped: fakeWrappedErr,
		},
		"ErrInvalidWorkloadHostFormat": ErrInvalidWorkloadHostFormat{
			Host: "-foo-bar-",
		},
		"ErrInvalidNodeHostFormat": ErrInvalidNodeHostFormat{
			Host: "unix:///node.sock",
		},
		"ErrOwnerInvalid": ErrOwnerInvalid{
			ResourceType: ServiceType,
			OwnerType:    WorkloadType,
		},
		"ErrInvalidPortReference": ErrInvalidPortReference{
			Name: "http",
		},
		"ErrInvalidReferenceType": ErrInvalidReferenceType{
			AllowedType: WorkloadType,
		},
		"ErrVirtualPortReused": ErrVirtualPortReused{
			Index: 3,
			Value: 8080,
		},
		"errMissing":                    errMissing,
		"errEmpty":                      errEmpty,
		"errNotDNSLabel":                errNotDNSLabel,
		"errNotIPAddress":               errNotIPAddress,
		"errUnixSocketMultiport":        errUnixSocketMultiport,
		"errInvalidPhysicalPort":        errInvalidPhysicalPort,
		"errInvalidVirtualPort":         errInvalidVirtualPort,
		"errDNSWarningWeightOutOfRange": errDNSWarningWeightOutOfRange,
		"errDNSPassingWeightOutOfRange": errDNSPassingWeightOutOfRange,
		"errLocalityZoneNoRegion":       errLocalityZoneNoRegion,
		"errReferenceTenancyNotEqual":   errReferenceTenancyNotEqual,
	}

	for name, err := range cases {
		t.Run(name, func(t *testing.T) {
			goldenError(t, name, err.Error())
		})
	}
}

func TestErrorUnwrap(t *testing.T) {
	type testCase struct {
		err      error
		expected string
	}

	fakeWrappedErr := fmt.Errorf("fake test error")

	cases := map[string]error{
		"ErrDataParse": ErrDataParse{
			TypeName: "hashicorp.consul.catalog.v1alpha1.Service",
			Wrapped:  fakeWrappedErr,
		},
		"ErrInvalidField": ErrInvalidField{
			Name:    "host",
			Wrapped: fakeWrappedErr,
		},
		"ErrInvalidListElement": ErrInvalidListElement{
			Name:    "addresses",
			Index:   42,
			Wrapped: fakeWrappedErr,
		},
		"ErrInvalidMapKey": ErrInvalidMapKey{
			Map:     "ports",
			Key:     "http",
			Wrapped: fakeWrappedErr,
		},
		"ErrInvalidMapValue": ErrInvalidMapValue{
			Map:     "ports",
			Key:     "http",
			Wrapped: fakeWrappedErr,
		},
	}

	for name, err := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, fakeWrappedErr, errors.Unwrap(err))
		})
	}
}

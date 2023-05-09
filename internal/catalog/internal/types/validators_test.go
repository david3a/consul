// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package types

import (
	"errors"
	"strings"
	"testing"

	pbcatalog "github.com/hashicorp/consul/proto-public/pbcatalog/v1alpha1"
	"github.com/hashicorp/consul/proto-public/pbresource"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestIsValidDNSLabel(t *testing.T) {
	type testCase struct {
		name  string
		valid bool
	}

	cases := map[string]testCase{
		"min-length": {
			name:  "a",
			valid: true,
		},
		"max-length": {
			name:  strings.Repeat("a1b2c3d4", 8),
			valid: true,
		},
		"underscores": {
			name:  "has_underscores",
			valid: true,
		},
		"hyphenated": {
			name:  "has-hyphen3",
			valid: true,
		},
		"uppercase-not-allowed": {
			name:  "UPPERCASE",
			valid: false,
		},
		"numeric-start": {
			name:  "1abc",
			valid: true,
		},
		"underscore-start-not-allowed": {
			name:  "_abc",
			valid: false,
		},
		"hyphen-start-not-allowed": {
			name:  "-abc",
			valid: false,
		},
		"underscore-end-not-allowed": {
			name:  "abc_",
			valid: false,
		},
		"hyphen-end-not-allowed": {
			name:  "abc-",
			valid: false,
		},
		"unicode-not allowed": {
			name:  "abc∑",
			valid: false,
		},
		"too-long": {
			name:  strings.Repeat("aaaaaaaaa", 8),
			valid: false,
		},
		"missing-name": {
			name:  "",
			valid: false,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tcase.valid, isValidDNSLabel(tcase.name))
		})
	}
}

func TestIsValidDNSName(t *testing.T) {
	// TestIsValidDNSLabel tests all of the individual label matching
	// criteria. This test therefore is more limited to just the extra
	// validations that IsValidDNSName does. Mainly that it ensures
	// the overall length is less than 256 and that generally is made
	// up of DNS labels joined with '.'

	type testCase struct {
		name  string
		valid bool
	}

	cases := map[string]testCase{
		"valid": {
			name:  "foo-something.example3.com",
			valid: true,
		},
		"exceeds-max-length": {
			name:  strings.Repeat("aaaa.aaaa", 29),
			valid: false,
		},
		"invalid-label": {
			name:  "foo._bar.com",
			valid: false,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tcase.valid, isValidDNSName(tcase.name))
		})
	}
}

func TestIsValidIPAddress(t *testing.T) {
	type testCase struct {
		name  string
		valid bool
	}

	cases := map[string]testCase{
		"ipv4": {
			name:  "192.168.1.2",
			valid: true,
		},
		"ipv6": {
			name:  "2001:db8::1",
			valid: true,
		},
		"ipv4-mapped-ipv6": {
			name:  "::ffff:192.0.2.128",
			valid: true,
		},
		"invalid": {
			name:  "foo",
			valid: false,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tcase.valid, isValidIPAddress(tcase.name))
		})
	}
}

func TestIsValidUnixSocketPath(t *testing.T) {
	type testCase struct {
		name  string
		valid bool
	}

	cases := map[string]testCase{
		"valid": {
			name:  "unix:///foo/bar.sock",
			valid: true,
		},
		"missing-prefix": {
			name:  "/foo/bar.sock",
			valid: false,
		},
		"nul-in-name": {
			name:  "unix:///foo/bar\000sock",
			valid: false,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tcase.valid, isValidUnixSocketPath(tcase.name))
		})
	}
}

func TestValidateHost(t *testing.T) {
	type testCase struct {
		name  string
		valid bool
	}

	cases := map[string]testCase{
		"ip-address": {
			name:  "198.18.0.1",
			valid: true,
		},
		"unix-socket": {
			name:  "unix:///foo.sock",
			valid: true,
		},
		"dns-name": {
			name:  "foo.com",
			valid: true,
		},
		"host-empty": {
			name:  "",
			valid: false,
		},
		"host-invalid": {
			name:  "unix:///foo/bar\000sock",
			valid: false,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateWorkloadHost(tcase.name)
			if tcase.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, ErrInvalidWorkloadHostFormat{Host: tcase.name}, err)
			}
		})
	}
}

func TestValidateSelector(t *testing.T) {
	type testCase struct {
		selector   *pbcatalog.WorkloadSelector
		allowEmpty bool
		err        error
	}

	cases := map[string]testCase{
		"nil-disallowed": {
			selector:   nil,
			allowEmpty: false,
			err:        errEmpty,
		},
		"nil-allowed": {
			selector:   nil,
			allowEmpty: true,
			err:        nil,
		},
		"empty-allowed": {
			selector:   &pbcatalog.WorkloadSelector{},
			allowEmpty: true,
			err:        nil,
		},
		"empty-disallowed": {
			selector:   &pbcatalog.WorkloadSelector{},
			allowEmpty: false,
			err:        errEmpty,
		},
		"ok": {
			selector: &pbcatalog.WorkloadSelector{
				Names:    []string{"foo", "bar"},
				Prefixes: []string{"foo", "bar"},
			},
			allowEmpty: false,
			err:        nil,
		},
		"empty-name": {
			selector: &pbcatalog.WorkloadSelector{
				Names:    []string{"", "bar", ""},
				Prefixes: []string{"foo", "bar"},
			},
			allowEmpty: false,
			err: &multierror.Error{
				Errors: []error{
					ErrInvalidListElement{
						Name:    "names",
						Index:   0,
						Wrapped: errEmpty,
					},
					ErrInvalidListElement{
						Name:    "names",
						Index:   2,
						Wrapped: errEmpty,
					},
				},
			},
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateSelector(tcase.selector, tcase.allowEmpty)
			if tcase.err == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, tcase.err, err)
			}
		})
	}
}

func TestValidateIPAddress(t *testing.T) {
	// this test does not perform extensive validation of what constitutes
	// an IP address. Instead that is handled in the test for the
	// isValidIPAddress function

	t.Run("empty", func(t *testing.T) {
		require.Equal(t, errEmpty, validateIPAddress(""))
	})

	t.Run("invalid", func(t *testing.T) {
		require.Equal(t, errNotIPAddress, validateIPAddress("foo.com"))
	})

	t.Run("ok", func(t *testing.T) {
		require.NoError(t, validateIPAddress("192.168.0.1"))
	})
}

func TestValidatePortName(t *testing.T) {
	// this test does not perform extensive validation of what constitutes
	// a valid port name. In general the criteria is that it must not
	// be empty and must be a valid DNS label. Therefore extensive testing
	// of what it means to be a valid DNS label is performed within the
	// test for the isValidDNSLabel function.

	t.Run("empty", func(t *testing.T) {
		require.Equal(t, errEmpty, validatePortName(""))
	})

	t.Run("invalid", func(t *testing.T) {
		require.Equal(t, errNotDNSLabel, validatePortName("foo.com"))
	})

	t.Run("ok", func(t *testing.T) {
		require.NoError(t, validatePortName("http"))
	})
}

func TestValidateWorkloadAddress(t *testing.T) {
	type testCase struct {
		addr        *pbcatalog.WorkloadAddress
		ports       map[string]struct{}
		validateErr func(*testing.T, error)
	}

	cases := map[string]testCase{
		"invalid-host": {
			addr: &pbcatalog.WorkloadAddress{
				Host: "-blarg",
			},
			ports: map[string]struct{}{},
			validateErr: func(t *testing.T, err error) {
				var actual ErrInvalidField
				require.True(t, errors.As(err, &actual))
				require.Equal(t, "host", actual.Name)
			},
		},
		"unix-socket-multiport-explicit": {
			addr: &pbcatalog.WorkloadAddress{
				Host:  "unix:///foo.sock",
				Ports: []string{"foo", "bar"},
			},
			ports: map[string]struct{}{
				"foo": {},
				"bar": {},
			},
			validateErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, errUnixSocketMultiport))
			},
		},
		"unix-socket-multiport-implicit": {
			addr: &pbcatalog.WorkloadAddress{
				Host: "unix:///foo.sock",
			},
			ports: map[string]struct{}{
				"foo": {},
				"bar": {},
			},
			validateErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, errUnixSocketMultiport))
			},
		},

		"unix-socket-singleport": {
			addr: &pbcatalog.WorkloadAddress{
				Host:  "unix:///foo.sock",
				Ports: []string{"foo"},
			},
			ports: map[string]struct{}{
				"foo": {},
				"bar": {},
			},
		},
		"invalid-port-reference": {
			addr: &pbcatalog.WorkloadAddress{
				Host:  "198.18.0.1",
				Ports: []string{"foo"},
			},
			ports: map[string]struct{}{
				"http": {},
			},
			validateErr: func(t *testing.T, err error) {
				var actual ErrInvalidPortReference
				require.True(t, errors.As(err, &actual))
				require.Equal(t, "foo", actual.Name)
			},
		},
		"ok": {
			addr: &pbcatalog.WorkloadAddress{
				Host:     "198.18.0.1",
				Ports:    []string{"http", "grpc"},
				External: true,
			},
			ports: map[string]struct{}{
				"http": {},
				"grpc": {},
			},
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateWorkloadAddress(tcase.addr, tcase.ports)
			if tcase.validateErr != nil {
				require.Error(t, err)
				tcase.validateErr(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateReferenceType(t *testing.T) {
	allowedType := &pbresource.Type{
		Group:        "foo",
		GroupVersion: "v1",
		Kind:         "Bar",
	}

	type testCase struct {
		check *pbresource.Type
		err   bool
	}

	cases := map[string]testCase{
		"ok": {
			check: allowedType,
			err:   false,
		},
		"group-mismatch": {
			check: &pbresource.Type{
				Group:        "food",
				GroupVersion: "v1",
				Kind:         "Bar",
			},
			err: true,
		},
		"group-version-mismatch": {
			check: &pbresource.Type{
				Group:        "foo",
				GroupVersion: "v2",
				Kind:         "Bar",
			},
			err: true,
		},
		"kind-mismatch": {
			check: &pbresource.Type{
				Group:        "foo",
				GroupVersion: "v1",
				Kind:         "Baz",
			},
			err: true,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateReferenceType(allowedType, tcase.check)
			if tcase.err {
				require.Equal(t, ErrInvalidReferenceType{AllowedType: allowedType}, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateReferenceTenancy(t *testing.T) {
	allowedTenancy := &pbresource.Tenancy{
		Partition: "default",
		Namespace: "default",
		PeerName:  "local",
	}

	type testCase struct {
		check *pbresource.Tenancy
		err   bool
	}

	cases := map[string]testCase{
		"ok": {
			check: allowedTenancy,
			err:   false,
		},
		"partition-mismatch": {
			check: &pbresource.Tenancy{
				Partition: "food",
				Namespace: "default",
				PeerName:  "local",
			},
			err: true,
		},
		"group-version-mismatch": {
			check: &pbresource.Tenancy{
				Partition: "default",
				Namespace: "v2",
				PeerName:  "local",
			},
			err: true,
		},
		"kind-mismatch": {
			check: &pbresource.Tenancy{
				Partition: "default",
				Namespace: "default",
				PeerName:  "Baz",
			},
			err: true,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateReferenceTenancy(allowedTenancy, tcase.check)
			if tcase.err {
				require.Equal(t, errReferenceTenancyNotEqual, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateReference(t *testing.T) {
	allowedTenancy := &pbresource.Tenancy{
		Partition: "default",
		Namespace: "default",
		PeerName:  "local",
	}

	allowedType := WorkloadType

	type testCase struct {
		check *pbresource.ID
		err   error
	}

	cases := map[string]testCase{
		"ok": {
			check: &pbresource.ID{
				Type:    allowedType,
				Tenancy: allowedTenancy,
				Name:    "foo",
			},
		},
		"type-err": {
			check: &pbresource.ID{
				Type:    NodeType,
				Tenancy: allowedTenancy,
				Name:    "foo",
			},
			err: ErrInvalidReferenceType{AllowedType: allowedType},
		},
		"tenancy-mismatch": {
			check: &pbresource.ID{
				Type: allowedType,
				Tenancy: &pbresource.Tenancy{
					Partition: "foo",
					Namespace: "bar",
					PeerName:  "baz",
				},
				Name: "foo",
			},
			err: errReferenceTenancyNotEqual,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateReference(allowedType, allowedTenancy, tcase.check)
			if tcase.err != nil {
				require.True(t, errors.Is(err, tcase.err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

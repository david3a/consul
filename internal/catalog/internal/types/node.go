// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package types

import (
	"github.com/hashicorp/consul/internal/resource"
	pbcatalog "github.com/hashicorp/consul/proto-public/pbcatalog/v1alpha1"
	"github.com/hashicorp/consul/proto-public/pbresource"
	"github.com/hashicorp/go-multierror"
)

const (
	NodeKind = "Node"
)

var (
	NodeV1Alpha1Type = &pbresource.Type{
		Group:        GroupName,
		GroupVersion: VersionV1Alpha1,
		Kind:         NodeKind,
	}

	NodeType = NodeV1Alpha1Type
)

func RegisterNode(r resource.Registry) {
	r.Register(resource.Registration{
		Type:     NodeV1Alpha1Type,
		Proto:    &pbcatalog.Node{},
		Validate: ValidateNode,
	})
}

func ValidateNode(res *pbresource.Resource) error {
	var node pbcatalog.Node

	if err := res.Data.UnmarshalTo(&node); err != nil {
		return newErrDataParse(&node, err)
	}

	var err error
	// Validate that the node has at least 1 address
	if len(node.Addresses) < 1 {
		err = multierror.Append(err, ErrInvalidField{
			Name:    "addresses",
			Wrapped: errEmpty,
		})
	}

	// Validate each node address
	for idx, addr := range node.Addresses {
		if addrErr := validateNodeAddress(addr); addrErr != nil {
			err = multierror.Append(err, ErrInvalidListElement{
				Name:    "addresses",
				Index:   idx,
				Wrapped: addrErr,
			})
		}
	}

	return err
}

// Validates a single node address
func validateNodeAddress(addr *pbcatalog.NodeAddress) error {
	// Currently the only field needing validation is the Host field. If that
	// changes then this func should get updated to use a multierror.Append
	// to collect the errors and return the overall set.

	// Check that the host is empty
	if addr.Host == "" {
		return ErrInvalidField{
			Name:    "host",
			Wrapped: errMissing,
		}
	}

	// Check if the host represents an IP address, unix socket path or a DNS name
	if !isValidIPAddress(addr.Host) && !isValidDNSName(addr.Host) {
		return ErrInvalidField{
			Name:    "host",
			Wrapped: ErrInvalidNodeHostFormat{Host: addr.Host},
		}
	}

	return nil
}

package platformclient

import (
	"context"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
	productinfov1 "code-code.internal/go-contract/product_info/v1"
)

func (s *SupportResources) ListVendors(ctx context.Context) ([]*supportv1.Vendor, error) {
	client, err := s.client.requireSupport()
	if err != nil {
		return nil, err
	}
	response, err := client.ListVendors(ctx, &supportv1.ListVendorsRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (s *SupportResources) ListCLIs(ctx context.Context) ([]*supportv1.CLI, error) {
	client, err := s.client.requireSupport()
	if err != nil {
		return nil, err
	}
	response, err := client.ListCLIs(ctx, &supportv1.ListCLIsRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (s *SupportResources) ListProductInfos(ctx context.Context) ([]*productinfov1.ProductInfo, error) {
	client, err := s.client.requireSupport()
	if err != nil {
		return nil, err
	}
	response, err := client.ListProductInfos(ctx, &supportv1.ListProductInfosRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

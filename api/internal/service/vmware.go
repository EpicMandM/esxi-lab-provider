package service

import (
	"context"
	"fmt"

	"net/url"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/soap"
)

type VMwareService struct {
	client *govmomi.Client
	finder *find.Finder
}

func NewVMwareService(ctx context.Context, vcenterURL, username, password string, insecure bool) (*VMwareService, error) {
	u, err := soap.ParseURL(vcenterURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	u.User = url.UserPassword(username, password)

	client, err := govmomi.NewClient(ctx, u, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	finder := find.NewFinder(client.Client, true)
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get datacenter: %w", err)
	}
	finder.SetDatacenter(dc)

	return &VMwareService{
		client: client,
		finder: finder,
	}, nil
}
func (s *VMwareService) GetFinder() *find.Finder {
	return s.finder
}
func (s *VMwareService) GetClient() *govmomi.Client {
	return s.client
}
func (s *VMwareService) Close(ctx context.Context) error {
	if s.client == nil {
		return s.client.Logout(ctx)
	}
	return nil
}

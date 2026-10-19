package hexcore

import (
	"context"
	"net/http"
)

// RegionsService lists deployment locations.
type RegionsService struct{ client *Client }

// List retrieves all documented deployment regions.
func (s *RegionsService) List(ctx context.Context) ([]Region, error) {
	return requestList[Region](ctx, s.client, http.MethodGet, "/regions", nil, http.StatusOK, "regions")
}

// SizesService lists standard VDS plans.
type SizesService struct{ client *Client }

// List retrieves all standard sizes available to the account.
func (s *SizesService) List(ctx context.Context) ([]Size, error) {
	return requestList[Size](ctx, s.client, http.MethodGet, "/sizes", nil, http.StatusOK, "sizes")
}

// ImagesService lists OS images.
type ImagesService struct{ client *Client }

// List retrieves all available OS images.
func (s *ImagesService) List(ctx context.Context) ([]Image, error) {
	return requestList[Image](ctx, s.client, http.MethodGet, "/images", nil, http.StatusOK, "images")
}

// ApplicationsService lists images with preinstalled applications.
type ApplicationsService struct{ client *Client }

// List retrieves all available application images.
func (s *ApplicationsService) List(ctx context.Context) ([]Application, error) {
	return requestList[Application](ctx, s.client, http.MethodGet, "/applications", nil, http.StatusOK, "applications")
}

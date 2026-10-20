package recipes

import (
	"context"
	"fmt"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

// #region regions
func PrintRegions(parent context.Context, client *hexcore.Client) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	regions, err := client.Regions.List(ctx)
	if err != nil {
		return err
	}
	for _, region := range regions {
		if region.Available {
			fmt.Printf("%s: %s\n", region.Slug, region.Name)
		}
	}
	return nil
}

// #endregion regions

// #region sizes
func PrintPlans(parent context.Context, client *hexcore.Client) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	sizes, err := client.Sizes.List(ctx)
	if err != nil {
		return err
	}
	for _, size := range sizes {
		fmt.Printf("%s: %d CPU, %d MB RAM, %d GB disk, %s RUB/month\n",
			size.Slug, size.VCPU, size.Memory, size.Disk, size.MonthPrice)
	}
	return nil
}

// #endregion sizes

// #region images
func PrintImages(parent context.Context, client *hexcore.Client) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	images, err := client.Images.List(ctx)
	if err != nil {
		return err
	}
	for _, image := range images {
		fmt.Printf("%s: %s %s (disk >= %d GB)\n",
			image.Slug, image.Name, image.Version, image.MinDiskSize)
	}
	return nil
}

// #endregion images

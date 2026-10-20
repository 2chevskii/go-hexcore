# Catalogs

Use the catalogs to choose the slugs needed for server creation.

| Method | Returns | Useful fields |
| --- | --- | --- |
| `Regions.List(ctx)` | `([]Region, error)` | `Slug`, `Name`, `Available` |
| `Sizes.List(ctx)` | `([]Size, error)` | `Slug`, `VCPU`, `Memory`, `Disk`, `MonthPrice`, `HourPrice` |
| `Images.List(ctx)` | `([]Image, error)` | `Slug`, `Name`, `Version`, `MinDiskSize` |
| `Applications.List(ctx)` | `([]Application, error)` | Same fields as `Image`, for images with preinstalled software. |

## Find available regions

<<< @/examples/recipes/catalogs.go#regions

## Compare server plans

<<< @/examples/recipes/catalogs.go#sizes

Memory is in MB, disk space is in GB, and prices are strings in rubles.

## Choose an image

<<< @/examples/recipes/catalogs.go#images

Use the image's `Slug` in your creation request. Choose a disk at least as large
as `MinDiskSize`. For application images, call `Applications.List(ctx)` instead.

See [catalog types](https://pkg.go.dev/github.com/2chevskii/go-hexcore#Region)
for all fields.

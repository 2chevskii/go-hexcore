# Data types and units

## IDs and units

IDs are strings, even when the API sends numbers. Pass them unchanged to other
methods. Quantities such as CPU count, memory, and disk size are `int64`.

| Field | Unit |
| --- | --- |
| `CustomSize.Memory` | GB in creation requests. |
| `VDS.Memory`, `Size.Memory` | MB in responses; `1` requested GB becomes `1024` MB. |
| VDS and disk storage sizes, image `MinDiskSize` | GB. |
| `VCPU` | CPU count. |

`Disks.Resize` takes the new total size, not the amount to add.

## Money and dates

Balances, prices, and discounts stay as strings to preserve decimal values.
Check `Account.Currency` for the account currency; catalog prices are in rubles.
Use decimal arithmetic if you need to calculate with them.

Dates and timestamps also stay as strings. Choose a parsing layout and timezone
that match the returned value; the library does not assume a timezone.

## Optional fields

Pointer fields can be nil when the API omits them or sends null. Check before
reading them:

<<< @/examples/recipes/reads.go#optional

`VDS.Get` may omit status and region that `VDS.List` includes. It does not make
another request to fill them in. Disk attachment fields can also be nil; treat
them as unknown rather than attached or detached.

Successful empty arrays become empty slices. Failed requests remain errors,
including the [403 responses used for some empty lists](./compatibility.md).

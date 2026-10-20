# Live API checks

Run offline checks first. Then read the [known API limits](../docs/compatibility.md)
and [verification record](LIVE_VERIFICATION.md). Live writes require permission
to spend account funds; additional disks need manual cleanup.

## Read checks

Set `HEXCORE_LIVE=1` and `HEXCORE_TOKEN` in your environment, then run:

```sh
go test -run '^TestLiveReads$' -v -count=1
```

Tests do not load `.env`. Set `HEXCORE_BASE_URL` only if you need another endpoint.

## Provision one VDS

Also set `HEXCORE_LIVE_WRITES=1` and choose `HEXCORE_REGION`, `HEXCORE_SIZE`, and
`HEXCORE_IMAGE` from the catalogs:

```sh
go test -run '^TestLiveVDSProvisioning$' -v -count=1 -timeout 25m
```

This creates, reads, and deletes one disposable server. Creation is attempted
once. Cleanup uses its unique name if the response is lost or cannot be decoded.

## Check power and password actions

With the same settings, run:

```sh
go test -run '^TestLiveVDSActions$' -v -count=1 -timeout 45m
```

To check only graceful reboot on a fresh server:

```sh
go test -run 'TestLiveVDSActions/^reboot$' -v -count=1 -timeout 45m
```

Each command creates and cleans up one disposable VDS. Dependent checks stop
after a failure.

## Run the broader write suite

Set these additional values:

| Variable | Value |
| --- | --- |
| `HEXCORE_CUSTOM_VCPU` | CPU count |
| `HEXCORE_CUSTOM_MEMORY` | Memory in GB |
| `HEXCORE_CUSTOM_DISK` | VDS disk size in GB |
| `HEXCORE_DISK_SIZE` | Additional disk size in GB; the test grows it by 1 |

```sh
go test -run '^TestLiveWrites$' -v -count=1 -timeout 45m
```

This creates two VDS sequentially and one additional disk, then checks creation,
power actions, password reset, disk operations, and deletion. Only one VDS is
allocated at a time. Passwords are generated in memory. Action waits allow ten
minutes; readiness waits allow five.

## Check cleanup

Cleanup attempts to unplug the disk, halt each test VDS, and destroy it, polling
resource state independently of action history. It uses separate disk and VDS
deadlines and reconciles unique names when a creation response is missing.

Remove every reported disk ID through the control panel after the run. If the
process is interrupted or discovery fails, inspect the printed resource names
and complete cleanup manually. Only resources created by that run may be deleted.

Accepted or ambiguously failed writes are not repeated. The harness only retries
guest commands explicitly rejected with `Action not allowed by VDS.` while
waiting for guest readiness. Stop on insufficient funds or unresolved API errors.
Record results separately from offline tests and sanitize any saved responses.

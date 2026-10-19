# Live verification: 2026-10-19

Phase 2 remains incomplete because disk attachment, graceful reboot, and
password-reset completion could not be verified end to end. The remaining
issues need clarification from HexCore before more billable tests.

## Verified behavior

- Account retrieval and all four catalogs decode successfully.
- VDS, action, and disk listing and singular retrieval decode successfully.
- Standard and custom VDS creation succeed. Custom memory `1` produces
  `1024` MB. VDS names are silently truncated to 45 characters.
- Hard shutdown, power-on, hard reboot, and graceful shutdown completed on a
  fresh Debian 12 server, with each command following a completed predecessor.
- Destruction requires a halted VDS, returns `202`, and refunds the unused
  portion of the reserved monthly payment. Removal was verified by listing.
- Insufficient funds produce an explicit `403` rejection.
- Disk creation produces a retrievable, billed record. Attachment is unverified.

Live account responses use an object envelope. Action submissions return
`202` instead of the documented `200`; destruction returns `202` instead of
`204`. The client now accepts these observed forms as well as the documented
ones. Nested API error messages are retained, with their original HTTP status.

## Disk blocker

Disk **207**, named `go-hexcore-test-50fd13d162922cc1-disk`, was created with
size `1`, region `ru-msk-2`, and numeric `vds_id: 100439`. Creation was accepted.
Subsequent reads returned both attachment fields as null:

```json
{"status":{"vdsId":null,"status":null}}
```

The documented plug request (`{"type":"plug"}`) and a resize request to
size `2` both returned HTTP `403`:

```json
{"message":"Can not get Disk status."}
```

The disk stayed at 1 GB. Adding a target ID to rejected plug requests did not
resolve the error. No such undocumented parameter was added to the SDK.

The documentation's cURL example quotes both `size` and `vds_id`, unlike its
integer schema. A literal request in that form was rejected with HTTP `403`,
`Size is not int.`; no disk was created by that attempt. Numeric size is required.

A final check used numeric size `1` and quoted `vds_id: "100443"` against a
fresh running VDS. Disk **208**, named
`go-hexcore-test-969e36577f80c90b-disk-string`, was accepted. Its attachment fields
also stayed null, and its plug request returned the same `Can not get Disk
status.` error. Both numeric and quoted target IDs have now been checked.
Changing the SDK's numeric encoding has no demonstrated benefit.

Read-only inspection of disk **207** in the signed-in control panel found an empty disk UUID
in its control URL and `vhdUUID` form field. The attachment form requires both
the disk UUID and a selected VDS UUID before submitting. A VDS was available
for selection, but the disk identifier was empty. No panel attachment request
was submitted. This is evidence of an incomplete disk record; the server-side
cause is unknown and needs clarification from HexCore.

## Action observations

The final isolated checks used fresh Debian 12 VDS in `ru-msk-2`:

| Action ID | Operation | VDS | Result |
| --- | --- | --- | --- |
| 92679 | hard_shutdown | 100440 | completed in 4 sec |
| 92680 | power_on | 100440 | completed in 5 sec |
| 92681 | hard_reboot | 100440 | completed in 7 sec |
| 92682 | shutdown | 100440 | completed in 4 sec |
| 92683 | power_on | 100440 | completed in 5 sec |
| 92684 | reset_password | 100440 | in-progress after a 10-minute wait |
| 92687 | reboot | 100441 | in-progress after a 10-minute wait |

The password-reset record started at `2026-10-19T08:30:47+03:00`. The reboot
record started at `2026-10-19T08:42:43+03:00` on a separate fresh server, without
any earlier action on that fixture. Both records were still in progress at the
final check after cleanup. The guest-side effects remain unknown.

Some commands in an earlier exploratory run overlapped; those results are not
used to infer isolated operation failures. One earlier hard shutdown completed
after 6 min 5 sec, so the harness now allows ten minutes for action completion.
Dependent checks stop after failure, and cleanup polls resource state separately
from action history. Callers choose their SDK waiter deadline; writes are never
automatically resubmitted by the SDK.

Graceful reboot, password reset, disk plug/unplug, and disk resize still need
completed checks before claiming full write compatibility.

## Cleanup and next steps

All disposable VDS from this session were removed, including the final isolated
fixtures 100440 through 100443.
The final VDS listing contained only the preexisting server 62527, which was
not modified. The post-cleanup balance was **299.09 RUB**.

Disks **207** and **208** remain for manual removal through the control panel,
as agreed. Both are 1 GB and billed at 3 RUB per month each. The public API
does not document disk deletion. Disk 207's missing UUID may require provider
assistance for cleanup too.

Ask HexCore to inspect disk 207's missing UUID, both disks' attachment/status
failures, and action records 92684 and 92687. The public plug operation's
target-selection behavior also needs clarification. Once resolved, rerun the
affected isolated checks and update these findings.

Offline validation passed with Go 1.22.12: 130 passing tests/subtests, 95.5%
statement coverage, and `go vet`. Race checks passed in the Go 1.22 Linux image.

# Branch `tgm-fallback-thirdpath`

Experimental eco-gotests branch that keeps a **third resolution path** in
`GetGmInterfaceToGPS`: when the profile has no plugins and `GetGmInterfaceFromHardwareConfig`
fails or is incomplete, fall back to `PtpProfile` fields (`ts2phc.master 1`,
`leadingInterface`, `interface`).

Use this branch only while validating whether helix103 / QE `HardwareConfig` CRs
are intentionally minimal (holdover-only) or should be brought in line with
[linuxptp-daemon `t-gm-dell-XR8720t.md`](https://github.com/openshift/linuxptp-daemon/blob/main/doc/t-gm-dell-XR8720t.md).

**`gnrd-tgm-holdover`** omits this fallback; GM GPS NIC resolution must succeed
via plugins or a complete `HardwareConfig` ClockChain.

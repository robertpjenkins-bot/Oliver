package main

import _ "embed"

// qemu-aarch64 (QEMU 8.2, static, GPL-2.0: see assets/qemu-COPYING.txt) runs
// the Pi's programs while Ollie OS is made on a PC (archbuild.sh).
//
//go:embed assets/qemu-aarch64-static.gz
var qemuAarch64 []byte

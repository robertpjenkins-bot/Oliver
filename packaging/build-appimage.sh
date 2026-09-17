#!/usr/bin/env bash
# Builds an AppImage from this repository.
#
# Two applications come out of the same binary. APP=oliver (the default) builds
# Oliver, the command line. APP=procoder builds Pro Coder, which opens the
# coder in a window instead — the same file, a different front door.
#
# The result is a single self-contained file that runs on any x86-64 Linux with
# a 2.6+ kernel: the binary is statically linked with no libc dependency, so
# there is no glibc-version floor to trip over.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "${here}/.." && pwd)"

APP="${APP:-oliver}"
case "${APP}" in
	oliver)
		app_name="Oliver"; app_slug="oliver"; app_run="${here}/AppRun" ;;
	procoder)
		app_name="ProCoder"; app_slug="procoder"; app_run="${here}/procoder.AppRun" ;;
	*) echo "unknown APP: ${APP} (use oliver or procoder)" >&2; exit 1 ;;
esac

ARCH="${ARCH:-x86_64}"
VERSION="${VERSION:-$(git -C "${root}" describe --tags --always --dirty 2>/dev/null || echo dev)}"
OUTDIR="${OUTDIR:-${root}/dist}"
APPDIR="${OUTDIR}/${app_name}.AppDir"
OUTPUT="${OUTPUT:-${OUTDIR}/${app_name}-${VERSION}-${ARCH}.AppImage}"

# Map the AppImage architecture name onto Go's.
case "${ARCH}" in
	x86_64)  goarch=amd64 ;;
	i686)    goarch=386   ;;
	aarch64) goarch=arm64 ;;
	armhf)   goarch=arm   ;;
	*) echo "unsupported architecture: ${ARCH}" >&2; exit 1 ;;
esac

say() { printf '\033[1m==>\033[0m %s\n' "$*"; }

say "building ${app_name} ${VERSION} for ${ARCH}"
rm -rf "${APPDIR}"
mkdir -p "${APPDIR}/usr/bin" \
         "${APPDIR}/usr/share/applications" \
         "${APPDIR}/usr/share/icons/hicolor/scalable/apps" \
         "${APPDIR}/usr/share/metainfo"

# CGO_ENABLED=0 is what makes the binary portable: no dynamic libc linkage, so
# it runs on distributions older and newer than the build machine alike.
CGO_ENABLED=0 GOOS=linux GOARCH="${goarch}" \
	go build -trimpath \
		-ldflags "-s -w -X main.version=${VERSION}" \
		-o "${APPDIR}/usr/bin/oliver" \
		"${root}/cmd/oliver"

install -m 0755 "${app_run}"                      "${APPDIR}/AppRun"
install -m 0644 "${here}/${app_slug}.desktop"     "${APPDIR}/${app_slug}.desktop"
install -m 0644 "${here}/${app_slug}.svg"         "${APPDIR}/${app_slug}.svg"
install -m 0644 "${here}/${app_slug}.desktop"     "${APPDIR}/usr/share/applications/${app_slug}.desktop"
install -m 0644 "${here}/${app_slug}.svg"         "${APPDIR}/usr/share/icons/hicolor/scalable/apps/${app_slug}.svg"
if [ -f "${here}/${app_slug}.appdata.xml" ]; then
	install -m 0644 "${here}/${app_slug}.appdata.xml" "${APPDIR}/usr/share/metainfo/${app_slug}.appdata.xml"
fi

# A desktop entry's Exec has to name something that resolves on PATH, and the
# binary here is called oliver. Pro Coder's entry therefore names a launcher of
# its own, which AppRun puts on PATH along with everything else in usr/bin.
if [ "${APP}" = "procoder" ]; then
	cat > "${APPDIR}/usr/bin/procoder" <<-'LAUNCHER'
	#!/bin/sh
	exec "$(dirname "$(readlink -f "${0}")")/oliver" procoder "$@"
	LAUNCHER
	chmod 0755 "${APPDIR}/usr/bin/procoder"
fi

say "AppDir ready at ${APPDIR}"

if [ "${APPDIR_ONLY:-0}" = "1" ]; then
	say "APPDIR_ONLY set; stopping before packaging"
	exit 0
fi

# appimagetool turns the AppDir into the single-file AppImage. Use a local copy
# if one is present, otherwise fetch it.
tool="${APPIMAGETOOL:-}"
if [ -z "${tool}" ]; then
	if command -v appimagetool >/dev/null 2>&1; then
		tool="$(command -v appimagetool)"
	else
		cache="${XDG_CACHE_HOME:-${HOME}/.cache}/oliver"
		tool="${cache}/appimagetool-${ARCH}.AppImage"
		if [ ! -x "${tool}" ]; then
			mkdir -p "${cache}"
			url="https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-${ARCH}.AppImage"
			say "downloading appimagetool"
			if ! curl -fsSL -o "${tool}" "${url}"; then
				cat >&2 <<-MSG

				Could not download appimagetool from:
				  ${url}

				The AppDir is built and complete at:
				  ${APPDIR}

				To finish packaging, install appimagetool and re-run this script,
				or point it at an existing copy:
				  APPIMAGETOOL=/path/to/appimagetool $0
				MSG
				exit 1
			fi
			chmod +x "${tool}"
		fi
	fi
fi

# appimagetool is itself an AppImage and needs FUSE to mount. Where FUSE is
# unavailable (containers, minimal systems) it can unpack itself instead.
run_tool() {
	if "${tool}" --version >/dev/null 2>&1; then
		"${tool}" "$@"
	else
		"${tool}" --appimage-extract-and-run "$@"
	fi
}

say "packaging"
mkdir -p "${OUTDIR}"
ARCH="${ARCH}" run_tool "${APPDIR}" "${OUTPUT}"

say "built ${OUTPUT}"
ls -lh "${OUTPUT}"

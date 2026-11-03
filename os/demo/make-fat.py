# Joins Mac programs for Intel and Apple Silicon into one ("universal")
# program, as Apple's lipo does: a fat header, then each program on a
# 16 KB boundary.
import struct, sys
out, *parts = sys.argv[1:]
CPU = {b"\x07\x00\x00\x01": (0x01000007, 3), b"\x0c\x00\x00\x01": (0x0100000C, 0)}  # x86_64, arm64
blobs = [open(p, "rb").read() for p in parts]
align = 14
offset = 1 << align
header = struct.pack(">II", 0xCAFEBABE, len(blobs))
body = b""
for b in blobs:
    cputype, sub = CPU[b[4:8]]
    header += struct.pack(">IIIII", cputype, sub, offset + len(body), len(b), align)
    body += b + b"\0" * ((-len(b)) % (1 << align))
open(out, "wb").write(header + b"\0" * (offset - len(header)) + body)

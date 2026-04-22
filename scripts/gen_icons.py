"""Generates minimal placeholder PNG icons without ImageMagick."""
import struct, zlib, os

def png(size: int, r: int, g: int, b: int) -> bytes:
    def chunk(name: bytes, data: bytes) -> bytes:
        c = zlib.crc32(name + data) & 0xFFFFFFFF
        return struct.pack(">I", len(data)) + name + data + struct.pack(">I", c)

    raw = b"".join(
        b"\x00" + bytes([r, g, b, 255] * size)
        for _ in range(size)
    )
    idat = zlib.compress(raw)
    return (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", struct.pack(">IIBBBBB", size, size, 8, 2, 0, 0, 0))
        + chunk(b"IDAT", idat)
        + chunk(b"IEND", b"")
    )


os.makedirs("src-tauri/icons", exist_ok=True)
for sz in (32, 128, 256):
    with open(f"src-tauri/icons/{sz}x{sz}.png", "wb") as f:
        f.write(png(sz, 124, 106, 255))

# Minimal stub for icns / ico (apps need real icons before shipping)
import shutil
shutil.copy("src-tauri/icons/256x256.png", "src-tauri/icons/icon.icns")
shutil.copy("src-tauri/icons/256x256.png", "src-tauri/icons/icon.ico")
print("Placeholder icons written to src-tauri/icons/")

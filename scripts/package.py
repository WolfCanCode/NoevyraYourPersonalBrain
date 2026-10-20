"""Archive a tested binary with license/docs, then checksum the archive (stdlib only)."""
import hashlib
import pathlib
import re
import sys
import tarfile
import zipfile

binary, version, goos, arch = sys.argv[1:]
if not re.fullmatch(r"[a-zA-Z0-9.+-]+", version):
    raise SystemExit("Invalid version")
root = pathlib.Path(__file__).resolve().parent.parent
out = root / "dist"
out.mkdir(exist_ok=True)
name = f"brain_{version}_{goos}_{arch}"
files = [(pathlib.Path(binary), "brain.exe" if goos == "windows" else "brain"),
         (root / "LICENSE", "LICENSE"), (root / "README.md", "README.md"),
         (root / "docs" / "RELEASING.md", "RELEASING.md")]
if goos == "windows":
    archive = out / (name + ".zip")
    with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as z:
        for source, target in files:
            z.write(source, target)
else:
    archive = out / (name + ".tar.gz")
    with tarfile.open(archive, "w:gz") as z:
        for source, target in files:
            z.add(source, arcname=target)
digest = hashlib.sha256(archive.read_bytes()).hexdigest()
(out / (archive.name + ".sha256")).write_text(f"{digest}  {archive.name}\n")
print(archive.name)

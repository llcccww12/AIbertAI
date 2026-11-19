#!/usr/bin/env python3
"""将自有宿主安装器编译为后台可分发的固定版本目录。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import struct
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--version', required=True)
parser.add_argument('--arch', choices=['amd64', 'arm64'], required=True)
parser.add_argument('--output', type=Path, default=Path('dist'))
args = parser.parse_args()
if not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}', args.version):
    parser.error('版本号无效')
root = Path(__file__).resolve().parent.parent
output = args.output.resolve() / args.version
output.mkdir(parents=True, exist_ok=True)
final = output / args.arch
if final.exists():
    parser.error('目标发行目录已存在，禁止覆盖已发布版本')
with tempfile.TemporaryDirectory(prefix='.package-', dir=output) as work:
    stage = Path(work)
    subprocess.run(['go', 'build', '-buildvcs=false', '-trimpath', '-ldflags', f'-s -w -X main.version={args.version}',
                    '-o', str(stage / 'monkeyai-engine'), './cmd/aibertai-engine'],
                   cwd=root, env={**os.environ, 'GOOS': 'linux', 'GOARCH': args.arch, 'CGO_ENABLED': '0'}, check=True)
    shutil.copyfile(stage / 'monkeyai-engine', stage / 'installer')
    digests = {}
    for name in ['monkeyai-engine', 'installer']:
        path = stage / name
        with path.open('rb') as file:
            header = file.read(20)
            if len(header) != 20 or header[:6] != b'\x7fELF\x02\x01' or struct.unpack('<H', header[18:20])[0] != {'amd64': 62, 'arm64': 183}[args.arch]:
                parser.error(f'{name} 不是目标架构的 Linux ELF 文件')
            file.seek(0)
            digests[name] = hashlib.file_digest(file, 'sha256').hexdigest()
        path.chmod(0o755)
    (stage / 'manifest.json').write_text(json.dumps({'version': args.version, 'architecture': args.arch, 'files': digests}, indent=2) + '\n')
    # 仅在所有文件构建和校验通过后发布完整目录。
    stage.rename(final)
print(f'发行目录已生成：{final}')

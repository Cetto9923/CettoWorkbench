#!/usr/bin/env python3
"""Verify native models and dynamic tables cannot inherit ownership exemptions."""
from pathlib import Path
import subprocess
import tempfile

root = Path(__file__).resolve().parents[3]
checker = (root / 'tools/check-zentao-write.sh').read_text()
fixtures = {
    'internal/model/user.go': 'package model\nfunc (User) TableName() string { return "zt_user" }\n',
    'internal/module/profile/repo.go': 'package profile\nfunc save() {\n db.Model(&model.User{}).Updates(fields)\n}\n',
    'internal/module/agileteam/repo_basic_atomic.go': 'package agileteam\nfunc edit() {\n table := r.teamgroupTable()\n db.Table(table).Updates(fields)\n}\n',
    'internal/module/sample/repo.go': 'package sample\nfunc save() {\n db.Table("zt_wb_unregistered").Create(row)\n}\n',
}
with tempfile.TemporaryDirectory(prefix='wb-write-gate-') as directory:
    base = Path(directory)
    for name, source in {**fixtures, 'tools/check-zentao-write.sh': checker,
                         'tools/zentao-write-allowlist.txt': '', 'db/install.sql': ''}.items():
        path = base / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(source)
    result = subprocess.run(['bash', str(base / 'tools/check-zentao-write.sh')], capture_output=True, text=True)
    assert result.returncode == 1, result.stdout
    for table in ['zt_user', 'zt_teamgroup', 'zt_wb_unregistered']:
        assert '表 ' + table in result.stdout, result.stdout
print('native write ownership regression passed')

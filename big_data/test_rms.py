# test_rms.py
import pytest
from rms import date_range, run_command, delete_local_files
import datetime


def test_date_range():
    """测试日期范围生成"""
    start_date = '20260111'
    end_date = '20260113'
    expected = ['20260111', '20260112', '20260113']
    assert date_range(start_date, end_date) == expected


def test_run_command():
    """测试命令执行"""
    # 测试成功执行
    success, output = run_command('echo "test"')
    assert success == True
    assert 'test' in output

    # 测试失败执行
    success, output = run_command('invalid_command_12345')
    assert success == False


def test_delete_local_files(tmp_path):
    """测试文件删除"""
    # 创建测试文件
    test_file = tmp_path / 'HGU-test.csv'
    test_file.write_text('test content')

    # 切换到临时目录并测试
    import os
    original_cwd = os.getcwd()
    os.chdir(tmp_path)

    try:
        delete_local_files()
        # 验证文件被删除
        assert not test_file.exists()
    finally:
        os.chdir(original_cwd)
#!/usr/bin/env python3
"""
代码质量报告脚本
生成项目的代码质量报告，包括代码检查、类型检查和测试覆盖率
"""

import json
import subprocess
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Any


def run_command_capture(cmd: List[str]) -> tuple[int, str, str]:
    """运行命令并捕获输出"""
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=Path(__file__).parent.parent,
        )
        return result.returncode, result.stdout, result.stderr
    except FileNotFoundError:
        return 1, "", f"Command not found: {' '.join(cmd)}"


def analyze_ruff_output(stdout: str, stderr: str) -> Dict[str, Any]:
    """分析Ruff输出"""
    lines = stdout.strip().split('\n') if stdout.strip() else []
    error_lines = [line for line in lines if line.strip() and not line.startswith('warning:')]
    
    # 统计错误类型
    error_types = {}
    for line in error_lines:
        if ':' in line and ' ' in line:
            parts = line.split(':')
            if len(parts) >= 4:
                error_code = parts[3].strip().split()[0]
                error_types[error_code] = error_types.get(error_code, 0) + 1
    
    return {
        'total_errors': len(error_lines),
        'error_types': error_types,
        'fixable': 'fixable' in stdout.lower(),
        'details': error_lines[:10]  # 只显示前10个错误
    }


def analyze_mypy_output(stdout: str, stderr: str) -> Dict[str, Any]:
    """分析MyPy输出"""
    lines = stdout.strip().split('\n') if stdout.strip() else []
    error_lines = [line for line in lines if 'error:' in line]
    
    # 统计错误类型
    error_types = {}
    for line in error_lines:
        if '[' in line and ']' in line:
            error_type = line.split('[')[-1].split(']')[0]
            error_types[error_type] = error_types.get(error_type, 0) + 1
    
    return {
        'total_errors': len(error_lines),
        'error_types': error_types,
        'details': error_lines[:10]  # 只显示前10个错误
    }


def get_file_stats() -> Dict[str, Any]:
    """获取文件统计信息"""
    project_root = Path(__file__).parent.parent
    
    # 统计Python文件
    py_files = list(project_root.rglob('*.py'))
    py_files = [f for f in py_files if not any(part.startswith('.') for part in f.parts)]
    
    # 统计代码行数
    total_lines = 0
    code_lines = 0
    comment_lines = 0
    blank_lines = 0
    
    for py_file in py_files:
        try:
            with open(py_file, 'r', encoding='utf-8') as f:
                lines = f.readlines()
                total_lines += len(lines)
                
                for line in lines:
                    stripped = line.strip()
                    if not stripped:
                        blank_lines += 1
                    elif stripped.startswith('#'):
                        comment_lines += 1
                    else:
                        code_lines += 1
        except Exception:
            continue
    
    return {
        'total_files': len(py_files),
        'total_lines': total_lines,
        'code_lines': code_lines,
        'comment_lines': comment_lines,
        'blank_lines': blank_lines,
        'comment_ratio': round(comment_lines / total_lines * 100, 2) if total_lines > 0 else 0
    }


def generate_quality_report() -> Dict[str, Any]:
    """生成代码质量报告"""
    print("🔍 正在生成代码质量报告...")
    
    report = {
        'timestamp': datetime.now().isoformat(),
        'project': 'quantitative-system',
        'file_stats': get_file_stats()
    }
    
    # 运行Ruff检查
    print("  📋 运行Ruff代码检查...")
    ruff_code, ruff_out, ruff_err = run_command_capture(['uv', 'run', 'ruff', 'check', '.'])
    report['ruff'] = {
        'exit_code': ruff_code,
        'analysis': analyze_ruff_output(ruff_out, ruff_err)
    }
    
    # 运行MyPy类型检查
    print("  🔍 运行MyPy类型检查...")
    mypy_code, mypy_out, mypy_err = run_command_capture(['uv', 'run', 'mypy', '.'])
    report['mypy'] = {
        'exit_code': mypy_code,
        'analysis': analyze_mypy_output(mypy_out, mypy_err)
    }
    
    # 运行测试（如果有的话）
    print("  🧪 运行测试...")
    test_code, test_out, test_err = run_command_capture(['uv', 'run', 'pytest', '--tb=no', '-q'])
    report['tests'] = {
        'exit_code': test_code,
        'output': test_out.strip(),
        'error': test_err.strip()
    }
    
    return report


def print_report(report: Dict[str, Any]) -> None:
    """打印格式化的报告"""
    print("\n" + "="*80)
    print(f"📊 代码质量报告 - {report['project']}")
    print(f"⏰ 生成时间: {report['timestamp']}")
    print("="*80)
    
    # 文件统计
    stats = report['file_stats']
    print(f"\n📁 文件统计:")
    print(f"  • Python文件数量: {stats['total_files']}")
    print(f"  • 总行数: {stats['total_lines']:,}")
    print(f"  • 代码行数: {stats['code_lines']:,}")
    print(f"  • 注释行数: {stats['comment_lines']:,}")
    print(f"  • 空白行数: {stats['blank_lines']:,}")
    print(f"  • 注释率: {stats['comment_ratio']}%")
    
    # Ruff报告
    ruff = report['ruff']
    ruff_analysis = ruff['analysis']
    print(f"\n🔧 Ruff代码检查:")
    if ruff['exit_code'] == 0:
        print("  ✅ 没有发现代码质量问题")
    else:
        print(f"  ❌ 发现 {ruff_analysis['total_errors']} 个问题")
        if ruff_analysis['error_types']:
            print("  📋 错误类型分布:")
            for error_type, count in sorted(ruff_analysis['error_types'].items()):
                print(f"    • {error_type}: {count}")
        if ruff_analysis['fixable']:
            print("  🔧 部分问题可以自动修复")
    
    # MyPy报告
    mypy = report['mypy']
    mypy_analysis = mypy['analysis']
    print(f"\n🔍 MyPy类型检查:")
    if mypy['exit_code'] == 0:
        print("  ✅ 没有发现类型错误")
    else:
        print(f"  ❌ 发现 {mypy_analysis['total_errors']} 个类型错误")
        if mypy_analysis['error_types']:
            print("  📋 错误类型分布:")
            for error_type, count in sorted(mypy_analysis['error_types'].items()):
                print(f"    • {error_type}: {count}")
    
    # 测试报告
    tests = report['tests']
    print(f"\n🧪 测试结果:")
    if tests['exit_code'] == 0:
        print("  ✅ 所有测试通过")
        if tests['output']:
            print(f"  📊 {tests['output']}")
    else:
        print("  ❌ 测试失败或未找到测试")
        if tests['error']:
            print(f"  📝 {tests['error']}")
    
    # 总体评估
    print(f"\n📈 总体评估:")
    total_issues = ruff_analysis['total_errors'] + mypy_analysis['total_errors']
    if total_issues == 0 and tests['exit_code'] == 0:
        print("  🎉 代码质量优秀！")
    elif total_issues < 50:
        print("  👍 代码质量良好，建议修复剩余问题")
    elif total_issues < 200:
        print("  ⚠️  代码质量一般，需要重点改进")
    else:
        print("  🚨 代码质量较差，需要大量改进")
    
    print(f"\n🔧 改进建议:")
    if ruff_analysis['total_errors'] > 0:
        print("  • 运行 'make format' 自动格式化代码")
        print("  • 运行 'uv run ruff check --fix .' 自动修复部分问题")
    if mypy_analysis['total_errors'] > 0:
        print("  • 为函数和变量添加类型注解")
        print("  • 修复类型不匹配的问题")
    if tests['exit_code'] != 0:
        print("  • 编写和完善测试用例")
        print("  • 确保测试覆盖率达到80%以上")
    
    print("\n" + "="*80)


def save_report(report: Dict[str, Any]) -> None:
    """保存报告到文件"""
    report_file = Path(__file__).parent.parent / 'quality_report.json'
    with open(report_file, 'w', encoding='utf-8') as f:
        json.dump(report, f, indent=2, ensure_ascii=False)
    print(f"\n💾 详细报告已保存到: {report_file}")


def main():
    """主函数"""
    try:
        report = generate_quality_report()
        print_report(report)
        save_report(report)
    except KeyboardInterrupt:
        print("\n\n⚠️  报告生成被用户中断")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ 生成报告时发生错误: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
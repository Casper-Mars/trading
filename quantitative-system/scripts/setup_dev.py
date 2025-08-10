#!/usr/bin/env python3
"""
开发环境设置脚本
用于快速设置量化系统的开发环境
"""

import shutil
import subprocess
import sys
from pathlib import Path


def run_command(cmd: list[str], description: str) -> bool:
    """运行命令并处理错误"""
    print(f"\n🔄 {description}...")
    try:
        # 检查命令是否存在并获取完整路径
        cmd_path = shutil.which(cmd[0])
        if not cmd_path:
            print(f"❌ 命令 '{cmd[0]}' 不可用")
            return False

        # 使用完整路径构建命令
        full_cmd = [cmd_path, *cmd[1:]]
        result = subprocess.run(
            full_cmd,
            check=True,
            capture_output=True,
            text=True,
            cwd=Path(__file__).parent.parent,
            timeout=300,  # 添加5分钟超时
        )
        print(f"✅ {description} 完成")
        if result.stdout:
            print(f"输出: {result.stdout.strip()}")
        return True
    except subprocess.CalledProcessError as e:
        print(f"❌ {description} 失败")
        print(f"错误: {e.stderr.strip()}")
        return False
    except FileNotFoundError:
        print(f"❌ 命令未找到: {' '.join(cmd)}")
        return False


def check_uv_installed() -> bool:
    """检查uv是否已安装"""
    return shutil.which("uv") is not None


def install_uv() -> bool:
    """安装uv包管理器"""
    print("\n📦 检测到uv未安装, 正在安装...")
    try:
        # 检查curl是否可用
        if not shutil.which("curl"):
            print("❌ curl 命令不可用,请手动安装 uv")
            return False

        # 使用官方安装脚本
        curl_path = shutil.which("curl")
        result = subprocess.run(
            [curl_path, "-LsSf", "https://astral.sh/uv/install.sh"],
            capture_output=True,
            text=True,
            timeout=30,  # 添加超时
        )
        if result.returncode == 0:
            # 检查sh是否可用
            sh_path = shutil.which("sh")
            if not sh_path:
                print("❌ sh 命令不可用")
                return False
            subprocess.run([sh_path], input=result.stdout, text=True, check=True, timeout=60)
            print("✅ uv 安装完成")
            return True
        else:
            print("❌ 下载uv安装脚本失败")
            return False
    except Exception as e:
        print(f"❌ 安装uv失败: {e}")
        print("请手动安装uv: https://docs.astral.sh/uv/getting-started/installation/")
        return False


def setup_development_environment() -> bool:
    """设置开发环境"""
    print("🚀 开始设置量化系统开发环境")

    # 检查uv是否安装
    if not check_uv_installed() and not install_uv():
        return False

    success = True

    # 安装开发依赖
    success &= run_command(["uv", "sync", "--extra", "dev"], "安装项目依赖和开发工具")

    # 安装pre-commit hooks
    success &= run_command(["uv", "add", "--dev", "pre-commit"], "安装pre-commit")

    success &= run_command(
        ["uv", "run", "pre-commit", "install"], "设置pre-commit hooks"
    )

    # 运行初始代码检查
    print("\n🔍 运行初始代码质量检查...")

    # 格式化代码
    run_command(["uv", "run", "ruff", "format", "."], "格式化代码")

    # 修复可自动修复的问题
    run_command(["uv", "run", "ruff", "check", "--fix", "."], "修复代码问题")

    # 运行类型检查(可能会有错误, 但不影响设置)
    run_command(["uv", "run", "mypy", "."], "运行类型检查")

    return success


def print_usage_instructions():
    """打印使用说明"""
    print("\n" + "=" * 60)
    print("🎉 开发环境设置完成!")
    print("=" * 60)
    print("\n📋 可用的开发命令:")
    print("  make help        - 查看所有可用命令")
    print("  make lint        - 运行代码检查")
    print("  make format      - 格式化代码")
    print("  make type-check  - 运行类型检查")
    print("  make test        - 运行测试")
    print("  make test-cov    - 运行测试并生成覆盖率报告")
    print("  make all-checks  - 运行所有检查")
    print("  make clean       - 清理缓存文件")

    print("\n🔧 开发工具说明:")
    print("  • Ruff: 快速的Python代码检查和格式化工具")
    print("  • MyPy: 静态类型检查器")
    print("  • Pytest: 测试框架")
    print("  • Pre-commit: Git提交前自动检查")

    print("\n📝 代码质量标准:")
    print("  • 所有函数必须有类型注解")
    print("  • 代码必须通过Ruff检查")
    print("  • 代码必须通过MyPy类型检查")
    print("  • 测试覆盖率应保持在80%以上")

    print("\n🚀 开始开发:")
    print("  1. 编写代码时确保添加类型注解")
    print("  2. 提交前运行 'make all-checks' 确保代码质量")
    print("  3. 编写测试用例覆盖新功能")
    print("  4. 使用 'make test-cov' 检查测试覆盖率")
    print("\n" + "=" * 60)


def main():
    """主函数"""
    try:
        if setup_development_environment():
            print_usage_instructions()
            sys.exit(0)
        else:
            print("\n❌ 开发环境设置失败, 请检查错误信息")
            sys.exit(1)
    except KeyboardInterrupt:
        print("\n\n⚠️  设置被用户中断")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ 设置过程中发生未知错误: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()

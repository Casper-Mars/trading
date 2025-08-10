# Quantitative System Makefile
# 提供便捷的开发工具命令

.PHONY: help install dev-install lint format type-check test test-cov clean all-checks

# 默认目标
help:
	@echo "Available commands:"
	@echo "  install      - 安装项目依赖"
	@echo "  dev-install  - 安装开发依赖"
	@echo "  lint         - 运行代码检查 (ruff)"
	@echo "  format       - 格式化代码 (ruff format)"
	@echo "  type-check   - 运行类型检查 (mypy)"
	@echo "  test         - 运行测试"
	@echo "  test-cov     - 运行测试并生成覆盖率报告"
	@echo "  all-checks   - 运行所有检查 (lint + type-check + test)"
	@echo "  clean        - 清理缓存文件"

# 安装项目依赖
install:
	uv sync

# 安装开发依赖
dev-install:
	uv sync --extra dev

# 代码检查
lint:
	@echo "Running ruff linter..."
	uv run ruff check .

# 代码格式化
format:
	@echo "Formatting code with ruff..."
	uv run ruff format .
	@echo "Fixing import order..."
	uv run ruff check --fix --select I .

# 类型检查
type-check:
	@echo "Running mypy type checker..."
	uv run mypy .

# 运行测试
test:
	@echo "Running tests..."
	uv run pytest

# 运行测试并生成覆盖率报告
test-cov:
	@echo "Running tests with coverage..."
	uv run pytest --cov=quantitative_system --cov-report=html --cov-report=term

# 运行所有检查
all-checks: lint type-check test
	@echo "All checks completed!"

# 清理缓存文件
clean:
	@echo "Cleaning cache files..."
	find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name ".pytest_cache" -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name ".mypy_cache" -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name ".ruff_cache" -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name "htmlcov" -exec rm -rf {} + 2>/dev/null || true
	find . -name "*.pyc" -delete 2>/dev/null || true
	find . -name ".coverage" -delete 2>/dev/null || true
	@echo "Cache files cleaned!"

# 开发环境设置
setup-dev: dev-install
	@echo "Development environment setup completed!"
	@echo "You can now run:"
	@echo "  make lint      - 检查代码质量"
	@echo "  make format    - 格式化代码"
	@echo "  make type-check - 类型检查"
	@echo "  make test      - 运行测试"
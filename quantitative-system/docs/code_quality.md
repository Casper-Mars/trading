# 代码质量保证指南

本文档介绍量化系统项目的代码质量保证工具和最佳实践。

## 🛠️ 工具链概览

我们使用以下工具来确保代码质量：

- **Ruff**: 快速的Python代码检查和格式化工具
- **MyPy**: 静态类型检查器
- **Pytest**: 测试框架
- **Pre-commit**: Git提交前自动检查
- **UV**: 现代Python包管理器

## 🚀 快速开始

### 1. 安装开发依赖

```bash
# 安装所有开发依赖
uv sync --extra dev

# 或者使用Makefile
make dev-install
```

### 2. 设置开发环境

```bash
# 运行自动化设置脚本
python scripts/setup_dev.py

# 或者手动设置pre-commit
uv add --dev pre-commit
uv run pre-commit install
```

### 3. 运行代码质量检查

```bash
# 查看所有可用命令
make help

# 运行所有检查
make all-checks

# 单独运行各项检查
make lint        # 代码检查
make format      # 代码格式化
make type-check  # 类型检查
make test        # 运行测试
```

## 📋 详细工具说明

### Ruff - 代码检查和格式化

Ruff是一个极快的Python代码检查器，集成了多个工具的功能：

```bash
# 检查代码问题
uv run ruff check .

# 自动修复可修复的问题
uv run ruff check --fix .

# 格式化代码
uv run ruff format .

# 检查特定规则
uv run ruff check --select E,W .
```

#### 启用的规则集

- **E, W**: pycodestyle错误和警告
- **F**: pyflakes错误
- **I**: isort导入排序
- **B**: flake8-bugbear常见错误
- **C4**: flake8-comprehensions列表推导式
- **UP**: pyupgrade现代化语法
- **N**: pep8-naming命名规范
- **S**: flake8-bandit安全检查
- **T20**: flake8-print打印语句检查
- **SIM**: flake8-simplify代码简化
- **RUF**: Ruff特定规则

#### 忽略的规则

- **E501**: 行长度限制（使用格式化工具处理）
- **B008**: 参数默认值中的函数调用
- **S101**: assert语句使用
- **T201**: print语句检查
- **RUF001/002/003**: Unicode字符检查（支持中文注释和文档字符串）

### MyPy - 类型检查

MyPy进行静态类型检查，确保类型安全：

```bash
# 运行类型检查
uv run mypy .

# 检查特定文件
uv run mypy services/

# 显示详细信息
uv run mypy --show-error-codes .
```

#### 配置要点

- 要求所有函数有类型注解
- 启用严格模式检查
- 支持Pydantic和SQLAlchemy插件
- 忽略第三方库的类型问题

### Pytest - 测试框架

```bash
# 运行所有测试
uv run pytest

# 运行特定测试
uv run pytest tests/test_services/

# 生成覆盖率报告
uv run pytest --cov=quantitative_system --cov-report=html

# 运行特定标记的测试
uv run pytest -m "not slow"
```

#### 测试标记

- `unit`: 单元测试
- `integration`: 集成测试
- `slow`: 慢速测试

### Pre-commit - 提交前检查

Pre-commit在Git提交前自动运行检查：

```bash
# 安装pre-commit hooks
uv run pre-commit install

# 手动运行所有hooks
uv run pre-commit run --all-files

# 跳过pre-commit检查（不推荐）
git commit --no-verify
```

## 📊 代码质量报告

使用我们的质量报告脚本获取项目整体质量状况：

```bash
# 生成代码质量报告
python scripts/quality_report.py
```

报告包含：
- 文件统计信息
- Ruff检查结果
- MyPy类型检查结果
- 测试运行结果
- 改进建议

## 🎯 代码质量标准

### 必须遵守的规则

1. **类型注解**: 所有公共函数必须有完整的类型注解
2. **代码格式**: 必须通过Ruff格式化检查
3. **代码质量**: 必须通过Ruff代码检查
4. **类型安全**: 必须通过MyPy类型检查
5. **测试覆盖**: 新功能必须有对应的测试用例

### 推荐的实践

1. **文档字符串**: 为复杂函数添加详细的docstring
2. **错误处理**: 适当的异常处理和错误信息
3. **代码复用**: 避免重复代码，提取公共函数
4. **命名规范**: 使用清晰、有意义的变量和函数名
5. **注释**: 为复杂逻辑添加解释性注释

## 🔧 常见问题解决

### Ruff问题

```bash
# 查看具体错误信息
uv run ruff check --show-source .

# 自动修复导入排序
uv run ruff check --fix --select I .

# 忽略特定规则（在代码中）
# ruff: noqa: E501
```

### MyPy问题

```bash
# 查看详细错误信息
uv run mypy --show-error-codes --show-column-numbers .

# 忽略特定错误（在代码中）
# type: ignore[error-code]

# 为第三方库添加类型存根
uv add --dev types-requests
```

### 测试问题

```bash
# 运行失败的测试并显示详细信息
uv run pytest -v --tb=long

# 只运行失败的测试
uv run pytest --lf

# 调试模式运行测试
uv run pytest -s --pdb
```

## 🚀 CI/CD集成

项目配置了GitHub Actions工作流，在每次推送和PR时自动运行：

- 代码质量检查（Ruff）
- 类型检查（MyPy）
- 测试运行（Pytest）
- 安全检查（Bandit）
- 依赖检查（Safety）

查看 `.github/workflows/ci.yml` 了解详细配置。

## 📝 开发工作流

### 日常开发

1. 编写代码时确保添加类型注解
2. 定期运行 `make format` 格式化代码
3. 提交前运行 `make all-checks` 确保质量
4. 编写测试用例覆盖新功能

### 提交代码

1. Pre-commit会自动运行检查
2. 修复所有检查失败的问题
3. 确保测试通过
4. 提交代码

### 代码审查

1. 检查类型注解是否完整
2. 验证测试覆盖率
3. 确认代码符合项目规范
4. 运行质量报告检查整体状况

## 🔗 相关资源

- [Ruff文档](https://docs.astral.sh/ruff/)
- [MyPy文档](https://mypy.readthedocs.io/)
- [Pytest文档](https://docs.pytest.org/)
- [Pre-commit文档](https://pre-commit.com/)
- [UV文档](https://docs.astral.sh/uv/)

## 💡 提示和技巧

### 编辑器集成

推荐在编辑器中安装以下插件：
- Ruff插件（实时代码检查）
- MyPy插件（实时类型检查）
- Python插件（语法高亮和智能提示）

### 性能优化

- 使用 `--cache-dir` 选项加速重复检查
- 配置编辑器实时检查避免批量修复
- 使用 `make clean` 清理缓存文件

### 团队协作

- 统一使用相同的工具版本
- 定期更新依赖和工具
- 分享最佳实践和经验
- 持续改进代码质量标准
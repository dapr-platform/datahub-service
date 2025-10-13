#!/bin/bash

# DataHub Service 测试运行脚本
# 用于运行单元测试并生成覆盖率报告

set -e

# 脚本配置
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE_DIR="$PROJECT_ROOT/coverage"
COVERAGE_FILE="$COVERAGE_DIR/coverage.out"
COVERAGE_HTML="$COVERAGE_DIR/coverage.html"
TEST_RESULTS="$COVERAGE_DIR/test_results.json"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# 打印标题
print_title() {
    print_message $BLUE "=================================="
    print_message $BLUE "$1"
    print_message $BLUE "=================================="
}

# 创建覆盖率目录
create_coverage_dir() {
    if [ ! -d "$COVERAGE_DIR" ]; then
        mkdir -p "$COVERAGE_DIR"
        print_message $GREEN "✅ 创建覆盖率目录: $COVERAGE_DIR"
    fi
}

# 清理旧的覆盖率文件
cleanup_old_coverage() {
    if [ -f "$COVERAGE_FILE" ]; then
        rm "$COVERAGE_FILE"
        print_message $YELLOW "🧹 清理旧的覆盖率文件"
    fi
    
    if [ -f "$COVERAGE_HTML" ]; then
        rm "$COVERAGE_HTML"
        print_message $YELLOW "🧹 清理旧的HTML报告"
    fi
}

# 运行单元测试
run_tests() {
    print_title "运行单元测试"
    
    cd "$PROJECT_ROOT"
    
    # 运行测试并生成覆盖率
    print_message $BLUE "🧪 运行测试套件..."
    
    # 设置测试环境变量
    export GO_ENV=test
    export DATABASE_URL=":memory:"
    
    # 运行测试
    if go test -v -race -coverprofile="$COVERAGE_FILE" -covermode=atomic ./...; then
        print_message $GREEN "✅ 所有测试通过"
    else
        print_message $RED "❌ 测试失败"
        exit 1
    fi
}

# 运行特定模块的测试
run_module_tests() {
    local module=$1
    print_title "运行模块测试: $module"
    
    cd "$PROJECT_ROOT"
    
    if go test -v -race -coverprofile="$COVERAGE_DIR/${module}_coverage.out" "./${module}"; then
        print_message $GREEN "✅ 模块 $module 测试通过"
    else
        print_message $RED "❌ 模块 $module 测试失败"
        exit 1
    fi
}

# 生成覆盖率报告
generate_coverage_report() {
    print_title "生成覆盖率报告"
    
    if [ ! -f "$COVERAGE_FILE" ]; then
        print_message $RED "❌ 覆盖率文件不存在: $COVERAGE_FILE"
        return 1
    fi
    
    # 生成HTML报告
    print_message $BLUE "📊 生成HTML覆盖率报告..."
    go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML"
    
    # 显示覆盖率统计
    print_message $BLUE "📈 覆盖率统计:"
    go tool cover -func="$COVERAGE_FILE" | tail -1
    
    # 生成详细的覆盖率信息
    print_message $BLUE "📋 详细覆盖率信息:"
    go tool cover -func="$COVERAGE_FILE" | head -20
    
    print_message $GREEN "✅ HTML报告已生成: $COVERAGE_HTML"
}

# 运行基准测试
run_benchmarks() {
    print_title "运行基准测试"
    
    cd "$PROJECT_ROOT"
    
    print_message $BLUE "⚡ 运行基准测试..."
    go test -bench=. -benchmem -run=^$ ./... | tee "$COVERAGE_DIR/benchmark_results.txt"
    
    print_message $GREEN "✅ 基准测试完成"
}

# 运行竞态条件检测
run_race_tests() {
    print_title "运行竞态条件检测"
    
    cd "$PROJECT_ROOT"
    
    print_message $BLUE "🏃 检测竞态条件..."
    if go test -race ./...; then
        print_message $GREEN "✅ 未发现竞态条件"
    else
        print_message $RED "❌ 发现竞态条件"
        exit 1
    fi
}

# 检查测试覆盖率阈值
check_coverage_threshold() {
    local threshold=${1:-80}
    
    print_title "检查覆盖率阈值"
    
    if [ ! -f "$COVERAGE_FILE" ]; then
        print_message $RED "❌ 覆盖率文件不存在"
        return 1
    fi
    
    # 提取总覆盖率
    local coverage=$(go tool cover -func="$COVERAGE_FILE" | grep "total:" | awk '{print $3}' | sed 's/%//')
    
    print_message $BLUE "📊 当前覆盖率: ${coverage}%"
    print_message $BLUE "🎯 目标阈值: ${threshold}%"
    
    # 比较覆盖率
    if [ $(echo "$coverage >= $threshold" | bc -l) -eq 1 ]; then
        print_message $GREEN "✅ 覆盖率达到阈值要求"
        return 0
    else
        print_message $RED "❌ 覆盖率未达到阈值要求"
        return 1
    fi
}

# 生成测试报告
generate_test_report() {
    print_title "生成测试报告"
    
    local report_file="$COVERAGE_DIR/test_report.md"
    
    cat > "$report_file" << EOF
# DataHub Service 测试报告

## 测试执行时间
- 执行时间: $(date)
- Go版本: $(go version)

## 测试覆盖率
$(go tool cover -func="$COVERAGE_FILE" | tail -1)

## 测试统计
\`\`\`
$(go test ./... -json 2>/dev/null | jq -s '
  {
    "total_tests": length,
    "passed_tests": map(select(.Action == "pass")) | length,
    "failed_tests": map(select(.Action == "fail")) | length,
    "skipped_tests": map(select(.Action == "skip")) | length
  }' 2>/dev/null || echo "统计信息生成失败")
\`\`\`

## 基准测试结果
$(if [ -f "$COVERAGE_DIR/benchmark_results.txt" ]; then cat "$COVERAGE_DIR/benchmark_results.txt"; else echo "未运行基准测试"; fi)

## 文件链接
- [HTML覆盖率报告]($COVERAGE_HTML)
- [覆盖率数据文件]($COVERAGE_FILE)
EOF

    print_message $GREEN "✅ 测试报告已生成: $report_file"
}

# 清理测试环境
cleanup_test_env() {
    print_message $YELLOW "🧹 清理测试环境..."
    
    # 清理临时文件
    find "$PROJECT_ROOT" -name "*.test" -delete 2>/dev/null || true
    find "$PROJECT_ROOT" -name "test.db" -delete 2>/dev/null || true
    
    print_message $GREEN "✅ 测试环境清理完成"
}

# 显示帮助信息
show_help() {
    cat << EOF
DataHub Service 测试运行脚本

用法:
    $0 [选项]

选项:
    -h, --help              显示帮助信息
    -a, --all              运行所有测试（默认）
    -u, --unit             仅运行单元测试
    -b, --benchmark        运行基准测试
    -r, --race             运行竞态条件检测
    -c, --coverage         生成覆盖率报告
    -t, --threshold <num>  设置覆盖率阈值（默认80）
    -m, --module <name>    运行特定模块的测试
    --clean               清理覆盖率文件
    --report              生成测试报告

示例:
    $0                     # 运行所有测试
    $0 -u                  # 仅运行单元测试
    $0 -b                  # 运行基准测试
    $0 -t 85               # 设置85%的覆盖率阈值
    $0 -m service/basic_library  # 运行特定模块测试
    $0 --clean             # 清理覆盖率文件
EOF
}

# 主函数
main() {
    local run_all=true
    local run_unit=false
    local run_benchmark=false
    local run_race=false
    local generate_coverage=false
    local generate_report=false
    local clean_coverage=false
    local coverage_threshold=80
    local test_module=""
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -a|--all)
                run_all=true
                shift
                ;;
            -u|--unit)
                run_all=false
                run_unit=true
                shift
                ;;
            -b|--benchmark)
                run_all=false
                run_benchmark=true
                shift
                ;;
            -r|--race)
                run_all=false
                run_race=true
                shift
                ;;
            -c|--coverage)
                generate_coverage=true
                shift
                ;;
            -t|--threshold)
                coverage_threshold="$2"
                shift 2
                ;;
            -m|--module)
                test_module="$2"
                run_all=false
                shift 2
                ;;
            --clean)
                clean_coverage=true
                shift
                ;;
            --report)
                generate_report=true
                shift
                ;;
            *)
                print_message $RED "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 创建必要目录
    create_coverage_dir
    
    # 清理覆盖率文件
    if [ "$clean_coverage" = true ]; then
        cleanup_old_coverage
        print_message $GREEN "✅ 覆盖率文件已清理"
        exit 0
    fi
    
    # 清理旧文件
    cleanup_old_coverage
    
    # 根据参数执行相应操作
    if [ "$run_all" = true ]; then
        run_tests
        generate_coverage_report
        check_coverage_threshold $coverage_threshold
        run_benchmarks
        run_race_tests
    elif [ "$run_unit" = true ]; then
        run_tests
    elif [ "$run_benchmark" = true ]; then
        run_benchmarks
    elif [ "$run_race" = true ]; then
        run_race_tests
    elif [ -n "$test_module" ]; then
        run_module_tests "$test_module"
    fi
    
    # 生成覆盖率报告
    if [ "$generate_coverage" = true ] && [ -f "$COVERAGE_FILE" ]; then
        generate_coverage_report
    fi
    
    # 生成测试报告
    if [ "$generate_report" = true ]; then
        generate_test_report
    fi
    
    # 清理测试环境
    cleanup_test_env
    
    print_message $GREEN "🎉 测试执行完成！"
}

# 执行主函数
main "$@"

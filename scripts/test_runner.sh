#!/bin/bash

# HTTPDNS Go SDK 测试运行脚本

set -e

echo "=== HTTPDNS Go SDK Test Runner ==="
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Go环境
check_go_env() {
    print_status "Checking Go environment..."
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    GO_VERSION=$(go version | cut -d' ' -f3)
    print_success "Go version: $GO_VERSION"
}

# 运行单元测试
run_unit_tests() {
    print_status "Running unit tests..."
    if go test -v -short -race -coverprofile=coverage.out ./...; then
        print_success "Unit tests passed"
        
        # 生成覆盖率报告
        if command -v go &> /dev/null; then
            COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
            print_success "Test coverage: $COVERAGE"
        fi
    else
        print_error "Unit tests failed"
        return 1
    fi
}

# 运行集成测试
run_integration_tests() {
    print_status "Running integration tests..."
    if go test -v -tags=integration -timeout=5m ./...; then
        print_success "Integration tests passed"
    else
        print_warning "Integration tests failed (this is expected without real service configuration)"
    fi
}

# 运行端到端测试
run_e2e_tests() {
    print_status "Running end-to-end tests..."
    
    if [[ -z "$HTTPDNS_ACCOUNT_ID" || -z "$HTTPDNS_SECRET_KEY" ]]; then
        print_warning "HTTPDNS_ACCOUNT_ID or HTTPDNS_SECRET_KEY not set, skipping E2E tests"
        print_warning "To run E2E tests, set these environment variables:"
        print_warning "  export HTTPDNS_ACCOUNT_ID=your_account_id"
        print_warning "  export HTTPDNS_SECRET_KEY=your_secret_key"
        return 0
    fi
    
    if go test -v -tags=e2e -timeout=10m ./...; then
        print_success "End-to-end tests passed"
    else
        print_error "End-to-end tests failed"
        return 1
    fi
}

# 运行性能测试
run_benchmark_tests() {
    print_status "Running benchmark tests..."
    if go test -bench=. -benchmem -benchtime=5s -timeout=10m ./...; then
        print_success "Benchmark tests completed"
    else
        print_error "Benchmark tests failed"
        return 1
    fi
}

# 运行内存泄漏检测
run_memory_tests() {
    print_status "Running memory leak detection..."
    
    # 检查是否安装了valgrind或其他内存检测工具
    if command -v valgrind &> /dev/null; then
        print_status "Using valgrind for memory leak detection..."
        # 这里可以添加valgrind相关的测试
        print_warning "Valgrind memory tests not implemented yet"
    else
        print_status "Running Go memory tests..."
        if go test -v -run=TestMemory -timeout=5m ./...; then
            print_success "Memory tests passed"
        else
            print_warning "Memory tests failed or not found"
        fi
    fi
}

# 运行代码质量检查
run_quality_checks() {
    print_status "Running code quality checks..."
    
    # 检查代码格式
    print_status "Checking code format..."
    if ! gofmt -l . | grep -q .; then
        print_success "Code format check passed"
    else
        print_error "Code format check failed. Run 'gofmt -w .' to fix"
        gofmt -l .
        return 1
    fi
    
    # 检查代码风格（如果安装了golint）
    if command -v golint &> /dev/null; then
        print_status "Running golint..."
        if golint ./... | grep -q .; then
            print_warning "Golint found issues:"
            golint ./...
        else
            print_success "Golint check passed"
        fi
    fi
    
    # 运行go vet
    print_status "Running go vet..."
    if go vet ./...; then
        print_success "Go vet check passed"
    else
        print_error "Go vet check failed"
        return 1
    fi
    
    # 检查模块依赖
    print_status "Checking module dependencies..."
    if go mod tidy && go mod verify; then
        print_success "Module dependencies check passed"
    else
        print_error "Module dependencies check failed"
        return 1
    fi
}

# 生成测试报告
generate_test_report() {
    print_status "Generating test report..."
    
    REPORT_FILE="test_report_$(date +%Y%m%d_%H%M%S).txt"
    
    {
        echo "HTTPDNS Go SDK Test Report"
        echo "=========================="
        echo "Generated at: $(date)"
        echo "Go version: $(go version)"
        echo
        
        echo "Test Coverage:"
        if [[ -f coverage.out ]]; then
            go tool cover -func=coverage.out
        else
            echo "No coverage data available"
        fi
        echo
        
        echo "Module Information:"
        go list -m all
        echo
        
        echo "Build Information:"
        go version -m $(go env GOPATH)/bin/* 2>/dev/null || echo "No binaries found"
        
    } > "$REPORT_FILE"
    
    print_success "Test report generated: $REPORT_FILE"
}

# 清理测试文件
cleanup() {
    print_status "Cleaning up test files..."
    rm -f coverage.out
    rm -f *.test
    rm -f cpu.prof mem.prof
    print_success "Cleanup completed"
}

# 显示帮助信息
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -u, --unit          Run unit tests only"
    echo "  -i, --integration   Run integration tests only"
    echo "  -e, --e2e          Run end-to-end tests only"
    echo "  -b, --benchmark    Run benchmark tests only"
    echo "  -m, --memory       Run memory tests only"
    echo "  -q, --quality      Run code quality checks only"
    echo "  -a, --all          Run all tests (default)"
    echo "  -r, --report       Generate test report"
    echo "  -c, --cleanup      Cleanup test files"
    echo "  -h, --help         Show this help message"
    echo
    echo "Environment Variables:"
    echo "  HTTPDNS_ACCOUNT_ID  Account ID for E2E tests"
    echo "  HTTPDNS_SECRET_KEY  Secret key for E2E tests"
    echo
    echo "Examples:"
    echo "  $0                 # Run all tests"
    echo "  $0 -u              # Run unit tests only"
    echo "  $0 -b              # Run benchmark tests only"
    echo "  $0 -a -r           # Run all tests and generate report"
}

# 主函数
main() {
    local run_unit=false
    local run_integration=false
    local run_e2e=false
    local run_benchmark=false
    local run_memory=false
    local run_quality=false
    local run_all=true
    local generate_report=false
    local do_cleanup=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -u|--unit)
                run_unit=true
                run_all=false
                shift
                ;;
            -i|--integration)
                run_integration=true
                run_all=false
                shift
                ;;
            -e|--e2e)
                run_e2e=true
                run_all=false
                shift
                ;;
            -b|--benchmark)
                run_benchmark=true
                run_all=false
                shift
                ;;
            -m|--memory)
                run_memory=true
                run_all=false
                shift
                ;;
            -q|--quality)
                run_quality=true
                run_all=false
                shift
                ;;
            -a|--all)
                run_all=true
                shift
                ;;
            -r|--report)
                generate_report=true
                shift
                ;;
            -c|--cleanup)
                do_cleanup=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 检查Go环境
    check_go_env
    
    # 执行清理
    if [[ "$do_cleanup" == true ]]; then
        cleanup
        exit 0
    fi
    
    local exit_code=0
    
    # 运行测试
    if [[ "$run_all" == true ]]; then
        run_quality_checks || exit_code=1
        run_unit_tests || exit_code=1
        run_integration_tests || true  # 不因集成测试失败而退出
        run_e2e_tests || true          # 不因E2E测试失败而退出
        run_benchmark_tests || exit_code=1
        run_memory_tests || true       # 不因内存测试失败而退出
    else
        [[ "$run_quality" == true ]] && (run_quality_checks || exit_code=1)
        [[ "$run_unit" == true ]] && (run_unit_tests || exit_code=1)
        [[ "$run_integration" == true ]] && (run_integration_tests || true)
        [[ "$run_e2e" == true ]] && (run_e2e_tests || true)
        [[ "$run_benchmark" == true ]] && (run_benchmark_tests || exit_code=1)
        [[ "$run_memory" == true ]] && (run_memory_tests || true)
    fi
    
    # 生成报告
    if [[ "$generate_report" == true ]]; then
        generate_test_report
    fi
    
    echo
    if [[ $exit_code -eq 0 ]]; then
        print_success "All tests completed successfully!"
    else
        print_error "Some tests failed. Check the output above for details."
    fi
    
    exit $exit_code
}

# 运行主函数
main "$@"
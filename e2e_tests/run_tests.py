#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
@module e2e_tests/run_tests
@description 端到端测试运行器，支持多种测试模式和报告生成
@architecture 测试运行器 - 统一管理测试执行和报告
@documentReference basic_library_controller.go
@stateFlow 参数解析 -> 环境检查 -> 测试执行 -> 报告生成 -> 清理
@rules 支持多种测试模式，生成详细的测试报告
@dependencies unittest, argparse, sys, os, logging, json
@refs config/test_config.py, basic-library/
"""

import sys
import os
import unittest
import argparse
import logging
import json
import time
from datetime import datetime
from typing import List, Dict, Any, Optional
from io import StringIO

# 添加项目路径
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from config.test_config import config

logger = logging.getLogger(__name__)

class TestResult:
    """测试结果类"""
    
    def __init__(self):
        self.start_time = None
        self.end_time = None
        self.total_tests = 0
        self.passed_tests = 0
        self.failed_tests = 0
        self.error_tests = 0
        self.skipped_tests = 0
        self.failures = []
        self.errors = []
        self.test_details = []
    
    @property
    def success_rate(self) -> float:
        """成功率"""
        if self.total_tests == 0:
            return 0.0
        return (self.passed_tests / self.total_tests) * 100
    
    @property
    def duration(self) -> float:
        """测试持续时间（秒）"""
        if self.start_time and self.end_time:
            return (self.end_time - self.start_time).total_seconds()
        return 0.0
    
    def to_dict(self) -> Dict[str, Any]:
        """转换为字典格式"""
        return {
            "start_time": self.start_time.isoformat() if self.start_time else None,
            "end_time": self.end_time.isoformat() if self.end_time else None,
            "duration": self.duration,
            "total_tests": self.total_tests,
            "passed_tests": self.passed_tests,
            "failed_tests": self.failed_tests,
            "error_tests": self.error_tests,
            "skipped_tests": self.skipped_tests,
            "success_rate": self.success_rate,
            "failures": self.failures,
            "errors": self.errors,
            "test_details": self.test_details
        }

class TestRunner:
    """测试运行器"""
    
    def __init__(self, verbosity: int = 2):
        self.verbosity = verbosity
        self.result = TestResult()
    
    def check_service_availability(self) -> bool:
        """检查服务可用性"""
        logger.info("检查服务可用性...")
        
        try:
            # 导入通用API客户端进行健康检查
            from utils.api_client import APIClient
            api_client = APIClient(
                base_url=config.api.base_url,
                api_prefix="",  # 用于健康检查，不需要API前缀
                timeout=10
            )
            
            # 尝试访问健康检查端点
            response = api_client.get("health")
            if response.is_success:
                logger.info("服务可用")
                return True
            else:
                logger.warning(f"服务健康检查失败: {response.status_code}")
                return True  # 即使健康检查失败，也继续测试
                
        except Exception as e:
            logger.warning(f"服务健康检查异常: {e}")
            return True  # 继续测试，让具体的测试用例来判断服务状态
    
    def discover_tests(self, test_pattern: str = "test*.py", 
                      start_dir: str = None) -> unittest.TestSuite:
        """发现测试用例"""
        if start_dir is None:
            start_dir = os.path.join(os.path.dirname(__file__), "basic-library")
        
        loader = unittest.TestLoader()
        suite = loader.discover(start_dir, pattern=test_pattern)
        return suite
    
    def discover_iot_tests(self) -> unittest.TestSuite:
        """发现物联网系统专用测试用例"""
        loader = unittest.TestLoader()
        start_dir = os.path.join(os.path.dirname(__file__), "basic-library")
        suite = loader.discover(start_dir, pattern="test_iot_*.py")
        return suite
    
    def run_tests(self, test_suite: unittest.TestSuite) -> TestResult:
        """运行测试套件"""
        logger.info("开始执行测试...")
        
        # 记录开始时间
        self.result.start_time = datetime.now()
        
        # 创建测试运行器
        stream = StringIO()
        runner = unittest.TextTestRunner(
            stream=stream,
            verbosity=self.verbosity,
            buffer=True
        )
        
        # 运行测试
        test_result = runner.run(test_suite)
        
        # 记录结束时间
        self.result.end_time = datetime.now()
        
        # 收集测试结果
        self.result.total_tests = test_result.testsRun
        self.result.failed_tests = len(test_result.failures)
        self.result.error_tests = len(test_result.errors)
        self.result.skipped_tests = len(test_result.skipped)
        self.result.passed_tests = (self.result.total_tests - 
                                   self.result.failed_tests - 
                                   self.result.error_tests - 
                                   self.result.skipped_tests)
        
        # 收集失败和错误信息
        for test, traceback in test_result.failures:
            self.result.failures.append({
                "test": str(test),
                "traceback": traceback
            })
        
        for test, traceback in test_result.errors:
            self.result.errors.append({
                "test": str(test),
                "traceback": traceback
            })
        
        # 输出测试结果到控制台
        print(stream.getvalue())
        
        return self.result
    
    def generate_report(self, output_file: str = None):
        """生成测试报告"""
        if output_file is None:
            output_file = f"test_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        
        report_data = {
            "test_info": {
                "framework": "Python unittest",
                "test_environment": {
                    "base_url": config.api.base_url,
                    "timeout": config.api.timeout
                },
                "generated_at": datetime.now().isoformat()
            },
            "test_results": self.result.to_dict()
        }
        
        with open(output_file, 'w', encoding='utf-8') as f:
            json.dump(report_data, f, indent=2, ensure_ascii=False)
        
        logger.info(f"测试报告已生成: {output_file}")
        
        # 输出摘要到控制台
        self.print_summary()
    
    def print_summary(self):
        """打印测试摘要"""
        print("\n" + "="*60)
        print("测试执行摘要")
        print("="*60)
        print(f"总测试数: {self.result.total_tests}")
        print(f"通过: {self.result.passed_tests}")
        print(f"失败: {self.result.failed_tests}")
        print(f"错误: {self.result.error_tests}")
        print(f"跳过: {self.result.skipped_tests}")
        print(f"成功率: {self.result.success_rate:.1f}%")
        print(f"执行时间: {self.result.duration:.2f}秒")
        print("="*60)
        
        if self.result.failed_tests > 0:
            print(f"\n失败的测试 ({self.result.failed_tests}):")
            for i, failure in enumerate(self.result.failures, 1):
                print(f"{i}. {failure['test']}")
        
        if self.result.error_tests > 0:
            print(f"\n错误的测试 ({self.result.error_tests}):")
            for i, error in enumerate(self.result.errors, 1):
                print(f"{i}. {error['test']}")

def main():
    """主函数"""
    parser = argparse.ArgumentParser(description="基础库端到端测试运行器")
    
    parser.add_argument(
        "--pattern", "-p",
        default="test*.py",
        help="测试文件匹配模式 (默认: test*.py)"
    )
    
    parser.add_argument(
        "--module", "-m",
        help="指定测试模块 (例如: test_basic_library_workflow)"
    )
    
    parser.add_argument(
        "--test-dir", "-d",
        default=None,
        help="测试目录路径"
    )
    
    parser.add_argument(
        "--verbosity", "-v",
        type=int,
        default=2,
        choices=[0, 1, 2],
        help="输出详细程度 (0=最少, 1=正常, 2=详细)"
    )
    
    parser.add_argument(
        "--output", "-o",
        help="测试报告输出文件"
    )
    
    parser.add_argument(
        "--skip-service-check",
        action="store_true",
        help="跳过服务可用性检查"
    )
    
    parser.add_argument(
        "--config-file", "-c",
        help="指定配置文件路径"
    )
    
    parser.add_argument(
        "--iot-test",
        action="store_true",
        help="运行物联网系统专用测试"
    )
    
    parser.add_argument(
        "--test-type", "-t",
        choices=["all", "workflow", "individual", "iot"],
        default="all",
        help="指定测试类型 (all=所有测试, workflow=工作流测试, individual=单个API测试, iot=物联网专用测试)"
    )
    
    parser.add_argument(
        "--skip-cleanup",
        action="store_true",
        help="跳过测试数据清理"
    )
    
    parser.add_argument(
        "--force-cleanup",
        action="store_true",
        help="强制清理所有测试数据"
    )
    
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="干运行模式，不执行实际操作"
    )
    
    args = parser.parse_args()
    
    # 如果指定了配置文件，重新加载配置
    if args.config_file:
        from config.test_config import TestConfig
        global config
        config = TestConfig(args.config_file)
    
    # 创建测试运行器
    runner = TestRunner(verbosity=args.verbosity)
    
    # 检查服务可用性
    if not args.skip_service_check:
        if not runner.check_service_availability():
            logger.error("服务不可用，测试终止")
            sys.exit(1)
    
    try:
        # 发现测试用例
        if args.module:
            # 运行指定模块
            loader = unittest.TestLoader()
            if args.test_dir:
                sys.path.append(args.test_dir)
            suite = loader.loadTestsFromName(args.module)
        elif args.iot_test or args.test_type == "iot":
            # 运行物联网专用测试
            suite = runner.discover_iot_tests()
        elif args.test_type == "workflow":
            # 运行工作流测试
            suite = runner.discover_tests("test_*workflow*.py", args.test_dir)
        elif args.test_type == "individual":
            # 运行单个API测试
            suite = runner.discover_tests("test_individual*.py", args.test_dir)
        else:
            # 发现所有测试
            suite = runner.discover_tests(args.pattern, args.test_dir)
        
        # 运行测试
        result = runner.run_tests(suite)
        
        # 生成报告
        runner.generate_report(args.output)
        
        # 根据测试结果设置退出码
        if result.failed_tests > 0 or result.error_tests > 0:
            sys.exit(1)
        else:
            sys.exit(0)
            
    except Exception as e:
        logger.error(f"测试执行异常: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()

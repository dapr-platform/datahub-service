# -*- coding: utf-8 -*-
"""
@module e2e_tests/utils/test_helpers
@description 测试辅助工具类，提供通用的测试工具函数
@architecture 工具类模式 - 提供测试过程中的通用功能
@documentReference basic_library_controller.go
@stateFlow 工具调用 -> 执行操作 -> 返回结果
@rules 提供可复用的测试工具函数，支持数据生成、验证、清理等
@dependencies uuid, random, string, datetime, json
@refs api_client.py, test_config.py
"""

import json
import uuid
import random
import string
import logging
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional, Union
from .api_client import APIResponse

logger = logging.getLogger(__name__)

class TestDataGenerator:
    """测试数据生成器"""
    
    @staticmethod
    def generate_id() -> str:
        """生成UUID"""
        return str(uuid.uuid4())
    
    @staticmethod
    def generate_random_string(length: int = 8, prefix: str = "") -> str:
        """生成随机字符串"""
        chars = string.ascii_lowercase + string.digits
        random_str = ''.join(random.choices(chars, k=length))
        return f"{prefix}{random_str}" if prefix else random_str
    
    @staticmethod
    def generate_timestamp() -> str:
        """生成时间戳"""
        return datetime.now().isoformat()
    
    @staticmethod
    def generate_basic_library_data(name_suffix: str = None) -> Dict[str, Any]:
        """生成基础库测试数据"""
        suffix = name_suffix or TestDataGenerator.generate_random_string(6)
        return {
            "id": TestDataGenerator.generate_id(),
            "name_zh": f"测试基础库_{suffix}",
            "name_en": f"test_library_{suffix}",
            "description": f"端到端测试基础库 - {suffix}",
            "status": "active",
            "category": "test",
            "created_at": TestDataGenerator.generate_timestamp()
        }
    
    @staticmethod
    def generate_datasource_data(basic_library_id: str, name_suffix: str = None) -> Dict[str, Any]:
        """生成数据源测试数据"""
        suffix = name_suffix or TestDataGenerator.generate_random_string(6)
        return {
            "id": TestDataGenerator.generate_id(),
            "basic_library_id": basic_library_id,
            "name": f"测试数据源_{suffix}",
            "type": "http_with_auth",
            "category": "api",
            "connection_config": {
                "base_url": "https://httpbin.org",
                "auth_type": "bearer",
                "api_key": f"test-key-{suffix}"
            },
            "params_config": {
                "timeout": 30,
                "retry_count": 3
            },
            "script_enabled": False,
            "script": "",
            "status": "active",
            "created_at": TestDataGenerator.generate_timestamp()
        }
    
    @staticmethod
    def generate_interface_data(datasource_id: str, name_suffix: str = None) -> Dict[str, Any]:
        """生成数据接口测试数据"""
        suffix = name_suffix or TestDataGenerator.generate_random_string(6)
        return {
            "id": TestDataGenerator.generate_id(),
            "datasource_id": datasource_id,
            "name": f"测试接口_{suffix}",
            "description": f"端到端测试接口 - {suffix}",
            "interface_type": "api",
            "method": "GET",
            "path": "/get",
            "interface_config": {
                "query_params": {
                    "limit": 10,
                    "offset": 0
                },
                "headers": {
                    "Accept": "application/json"
                }
            },
            "status": "active",
            "created_at": TestDataGenerator.generate_timestamp()
        }
    
    @staticmethod
    def generate_schedule_config_data(datasource_id: str) -> Dict[str, Any]:
        """生成调度配置测试数据"""
        return {
            "data_source_id": datasource_id,
            "schedule_type": "cron",
            "schedule_config": {
                "cron_expression": "0 */1 * * *",
                "timezone": "Asia/Shanghai",
                "max_retry": 3,
                "timeout": 300
            },
            "is_enabled": True,
            "created_at": TestDataGenerator.generate_timestamp()
        }

class ResponseValidator:
    """响应验证器"""
    
    @staticmethod
    def validate_response_structure(response: APIResponse, expected_fields: List[str]) -> bool:
        """验证响应结构"""
        if not response.json_data:
            logger.error("响应不是有效的JSON格式")
            return False
        
        missing_fields = []
        for field in expected_fields:
            if field not in response.json_data:
                missing_fields.append(field)
        
        if missing_fields:
            logger.error(f"响应缺少必需字段: {missing_fields}")
            return False
        
        return True
    
    @staticmethod
    def validate_success_response(response: APIResponse) -> bool:
        """验证成功响应"""
        if not response.is_success:
            logger.error(f"HTTP状态码错误: {response.status_code}")
            return False
        
        if not ResponseValidator.validate_response_structure(response, ['status', 'msg']):
            return False
        
        if not response.business_success:
            logger.error(f"业务逻辑失败: {response.business_message}")
            return False
        
        return True
    
    @staticmethod
    def validate_error_response(response: APIResponse, expected_status_codes: List[int] = None) -> bool:
        """验证错误响应"""
        expected_codes = expected_status_codes or [400, 401, 403, 404, 500]
        
        if response.status_code not in expected_codes:
            logger.error(f"错误响应状态码不符合预期: {response.status_code}, 期望: {expected_codes}")
            return False
        
        if not ResponseValidator.validate_response_structure(response, ['status', 'msg']):
            return False
        
        return True
    
    @staticmethod
    def validate_data_response(response: APIResponse, required_data_fields: List[str] = None) -> bool:
        """验证包含数据的响应"""
        if not ResponseValidator.validate_success_response(response):
            return False
        
        data = response.business_data
        if data is None:
            logger.error("响应中缺少data字段")
            return False
        
        if required_data_fields:
            if isinstance(data, dict):
                missing_fields = [field for field in required_data_fields if field not in data]
                if missing_fields:
                    logger.error(f"响应数据缺少必需字段: {missing_fields}")
                    return False
            elif isinstance(data, list) and data:
                # 验证列表中第一个元素的字段
                first_item = data[0]
                if isinstance(first_item, dict):
                    missing_fields = [field for field in required_data_fields if field not in first_item]
                    if missing_fields:
                        logger.error(f"响应数据列表项缺少必需字段: {missing_fields}")
                        return False
        
        return True

class TestCleaner:
    """测试清理器"""
    
    def __init__(self, api_client):
        self.api_client = api_client
        self.created_resources = {
            'basic_libraries': [],
            'datasources': [],
            'interfaces': [],
            'schedules': []
        }
    
    def register_basic_library(self, library_id: str, library_data: Dict[str, Any]):
        """注册基础库用于清理"""
        self.created_resources['basic_libraries'].append({
            'id': library_id,
            'data': library_data
        })
    
    def register_datasource(self, datasource_id: str, datasource_data: Dict[str, Any]):
        """注册数据源用于清理"""
        self.created_resources['datasources'].append({
            'id': datasource_id,
            'data': datasource_data
        })
    
    def register_interface(self, interface_id: str, interface_data: Dict[str, Any]):
        """注册数据接口用于清理"""
        self.created_resources['interfaces'].append({
            'id': interface_id,
            'data': interface_data
        })
    
    def cleanup_all(self):
        """清理所有创建的资源"""
        logger.info("开始清理测试资源...")
        
        # 清理顺序：接口 -> 数据源 -> 基础库
        self._cleanup_interfaces()
        self._cleanup_datasources()
        self._cleanup_basic_libraries()
        
        logger.info("测试资源清理完成")
    
    def _cleanup_interfaces(self):
        """清理数据接口"""
        for interface in self.created_resources['interfaces']:
            try:
                response = self.api_client.delete_interface(interface['data'])
                if response.business_success:
                    logger.info(f"成功删除接口: {interface['id']}")
                else:
                    logger.warning(f"删除接口失败: {interface['id']}, 错误: {response.error_info}")
            except Exception as e:
                logger.error(f"删除接口异常: {interface['id']}, 错误: {e}")
    
    def _cleanup_datasources(self):
        """清理数据源"""
        for datasource in self.created_resources['datasources']:
            try:
                response = self.api_client.delete_datasource(datasource['data'])
                if response.business_success:
                    logger.info(f"成功删除数据源: {datasource['id']}")
                else:
                    logger.warning(f"删除数据源失败: {datasource['id']}, 错误: {response.error_info}")
            except Exception as e:
                logger.error(f"删除数据源异常: {datasource['id']}, 错误: {e}")
    
    def _cleanup_basic_libraries(self):
        """清理基础库"""
        for library in self.created_resources['basic_libraries']:
            try:
                response = self.api_client.delete_basic_library(library['data'])
                if response.business_success:
                    logger.info(f"成功删除基础库: {library['id']}")
                else:
                    logger.warning(f"删除基础库失败: {library['id']}, 错误: {response.error_info}")
            except Exception as e:
                logger.error(f"删除基础库异常: {library['id']}, 错误: {e}")

def wait_for_condition(condition_func, timeout: int = 30, interval: float = 1.0) -> bool:
    """等待条件满足"""
    import time
    
    start_time = time.time()
    while time.time() - start_time < timeout:
        try:
            if condition_func():
                return True
        except Exception as e:
            logger.debug(f"条件检查异常: {e}")
        
        time.sleep(interval)
    
    return False

def compare_dict_subset(actual: Dict[str, Any], expected: Dict[str, Any], 
                       ignore_keys: List[str] = None) -> bool:
    """比较字典子集（检查expected的所有字段是否在actual中且值相等）"""
    ignore_keys = ignore_keys or []
    
    for key, expected_value in expected.items():
        if key in ignore_keys:
            continue
        
        if key not in actual:
            logger.error(f"实际结果中缺少字段: {key}")
            return False
        
        actual_value = actual[key]
        if isinstance(expected_value, dict) and isinstance(actual_value, dict):
            if not compare_dict_subset(actual_value, expected_value, ignore_keys):
                return False
        elif actual_value != expected_value:
            logger.error(f"字段值不匹配: {key}, 期望: {expected_value}, 实际: {actual_value}")
            return False
    
    return True

def log_test_step(step_name: str, details: str = ""):
    """记录测试步骤"""
    logger.info(f"=== 测试步骤: {step_name} ===")
    if details:
        logger.info(details)

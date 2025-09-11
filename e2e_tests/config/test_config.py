# -*- coding: utf-8 -*-
"""
@module e2e_tests/config/test_config
@description 端到端测试通用配置管理器
@architecture 配置模式 - 分层配置管理，支持模块化扩展
@documentReference test_config.json
@stateFlow 加载通用配置 -> 合并模块配置 -> 提供统一接口
@rules 通用配置在顶层，模块特定配置在各自目录
@dependencies json, os, logging
@refs ../basic-library/config.json
"""

import json
import os
import logging
from typing import Dict, Any, Optional
from dataclasses import dataclass

logger = logging.getLogger(__name__)

@dataclass
class APIConfig:
    """API配置类"""
    base_url: str
    api_prefix: str = ""
    timeout: int = 30
    retry_count: int = 3
    headers: Dict[str, str] = None
    
    def __post_init__(self):
        if self.headers is None:
            self.headers = {}

class TestConfig:
    """测试配置管理器"""
    
    def __init__(self, config_file: str = None):
        """初始化配置管理器"""
        self.base_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        self.config_dir = os.path.join(self.base_dir, 'config')
        
        # 加载通用配置
        if config_file:
            self.common_config = self._load_config(config_file)
        else:
            self.common_config = self._load_config(os.path.join(self.config_dir, 'test_config.json'))
        
        # 模块配置缓存
        self._module_configs = {}
        
        # 设置日志
        self._setup_logging()
    
    def _load_config(self, config_path: str) -> Dict[str, Any]:
        """加载配置文件"""
        try:
            with open(config_path, 'r', encoding='utf-8') as f:
                return json.load(f)
        except FileNotFoundError:
            logger.warning(f"配置文件不存在: {config_path}")
            return self._get_default_common_config() if 'test_config.json' in config_path else {}
        except json.JSONDecodeError as e:
            logger.error(f"配置文件格式错误: {config_path}, 错误: {e}")
            return {}
    
    def _get_default_common_config(self) -> Dict[str, Any]:
        """获取默认通用配置"""
        return {
            "api": {
                "base_url": "http://localhost:8080",
                "timeout": 30,
                "retry_count": 3,
                "headers": {
                    "Content-Type": "application/json",
                    "Accept": "application/json"
                }
            },
            "logging": {
                "level": "INFO",
                "format": "%(asctime)s - %(name)s - %(levelname)s - %(message)s",
                "file": "e2e_test.log"
            },
            "validation": {
                "required_response_fields": {
                    "basic": ["status", "msg"],
                    "success": ["status", "msg", "data"],
                    "error": ["status", "msg", "error"]
                },
                "expected_status_codes": {
                    "success": [200, 201],
                    "client_error": [400, 401, 403, 404],
                    "server_error": [500, 502, 503]
                }
            },
            "test_environment": {
                "cleanup_on_failure": True,
                "max_test_duration": 300,
                "parallel_test_limit": 5
            }
        }
    
    def load_module_config(self, module_name: str) -> Dict[str, Any]:
        """加载模块特定配置"""
        if module_name in self._module_configs:
            return self._module_configs[module_name]
        
        module_config_path = os.path.join(self.base_dir, module_name, 'config.json')
        module_config = self._load_config(module_config_path)
        
        # 缓存配置
        self._module_configs[module_name] = module_config
        return module_config
    
    def get_merged_config(self, module_name: str) -> Dict[str, Any]:
        """获取合并后的配置（通用配置 + 模块配置）"""
        module_config = self.load_module_config(module_name)
        
        # 深度合并配置
        merged_config = self._deep_merge(self.common_config.copy(), module_config)
        return merged_config
    
    def _deep_merge(self, base_dict: Dict[str, Any], update_dict: Dict[str, Any]) -> Dict[str, Any]:
        """深度合并字典"""
        for key, value in update_dict.items():
            if key in base_dict and isinstance(base_dict[key], dict) and isinstance(value, dict):
                base_dict[key] = self._deep_merge(base_dict[key], value)
            else:
                base_dict[key] = value
        return base_dict
    
    def _setup_logging(self):
        """设置日志配置"""
        log_config = self.common_config.get('logging', {})
        logging.basicConfig(
            level=getattr(logging, log_config.get('level', 'INFO')),
            format=log_config.get('format', '%(asctime)s - %(levelname)s - %(message)s'),
            handlers=[
                logging.FileHandler(log_config.get('file', 'test.log'), encoding='utf-8'),
                logging.StreamHandler()
            ]
        )
    
    @property
    def api(self) -> APIConfig:
        """通用API配置"""
        api_config = self.common_config.get('api', {})
        return APIConfig(
            base_url=api_config.get('base_url', 'http://localhost:8080'),
            api_prefix=api_config.get('api_prefix', ''),
            timeout=api_config.get('timeout', 30),
            retry_count=api_config.get('retry_count', 3),
            headers=api_config.get('headers', {})
        )
    
    @property
    def logging(self) -> Dict[str, Any]:
        """日志配置"""
        return self.common_config.get('logging', {})
    
    @property
    def validation(self) -> Dict[str, Any]:
        """验证配置"""
        return self.common_config.get('validation', {})
    
    @property
    def test_environment(self) -> Dict[str, Any]:
        """测试环境配置"""
        return self.common_config.get('test_environment', {})
    
    def get_api_config_for_module(self, module_name: str) -> APIConfig:
        """获取模块的API配置"""
        merged_config = self.get_merged_config(module_name)
        api_config = merged_config.get('api', {})
        
        return APIConfig(
            base_url=api_config.get('base_url', self.api.base_url),
            api_prefix=api_config.get('api_prefix', ''),
            timeout=api_config.get('timeout', self.api.timeout),
            retry_count=api_config.get('retry_count', self.api.retry_count),
            headers=api_config.get('headers', self.api.headers)
        )
    
    def get_test_data_for_module(self, module_name: str) -> Dict[str, Any]:
        """获取模块的测试数据"""
        merged_config = self.get_merged_config(module_name)
        return merged_config.get('test_data', {})
    
    def get_test_scenarios_for_module(self, module_name: str) -> Dict[str, Any]:
        """获取模块的测试场景配置"""
        merged_config = self.get_merged_config(module_name)
        return merged_config.get('test_scenarios', {})
    
    def get_cleanup_config_for_module(self, module_name: str) -> Dict[str, Any]:
        """获取模块的清理配置"""
        merged_config = self.get_merged_config(module_name)
        return merged_config.get('cleanup', {})
    
    def get(self, key: str, default: Any = None) -> Any:
        """获取通用配置项"""
        keys = key.split('.')
        value = self.common_config
        for k in keys:
            if isinstance(value, dict) and k in value:
                value = value[k]
            else:
                return default
        return value

# 创建全局配置实例
config = TestConfig()

# 为了向后兼容，保留一些常用的配置项
BASE_URL = config.api.base_url
TIMEOUT = config.api.timeout
RETRY_COUNT = config.api.retry_count
HEADERS = config.api.headers

# 日志配置
LOG_LEVEL = config.logging.get('level', 'INFO')
LOG_FORMAT = config.logging.get('format', '%(asctime)s - %(name)s - %(levelname)s - %(message)s')
LOG_FILE = config.logging.get('file', 'e2e_test.log')

# 验证配置
REQUIRED_RESPONSE_FIELDS = config.validation.get('required_response_fields', {})
EXPECTED_STATUS_CODES = config.validation.get('expected_status_codes', {})
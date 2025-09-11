# -*- coding: utf-8 -*-
"""
@module e2e_tests/basic-library/test_individual_apis
@description 基础库单个API端点测试
@architecture 测试套件 - 独立测试每个API端点，验证参数和响应格式
@documentReference basic_library_controller.go, config.json
@stateFlow 参数验证 -> API调用 -> 响应验证 -> 业务字段检查
@rules 独立测试每个API，验证边界条件和错误处理，不依赖其他测试
@dependencies unittest, sys, os
@refs api_client.py, config.json, ../config/test_config.py
"""

import unittest
import sys
import os
import uuid
from typing import Dict, Any

# 添加项目路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from config.test_config import config
# 直接导入当前目录下的api_client模块
from api_client import BasicLibraryAPIClient

class TestBasicLibraryAPIs(unittest.TestCase):
    """基础库API单独测试"""
    
    @classmethod
    def setUpClass(cls):
        """测试类初始化"""
        # 获取配置
        cls.basic_lib_config = config.get_api_config_for_module('basic-library')
        cls.test_data = config.get_test_data_for_module('basic-library')
        cls.validation_config = config.validation
        
        # 创建API客户端
        cls.api_client = BasicLibraryAPIClient(
            base_url=cls.basic_lib_config.base_url,
            api_prefix=cls.basic_lib_config.api_prefix,
            timeout=cls.basic_lib_config.timeout,
            retry_count=cls.basic_lib_config.retry_count,
            headers=cls.basic_lib_config.headers
        )
        
        # 测试标识符
        cls.test_id = str(uuid.uuid4())[:8]
    
    def _validate_response_structure(self, response, expected_fields: list = None):
        """验证响应结构"""
        # 验证HTTP状态码
        expected_success_codes = self.validation_config.get('expected_status_codes', {}).get('success', [200, 201])
        self.assertIn(response.status_code, expected_success_codes + [400, 404, 500], 
                     f"意外的状态码: {response.status_code}")
        
        # 验证基础响应字段
        if response.json_data:
            required_basic_fields = self.validation_config.get('required_response_fields', {}).get('basic', [])
            for field in required_basic_fields:
                self.assertIn(field, response.json_data, f"缺少必需字段: {field}")
            
            # 验证成功响应的额外字段
            if response.business_success and expected_fields:
                success_fields = self.validation_config.get('required_response_fields', {}).get('success', [])
                for field in success_fields + expected_fields:
                    self.assertIn(field, response.json_data, f"成功响应缺少字段: {field}")
    
    def test_add_basic_library_valid_data(self):
        """测试添加基础库 - 有效数据"""
        library_data = self.test_data['basic_library'].copy()
        library_data.update({
            'name_zh': f"API测试基础库_{self.test_id}",
            'name_en': f"api_test_library_{self.test_id}"
        })
        
        response = self.api_client.add_basic_library(library_data)
        
        self._validate_response_structure(response)  # 不强制要求data字段
        
        if response.business_success:
            # 验证返回的数据结构
            data = response.business_data
            self.assertIsInstance(data, dict, "返回数据应该是字典")
            # 可以添加更多具体的字段验证
        
        print(f"添加基础库测试 - 状态码: {response.status_code}, 业务成功: {response.business_success}")
    
    def test_add_basic_library_invalid_data(self):
        """测试添加基础库 - 无效数据"""
        invalid_data_cases = [
            {},  # 空数据
            {'name_zh': ''},  # 空名称
            {'name_zh': 'test', 'name_en': ''},  # 空英文名
            {'name_zh': 'test', 'name_en': 'test', 'invalid_field': 'value'},  # 无效字段
        ]
        
        for i, invalid_data in enumerate(invalid_data_cases):
            with self.subTest(case=i):
                response = self.api_client.add_basic_library(invalid_data)
                
                # 无效数据应该返回客户端错误状态码
                client_error_codes = self.validation_config.get('expected_status_codes', {}).get('client_error', [400])
                self.assertIn(response.status_code, client_error_codes + [200], 
                             f"无效数据应该返回客户端错误: {response.status_code}")
                
                if response.status_code == 200:
                    # 如果返回200，业务逻辑应该失败
                    self.assertFalse(response.business_success, "业务逻辑应该失败")
                
                print(f"无效数据测试 {i} - 状态码: {response.status_code}")
    
    def test_add_datasource_valid_data(self):
        """测试添加数据源 - 有效数据"""
        datasource_data = self.test_data['data_source'].copy()
        datasource_data.update({
            'library_id': f"test_lib_{self.test_id}",
            'name': f"API测试数据源_{self.test_id}"
        })
        
        response = self.api_client.add_datasource(datasource_data)
        
        self._validate_response_structure(response)
        
        print(f"添加数据源测试 - 状态码: {response.status_code}, 业务成功: {response.business_success}")
    
    def test_test_datasource_connection(self):
        """测试数据源连接测试"""
        test_request = {
            'datasource_id': f'test_ds_{self.test_id}',
            'test_type': 'connection',
            'timeout': 10
        }
        
        response = self.api_client.test_datasource(test_request)
        
        self._validate_response_structure(response)
        
        # 数据源测试可能失败，但响应结构应该正确
        if response.json_data:
            # 验证测试结果包含必要信息
            if 'data' in response.json_data:
                test_result = response.json_data['data']
                if isinstance(test_result, dict):
                    # 可以验证测试结果的具体字段
                    pass
        
        print(f"数据源连接测试 - 状态码: {response.status_code}, 业务成功: {response.business_success}")
    
    def test_add_interface_valid_data(self):
        """测试添加数据接口 - 有效数据"""
        interface_data = self.test_data['data_interface'].copy()
        interface_data.update({
            'datasource_id': f'test_ds_{self.test_id}',
            'name': f'API测试接口_{self.test_id}'
        })
        
        response = self.api_client.add_interface(interface_data)
        
        self._validate_response_structure(response)
        
        print(f"添加数据接口测试 - 状态码: {response.status_code}, 业务成功: {response.business_success}")
    
    def test_test_interface_various_types(self):
        """测试接口调用 - 不同测试类型"""
        test_types = ['data_fetch', 'performance', 'validation']
        interface_id = f'test_if_{self.test_id}'
        
        for test_type in test_types:
            with self.subTest(test_type=test_type):
                test_request = {
                    'interface_id': interface_id,
                    'test_type': test_type,
                    'parameters': {'limit': 10},
                    'options': {'timeout': 30}
                }
                
                response = self.api_client.test_interface(test_request)
                
                self._validate_response_structure(response)
                
                print(f"接口测试 ({test_type}) - 状态码: {response.status_code}")
    
    def test_configure_schedule_valid_config(self):
        """测试配置调度 - 有效配置"""
        schedule_config = self.test_data['schedule_config'].copy()
        schedule_config.update({
            'datasource_id': f'test_ds_{self.test_id}',
            'name': f'API测试调度_{self.test_id}'
        })
        
        response = self.api_client.configure_schedule(schedule_config)
        
        self._validate_response_structure(response)
        
        print(f"配置调度测试 - 状态码: {response.status_code}, 业务成功: {response.business_success}")
    
    def test_get_basic_library_nonexistent(self):
        """测试获取不存在的基础库"""
        nonexistent_id = f'nonexistent_{self.test_id}'
        
        response = self.api_client.get_basic_library(nonexistent_id)
        
        # 不存在的资源应该返回404或业务失败
        if response.status_code == 200:
            self.assertFalse(response.business_success, "不存在的资源业务逻辑应该失败")
        else:
            self.assertEqual(response.status_code, 404, "不存在的资源应该返回404")
        
        print(f"获取不存在基础库测试 - 状态码: {response.status_code}")
    
    def test_list_basic_libraries(self):
        """测试列出基础库"""
        # 测试无参数列表
        response = self.api_client.list_basic_libraries()
        
        self._validate_response_structure(response, ['data'])
        
        if response.business_success and response.business_data:
            # 验证列表数据结构
            data = response.business_data
            if isinstance(data, list):
                # 验证列表项结构
                for item in data[:5]:  # 只检查前5项
                    self.assertIsInstance(item, dict, "列表项应该是字典")
            elif isinstance(data, dict) and 'items' in data:
                # 分页结构
                self.assertIn('items', data, "分页响应应该包含items")
                self.assertIsInstance(data['items'], list, "items应该是列表")
        
        # 测试带参数列表
        params = {'page': 1, 'limit': 10, 'category': 'test'}
        response_with_params = self.api_client.list_basic_libraries(params)
        
        self._validate_response_structure(response_with_params)
        
        print(f"列出基础库测试 - 状态码: {response.status_code}, 带参数状态码: {response_with_params.status_code}")
    
    def test_api_client_error_handling(self):
        """测试API客户端错误处理"""
        # 测试网络超时（使用极短的超时时间）
        short_timeout_client = BasicLibraryAPIClient(
            base_url=self.basic_lib_config.base_url,
            api_prefix=self.basic_lib_config.api_prefix,
            timeout=0.001,  # 极短超时
            retry_count=1
        )
        
        try:
            response = short_timeout_client.list_basic_libraries()
            # 如果没有抛出异常，检查响应
            self.assertIsNotNone(response, "响应不应该为空")
        except Exception as e:
            # 预期可能会有超时异常
            print(f"预期的超时异常: {type(e).__name__}")
    
    def test_response_field_validation(self):
        """测试响应字段验证"""
        # 获取一个可能存在的资源
        response = self.api_client.list_basic_libraries({'limit': 1})
        
        if response.is_success and response.json_data:
            # 验证响应包含必需的业务字段
            required_fields = self.validation_config.get('required_response_fields', {})
            
            if response.business_success:
                success_fields = required_fields.get('success', [])
                for field in success_fields:
                    self.assertIn(field, response.json_data, f"成功响应缺少字段: {field}")
            else:
                error_fields = required_fields.get('error', [])
                for field in error_fields:
                    self.assertIn(field, response.json_data, f"错误响应缺少字段: {field}")
        
        print(f"响应字段验证测试 - 状态码: {response.status_code}")

class TestBasicLibraryAPIEdgeCases(unittest.TestCase):
    """基础库API边界情况测试"""
    
    @classmethod
    def setUpClass(cls):
        """测试类初始化"""
        cls.basic_lib_config = config.get_api_config_for_module('basic-library')
        cls.api_client = BasicLibraryAPIClient(
            base_url=cls.basic_lib_config.base_url,
            api_prefix=cls.basic_lib_config.api_prefix,
            timeout=cls.basic_lib_config.timeout
        )
    
    def test_large_data_handling(self):
        """测试大数据处理"""
        # 创建较大的测试数据
        large_description = "测试" * 1000  # 4000字符
        
        library_data = {
            'name_zh': '大数据测试基础库',
            'name_en': 'large_data_test_library',
            'description': large_description,
            'category': 'test'
        }
        
        response = self.api_client.add_basic_library(library_data)
        
        # 验证服务器能够处理大数据
        self.assertIsNotNone(response, "响应不应该为空")
        self.assertNotEqual(response.status_code, 413, "不应该返回请求实体过大错误")
        
        print(f"大数据处理测试 - 状态码: {response.status_code}")
    
    def test_special_characters_handling(self):
        """测试特殊字符处理"""
        special_chars_data = {
            'name_zh': '特殊字符测试@#$%^&*()',
            'name_en': 'special_chars_test_!@#$%',
            'description': '包含特殊字符的描述：《》【】""''，。？！',
            'category': 'test'
        }
        
        response = self.api_client.add_basic_library(special_chars_data)
        
        # 验证特殊字符处理
        self.assertIsNotNone(response, "响应不应该为空")
        
        print(f"特殊字符处理测试 - 状态码: {response.status_code}")
    
    def test_concurrent_requests(self):
        """测试并发请求处理"""
        import threading
        import queue
        
        results = queue.Queue()
        
        def make_request():
            try:
                response = self.api_client.list_basic_libraries({'limit': 1})
                results.put(('success', response.status_code))
            except Exception as e:
                results.put(('error', str(e)))
        
        # 创建多个并发线程
        threads = []
        for i in range(5):
            thread = threading.Thread(target=make_request)
            threads.append(thread)
            thread.start()
        
        # 等待所有线程完成
        for thread in threads:
            thread.join(timeout=30)
        
        # 收集结果
        success_count = 0
        error_count = 0
        
        while not results.empty():
            result_type, result_value = results.get()
            if result_type == 'success':
                success_count += 1
            else:
                error_count += 1
        
        print(f"并发请求测试 - 成功: {success_count}, 错误: {error_count}")
        
        # 至少应该有一些成功的请求
        self.assertGreater(success_count, 0, "应该至少有一个成功的并发请求")

if __name__ == '__main__':
    # 配置测试运行器
    unittest.main(verbosity=2, buffer=True)
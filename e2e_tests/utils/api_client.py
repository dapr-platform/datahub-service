# -*- coding: utf-8 -*-
"""
@module e2e_tests/utils/api_client
@description HTTP API客户端工具类，提供统一的API调用接口
@architecture 适配器模式 - 封装HTTP请求细节，提供统一接口
@documentReference basic_library_controller.go
@stateFlow 请求构建 -> 发送请求 -> 响应处理 -> 结果验证
@rules 统一错误处理，支持重试机制，记录请求响应日志
@dependencies requests, json, logging, time
@refs test_config.py
"""

import json
import logging
import time
from typing import Dict, Any, Optional, Union, List
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

logger = logging.getLogger(__name__)

class APIResponse:
    """API响应封装类"""
    
    def __init__(self, response: requests.Response):
        self.raw_response = response
        self.status_code = response.status_code
        self.headers = dict(response.headers)
        self.url = response.url
        self.request_time = getattr(response, 'elapsed', None)
        
        # 解析响应体
        try:
            self.json_data = response.json()
        except (json.JSONDecodeError, ValueError):
            self.json_data = None
        
        self.text = response.text
        self.is_success = 200 <= self.status_code < 300
    
    @property
    def business_success(self) -> bool:
        """检查业务逻辑是否成功"""
        if not self.json_data:
            return False
        
        # 检查业务状态码
        status = self.json_data.get('status')
        if isinstance(status, int):
            # 根据实际API响应，0表示成功
            return status == 0
        elif isinstance(status, str):
            return status.lower() in ['success', 'ok', '200']
        
        return self.is_success
    
    @property
    def business_message(self) -> str:
        """获取业务消息"""
        if self.json_data:
            return self.json_data.get('msg', self.json_data.get('message', ''))
        return ''
    
    @property
    def business_data(self) -> Any:
        """获取业务数据"""
        if self.json_data:
            return self.json_data.get('data')
        return None
    
    @property
    def error_info(self) -> str:
        """获取错误信息"""
        if self.json_data:
            error = self.json_data.get('error', self.json_data.get('msg', ''))
            if error:
                return str(error)
        return f"HTTP {self.status_code}: {self.text[:200]}"

class APIClient:
    """API客户端类"""
    
    def __init__(self, base_url: str, api_prefix: str = '', timeout: int = 30, 
                 retry_count: int = 3, headers: Optional[Dict[str, str]] = None):
        self.base_url = base_url.rstrip('/')
        self.api_prefix = api_prefix.strip('/')
        self.timeout = timeout
        self.retry_count = retry_count
        self.headers = headers or {}
        
        # 创建会话
        self.session = requests.Session()
        
        # 设置重试策略（兼容新旧版本的urllib3）
        retry_kwargs = {
            'total': retry_count,
            'backoff_factor': 1,
            'status_forcelist': [429, 500, 502, 503, 504]
        }
        
        # 尝试使用新的参数名，如果失败则使用旧的参数名
        try:
            retry_strategy = Retry(
                allowed_methods=["HEAD", "GET", "OPTIONS", "POST", "PUT", "DELETE"],
                **retry_kwargs
            )
        except TypeError:
            # 兼容旧版本urllib3
            retry_strategy = Retry(
                method_whitelist=["HEAD", "GET", "OPTIONS", "POST", "PUT", "DELETE"],
                **retry_kwargs
            )
        
        adapter = HTTPAdapter(max_retries=retry_strategy)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)
        
        # 设置默认头部
        self.session.headers.update(self.headers)
    
    def _build_url(self, endpoint: str) -> str:
        """构建完整的URL"""
        endpoint = endpoint.lstrip('/')
        if self.api_prefix:
            return f"{self.base_url}/{self.api_prefix}/{endpoint}"
        return f"{self.base_url}/{endpoint}"
    
    def _log_request(self, method: str, url: str, **kwargs):
        """记录请求日志"""
        logger.info(f"API请求: {method} {url}")
        if 'json' in kwargs:
            logger.debug(f"请求体: {json.dumps(kwargs['json'], ensure_ascii=False, indent=2)}")
        if 'params' in kwargs:
            logger.debug(f"查询参数: {kwargs['params']}")
    
    def _log_response(self, response: APIResponse):
        """记录响应日志"""
        logger.info(f"API响应: {response.status_code} ({response.request_time})")
        if response.json_data:
            logger.debug(f"响应体: {json.dumps(response.json_data, ensure_ascii=False, indent=2)}")
        elif response.text:
            logger.debug(f"响应文本: {response.text[:500]}")
    
    def request(self, method: str, endpoint: str, **kwargs) -> APIResponse:
        """通用请求方法"""
        url = self._build_url(endpoint)
        
        # 设置超时
        kwargs.setdefault('timeout', self.timeout)
        
        # 记录请求
        self._log_request(method, url, **kwargs)
        
        try:
            start_time = time.time()
            raw_response = self.session.request(method, url, **kwargs)
            elapsed = time.time() - start_time
            raw_response.elapsed = elapsed
            
            response = APIResponse(raw_response)
            self._log_response(response)
            
            return response
            
        except requests.exceptions.RequestException as e:
            logger.error(f"API请求失败: {e}")
            raise
    
    def get(self, endpoint: str, params: Optional[Dict[str, Any]] = None, **kwargs) -> APIResponse:
        """GET请求"""
        if params:
            kwargs['params'] = params
        return self.request('GET', endpoint, **kwargs)
    
    def post(self, endpoint: str, data: Optional[Dict[str, Any]] = None, **kwargs) -> APIResponse:
        """POST请求"""
        if data is not None:
            kwargs['json'] = data
        return self.request('POST', endpoint, **kwargs)
    
    def put(self, endpoint: str, data: Optional[Dict[str, Any]] = None, **kwargs) -> APIResponse:
        """PUT请求"""
        if data is not None:
            kwargs['json'] = data
        return self.request('PUT', endpoint, **kwargs)
    
    def delete(self, endpoint: str, **kwargs) -> APIResponse:
        """DELETE请求"""
        return self.request('DELETE', endpoint, **kwargs)
    
    def patch(self, endpoint: str, data: Optional[Dict[str, Any]] = None, **kwargs) -> APIResponse:
        """PATCH请求"""
        if data is not None:
            kwargs['json'] = data
        return self.request('PATCH', endpoint, **kwargs)


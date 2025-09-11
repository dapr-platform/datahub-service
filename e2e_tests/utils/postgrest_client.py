# -*- coding: utf-8 -*-
"""
@module e2e_tests/utils/postgrest_client
@description PostgREST客户端工具类，使用postgrest-py SDK访问数据库视图
@architecture 适配器模式 - 基于postgrest-py SDK，提供统一的数据库访问接口
@documentReference https://github.com/supabase/postgrest-py
@stateFlow 初始化客户端 -> 设置schema -> 执行查询 -> 处理响应
@rules 支持schema切换，统一错误处理，提供同步和异步接口
@dependencies postgrest, asyncio, typing
@refs basic_libraries_view.go
"""

import asyncio
import logging
from typing import Dict, Any, List, Optional, Union
from postgrest import AsyncPostgrestClient, SyncPostgrestClient
from postgrest.types import RequestMethod

logger = logging.getLogger(__name__)

class PostgRESTResponse:
    """PostgREST响应封装类"""
    
    def __init__(self, data: Any, count: Optional[int] = None, error: Optional[str] = None):
        self.data = data
        self.count = count
        self.error = error
        self.is_success = error is None
    
    @property
    def business_success(self) -> bool:
        """检查业务逻辑是否成功"""
        return self.is_success and self.data is not None
    
    @property
    def business_message(self) -> str:
        """获取业务消息"""
        if self.error:
            return str(self.error)
        return "操作成功"
    
    @property
    def business_data(self) -> Any:
        """获取业务数据"""
        return self.data

class PostgRESTClientWrapper:
    """PostgREST客户端封装类"""
    
    def __init__(self, base_url: str = "http://localhost:9080/api/postgrest", 
                 schema: str = "public", headers: Dict[str, str] = None):
        self.base_url = base_url
        self.schema = schema
        self.headers = headers or {}
        self._sync_client = None
        self._async_client = None
    
    @property
    def sync_client(self) -> SyncPostgrestClient:
        """获取同步客户端"""
        if self._sync_client is None:
            self._sync_client = SyncPostgrestClient(self.base_url, schema=self.schema, headers=self.headers)
        return self._sync_client
    
    @property
    def async_client(self) -> AsyncPostgrestClient:
        """获取异步客户端"""
        if self._async_client is None:
            self._async_client = AsyncPostgrestClient(self.base_url, schema=self.schema, headers=self.headers)
        return self._async_client
    
    def list_basic_libraries(self, params: Dict[str, Any] = None) -> PostgRESTResponse:
        """同步获取基础库列表"""
        try:
            query = self.sync_client.from_("basic_libraries_info").select("*")
            
            # 应用参数
            if params:
                if 'limit' in params:
                    query = query.limit(params['limit'])
                if 'offset' in params:
                    query = query.offset(params['offset'])
                if 'order' in params:
                    # PostgREST排序语法: 'column.direction' 
                    order_param = params['order']
                    if order_param == 'created_at.desc':
                        query = query.order('created_at', desc=True)
                    elif order_param == 'created_at.asc':
                        query = query.order('created_at', desc=False)
                    else:
                        query = query.order(order_param)
                # 添加支持的过滤条件（排除不存在的字段）
                supported_filters = ['status', 'category', 'name_zh', 'name_en']
                for key, value in params.items():
                    if key not in ['limit', 'offset', 'order', 'page'] and key in supported_filters:
                        query = query.eq(key, value)
            
            response = query.execute()
            return PostgRESTResponse(data=response.data, count=response.count)
            
        except Exception as e:
            logger.error(f"获取基础库列表失败: {e}")
            return PostgRESTResponse(data=[], error=str(e))
    
    async def list_basic_libraries_async(self, params: Dict[str, Any] = None) -> PostgRESTResponse:
        """异步获取基础库列表"""
        try:
            async with self.async_client as client:
                query = client.from_("basic_libraries_info").select("*")
                
                # 应用参数
                if params:
                    if 'limit' in params:
                        query = query.limit(params['limit'])
                    if 'offset' in params:
                        query = query.offset(params['offset'])
                    if 'order' in params:
                        query = query.order(params['order'])
                    # 添加更多过滤条件
                    for key, value in params.items():
                        if key not in ['limit', 'offset', 'order']:
                            query = query.eq(key, value)
                
                response = await query.execute()
                return PostgRESTResponse(data=response.data, count=response.count)
                
        except Exception as e:
            logger.error(f"异步获取基础库列表失败: {e}")
            return PostgRESTResponse(data=[], error=str(e))
    
    def get_basic_library_by_id(self, library_id: str) -> PostgRESTResponse:
        """根据ID获取基础库详情"""
        try:
            response = self.sync_client.from_("basic_libraries_info").select("*").eq("id", library_id).execute()
            data = response.data[0] if response.data else None
            return PostgRESTResponse(data=data, count=response.count)
            
        except Exception as e:
            logger.error(f"获取基础库详情失败: {e}")
            return PostgRESTResponse(data=None, error=str(e))
    
    def list_data_interfaces(self, params: Dict[str, Any] = None) -> PostgRESTResponse:
        """获取数据接口列表"""
        try:
            query = self.sync_client.from_("data_interfaces_info").select("*")
            
            # 应用参数
            if params:
                if 'limit' in params:
                    query = query.limit(params['limit'])
                if 'offset' in params:
                    query = query.offset(params['offset'])
                if 'order' in params:
                    # PostgREST排序语法处理
                    order_param = params['order']
                    if order_param == 'created_at.desc':
                        query = query.order('created_at', desc=True)
                    elif order_param == 'created_at.asc':
                        query = query.order('created_at', desc=False)
                    else:
                        query = query.order(order_param)
                if 'library_id' in params:
                    query = query.eq("library_id", params['library_id'])
                # 添加更多过滤条件
                for key, value in params.items():
                    if key not in ['limit', 'offset', 'order'] and key != 'library_id':
                        query = query.eq(key, value)
            
            response = query.execute()
            return PostgRESTResponse(data=response.data, count=response.count)
            
        except Exception as e:
            logger.error(f"获取数据接口列表失败: {e}")
            return PostgRESTResponse(data=[], error=str(e))
    
    def list_data_sources(self, params: Dict[str, Any] = None) -> PostgRESTResponse:
        """获取数据源列表"""
        try:
            query = self.sync_client.from_("data_sources_info").select("*")
            
            # 应用参数
            if params:
                if 'limit' in params:
                    query = query.limit(params['limit'])
                if 'offset' in params:
                    query = query.offset(params['offset'])
                if 'order' in params:
                    # PostgREST排序语法处理
                    order_param = params['order']
                    if order_param == 'created_at.desc':
                        query = query.order('created_at', desc=True)
                    elif order_param == 'created_at.asc':
                        query = query.order('created_at', desc=False)
                    else:
                        query = query.order(order_param)
                if 'library_id' in params:
                    query = query.eq("library_id", params['library_id'])
                # 添加更多过滤条件
                for key, value in params.items():
                    if key not in ['limit', 'offset', 'order'] and key != 'library_id':
                        query = query.eq(key, value)
            
            response = query.execute()
            return PostgRESTResponse(data=response.data, count=response.count)
            
        except Exception as e:
            logger.error(f"获取数据源列表失败: {e}")
            return PostgRESTResponse(data=[], error=str(e))
    
    def test_connection(self) -> PostgRESTResponse:
        """测试PostgREST连接"""
        try:
            # 尝试获取一条记录来测试连接
            response = self.sync_client.from_("basic_libraries_info").select("id").limit(1).execute()
            return PostgRESTResponse(data={"status": "connected", "count": response.count})
            
        except Exception as e:
            logger.error(f"PostgREST连接测试失败: {e}")
            return PostgRESTResponse(data=None, error=str(e))

# 创建全局客户端实例
postgrest_client = PostgRESTClientWrapper()

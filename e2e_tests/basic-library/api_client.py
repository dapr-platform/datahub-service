# -*- coding: utf-8 -*-
"""
@module e2e_tests/basic-library/api_client
@description 基础库API客户端，封装基础库相关的API调用
@architecture 适配器模式 - 基于通用API客户端，扩展基础库特定功能
@documentReference basic_library_controller.go
@stateFlow 继承通用API客户端 -> 扩展基础库方法 -> 提供业务级接口
@rules 只包含基础库相关的API方法，复用通用客户端的基础功能
@dependencies utils.api_client.APIClient, typing
@refs config.json, ../utils/api_client.py
"""

import sys
import os
from typing import Dict, Any

# 添加父目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from utils.api_client import APIClient, APIResponse

class BasicLibraryAPIClient(APIClient):
    """基础库API客户端"""
    
    def __init__(self, base_url: str, api_prefix: str = '/swagger/datahub-service/basic-libraries', 
                 timeout: int = 30, retry_count: int = 3, headers: Dict[str, str] = None):
        """初始化基础库API客户端"""
        super().__init__(base_url, api_prefix, timeout, retry_count, headers)
    
    # ==================== 基础库管理接口 ====================
    
    def add_basic_library(self, library_data: Dict[str, Any]) -> APIResponse:
        """添加数据基础库"""
        return self.post('add-basic-library', library_data)
    
    def delete_basic_library(self, library_data: Dict[str, Any]) -> APIResponse:
        """删除数据基础库"""
        return self.post('delete-basic-library', library_data)
    
    def update_basic_library(self, update_data: Dict[str, Any]) -> APIResponse:
        """修改数据基础库"""
        return self.post('update-basic-library', update_data)
    
    def get_basic_library(self, library_id: str) -> APIResponse:
        """获取基础库详情"""
        return self.get(f'basic-library/{library_id}')
    
    def list_basic_libraries(self, params: Dict[str, Any] = None) -> APIResponse:
        """列出基础库 - 使用原生API"""
        return self.get('', params)
    
    # ==================== 数据源管理接口 ====================
    
    def add_datasource(self, datasource_data: Dict[str, Any]) -> APIResponse:
        """添加数据源"""
        return self.post('add-datasource', datasource_data)
    
    def delete_datasource(self, datasource_data: Dict[str, Any]) -> APIResponse:
        """删除数据源"""
        return self.post('delete-datasource', datasource_data)
    
    def update_datasource(self, update_data: Dict[str, Any]) -> APIResponse:
        """更新数据源"""
        return self.post('update-datasource', update_data)
    
    def test_datasource(self, test_request: Dict[str, Any]) -> APIResponse:
        """测试数据源连接"""
        return self.post('test-datasource', test_request)
    
    def get_datasource_status(self, datasource_id: str) -> APIResponse:
        """获取数据源状态"""
        return self.get(f'datasource-status/{datasource_id}')
    
    def list_datasources(self, library_id: str = None, params: Dict[str, Any] = None) -> APIResponse:
        """列出数据源 - 使用原生API"""
        # 使用基础库控制器的数据源列表API
        from utils.api_client import APIClient
        datasource_client = APIClient(
            base_url=self.base_url,
            api_prefix='/swagger/datahub-service/basic-libraries',
            timeout=self.timeout,
            retry_count=self.retry_count,
            headers=self.headers
        )
        
        # 构建查询参数
        query_params = params or {}
        if library_id:
            query_params['library_id'] = library_id
            
        return datasource_client.get('datasources', query_params)
    
    # ==================== 数据接口管理接口 ====================
    
    def add_interface(self, interface_data: Dict[str, Any]) -> APIResponse:
        """添加数据接口"""
        return self.post('add-interface', interface_data)
    
    def delete_interface(self, interface_data: Dict[str, Any]) -> APIResponse:
        """删除数据接口"""
        return self.post('delete-interface', interface_data)
    
    def update_interface(self, update_data: Dict[str, Any]) -> APIResponse:
        """更新数据接口"""
        return self.post('update-interface', update_data)
    
    def test_interface(self, test_request: Dict[str, Any]) -> APIResponse:
        """测试接口调用"""
        return self.post('test-interface', test_request)
    
    def preview_interface_data(self, interface_id: str, limit: int = 10) -> APIResponse:
        """预览接口数据"""
        return self.get(f'interface-preview/{interface_id}', {'limit': limit})
    
    def get_interface(self, interface_id: str) -> APIResponse:
        """获取接口详情"""
        return self.get(f'interface/{interface_id}')
    
    def list_interfaces(self, datasource_id: str = None, params: Dict[str, Any] = None) -> APIResponse:
        """列出数据接口 - 使用原生API"""
        # 使用基础库控制器的接口列表API
        from utils.api_client import APIClient
        interface_client = APIClient(
            base_url=self.base_url,
            api_prefix='/swagger/datahub-service/basic-libraries',
            timeout=self.timeout,
            retry_count=self.retry_count,
            headers=self.headers
        )
        
        # 构建查询参数
        query_params = params or {}
        if datasource_id:
            query_params['data_source_id'] = datasource_id
            
        return interface_client.get('interfaces', query_params)
    
    # ==================== 调度配置接口 ====================
    
    def configure_schedule(self, schedule_config: Dict[str, Any]) -> APIResponse:
        """配置数据源调度"""
        return self.post('configure-schedule', schedule_config)
    
    def get_schedule_config(self, datasource_id: str) -> APIResponse:
        """获取调度配置"""
        return self.get(f'schedule-config/{datasource_id}')
    
    def update_schedule_config(self, schedule_config: Dict[str, Any]) -> APIResponse:
        """更新调度配置"""
        return self.post('update-schedule-config', schedule_config)
    
    def delete_schedule_config(self, datasource_id: str) -> APIResponse:
        """删除调度配置"""
        return self.delete(f'schedule-config/{datasource_id}')
    
    # ==================== 业务级便捷方法 ====================
    
    def create_complete_workflow(self, library_data: Dict[str, Any], 
                                datasource_data: Dict[str, Any], 
                                interface_data: Dict[str, Any],
                                schedule_config: Dict[str, Any] = None) -> Dict[str, APIResponse]:
        """创建完整的工作流：基础库 -> 数据源 -> 数据接口 -> 调度配置"""
        results = {}
        
        # 1. 创建基础库
        results['basic_library'] = self.add_basic_library(library_data)
        if not results['basic_library'].business_success:
            return results
        
        # 2. 创建数据源
        results['datasource'] = self.add_datasource(datasource_data)
        if not results['datasource'].business_success:
            return results
        
        # 3. 创建数据接口
        results['interface'] = self.add_interface(interface_data)
        if not results['interface'].business_success:
            return results
        
        # 4. 配置调度（可选）
        if schedule_config:
            results['schedule'] = self.configure_schedule(schedule_config)
        
        return results
    
    def cleanup_test_data(self, library_id: str = None, datasource_id: str = None, 
                         interface_id: str = None) -> Dict[str, APIResponse]:
        """清理测试数据"""
        results = {}
        
        # 清理接口
        if interface_id:
            results['delete_interface'] = self.delete_interface({'interface_id': interface_id})
        
        # 清理数据源
        if datasource_id:
            results['delete_datasource'] = self.delete_datasource({'datasource_id': datasource_id})
        
        # 清理基础库
        if library_id:
            results['delete_library'] = self.delete_basic_library({'library_id': library_id})
        
        return results
    
    def validate_workflow_health(self, library_id: str, datasource_id: str, 
                                interface_id: str) -> Dict[str, Any]:
        """验证工作流健康状态"""
        health_status = {
            'library_status': 'unknown',
            'datasource_status': 'unknown', 
            'interface_status': 'unknown',
            'overall_health': False
        }
        
        # 检查基础库状态
        library_resp = self.get_basic_library(library_id)
        if library_resp.business_success:
            health_status['library_status'] = 'healthy'
        
        # 检查数据源状态
        datasource_resp = self.get_datasource_status(datasource_id)
        if datasource_resp.business_success:
            health_status['datasource_status'] = 'healthy'
        
        # 检查接口状态
        interface_resp = self.get_interface(interface_id)
        if interface_resp.business_success:
            health_status['interface_status'] = 'healthy'
        
        # 整体健康状态
        health_status['overall_health'] = all([
            health_status['library_status'] == 'healthy',
            health_status['datasource_status'] == 'healthy', 
            health_status['interface_status'] == 'healthy'
        ])
        
        return health_status
    
    def get_basic_libraries(self, params: Dict[str, Any] = None) -> APIResponse:
        """获取基础库列表 - 兼容性方法"""
        return self.list_basic_libraries(params)
    
    def find_library_by_name_en(self, name_en: str) -> APIResponse:
        """通过name_en查找基础库"""
        return self.list_basic_libraries({"name": name_en, "size": 1})
    
    # ==================== 同步任务管理接口 ====================
    
    def create_sync_task(self, sync_task_data: Dict[str, Any]) -> APIResponse:
        """创建同步任务"""
        # 使用统一的同步任务接口
        from utils.api_client import APIClient
        sync_client = APIClient(
            base_url=self.base_url,
            api_prefix='/swagger/datahub-service/sync/tasks',
            timeout=self.timeout,
            retry_count=self.retry_count,
            headers=self.headers
        )
        return sync_client.post('', sync_task_data)
    
    def get_sync_task_list(self, params: Dict[str, Any] = None) -> APIResponse:
        """获取同步任务列表"""
        from utils.api_client import APIClient
        sync_client = APIClient(
            base_url=self.base_url,
            api_prefix='/swagger/datahub-service/sync/tasks',
            timeout=self.timeout,
            retry_count=self.retry_count,
            headers=self.headers
        )
        return sync_client.get('', params)
    
    def get_sync_task(self, task_id: str) -> APIResponse:
        """获取同步任务详情"""
        from utils.api_client import APIClient
        sync_client = APIClient(
            base_url=self.base_url,
            api_prefix='/swagger/datahub-service/sync/tasks',
            timeout=self.timeout,
            retry_count=self.retry_count,
            headers=self.headers
        )
        return sync_client.get(task_id)
    
    def start_sync_task(self, task_id: str) -> APIResponse:
        """启动同步任务"""
        from utils.api_client import APIClient
        sync_client = APIClient(
            base_url=self.base_url,
            api_prefix='/swagger/datahub-service/sync/tasks',
            timeout=self.timeout,
            retry_count=self.retry_count,
            headers=self.headers
        )
        return sync_client.post(f'{task_id}/start', {})
    
    def get_sync_task_status(self, task_id: str) -> APIResponse:
        """获取同步任务状态"""
        from utils.api_client import APIClient
        sync_client = APIClient(
            base_url=self.base_url,
            api_prefix='/swagger/datahub-service/sync/tasks',
            timeout=self.timeout,
            retry_count=self.retry_count,
            headers=self.headers
        )
        return sync_client.get(f'{task_id}/status')
    
    def get_sync_task_executions(self, task_id: str) -> APIResponse:
        """获取同步任务执行记录"""
        from utils.api_client import APIClient
        sync_client = APIClient(
            base_url=self.base_url,
            api_prefix='/swagger/datahub-service/sync/tasks',
            timeout=self.timeout,
            retry_count=self.retry_count,
            headers=self.headers
        )
        return sync_client.get(f'{task_id}/executions')
当数据库监听收到新建 数据基础库和数据主题库的通知时，需要根据数据的名称创建 schema,并且在 postgrest.schema 表中插入一条记录
例如下面的语句。名称为 test
create schema test;

insert into postgrest.schema_config values('test','test');
当收到删除通知时，同样进行删除schema。可强制删除
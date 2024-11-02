# 配置说明

用 YAML 格式的配置文件。以下是主要配置项的说明：

- 如果 include 和 exclude 同时存在，则 include 优先级高于 exclude
- include 和 exclude 支持正则表达式
- 部分 filter_by 已固定字段，无需配置

```yaml
server_port: 8080  # 服务器端口
external_url: "http://example.com"  # 外部访问地址

log:
  output: "stdout"  # 日志输出位置，stdout表示标准输出
  level: 2  # 日志级别，2表示Info级别
sources:
  - name: "main_source"  # 源名称
    url: "https://example.com/main_source.json"  # 源地址
    type: "tvbox_single"  # 源类型，tvbox_single表示单仓
    interval: 3600  # 更新间隔，单位为秒, 默认 60s, -1 表示不更新
  - name: "foo_source"
    url: "https://foo.com/main_source.json"
    type: "tvbox_single"
    disabled: true
  - name: "foo_source"
    url: "https://bar.com/main_source.json"
    type: "tvbox_single"
  - name: "multi_source"
    url: "file:///app/multi.json"  # 本地文件源
    type: "tvbox_multi"  # 多仓源
    interval: 7200
single_repo_opt: # 单仓配置
  disable: false  # 是否禁用单仓配置
  spider:
    source_name: "main_source"  # 使用main_source的spider配置
  sites:
    - disabled: false  # 是否禁用doh配置
      source_name: "main_source"  # 使用main_source的sites配置
      filter_by: "key"  # 按key进行过滤
      include: ".*"  # 包含所有站点
      exclude: "^adult_"  # 排除以adult_开头的站点
  doh: # lives/parses/flags/ijk
    - disabled: false  # 是否禁用doh配置
      source_name: "main_source"  # 使用main_source的doh配置
  fallback:
    source_name: "bar_source"  # 使用bar_source的fallback配置
multi_repo_opt:
  disable: false  # 是否禁用多仓配置
  include_single_repo: true  # 是否包含单仓配置
  repos:
  - source_name: "multi_source"  # 使用multi_source的repos配置
    field: "repos"  # 字段名
    filter_by: "name"  # 按name进行过滤
    include: ".*"  # 包含所有仓库
    exclude: "^test_"  # 排除以test_开头的仓库
epg_opt:
  disable: false  # 是否禁用EPG源
  filters:
    - source_name: "main_source"  # 使用main_source的channel_filter配置
      filter_by: "channel_id"  # 按channel_id/program_title进行过滤
m3u_opt:
  disable: false  # 是否禁用M3U源
  media_playlist_fallback:
    source_name: "main_source"  # 使用main_source的channel_filter配置
  media_playlist_filters:
    - source_name: "main_source"  # 使用main_source的channel_filter配置
      include: ".*"  # 包含所有站点, 根据名字过滤
      exclude: "^test_"  # 排除以test_开头的站点
```

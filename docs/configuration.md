# 配置说明

用 YAML 格式的配置文件，以下是主要配置规则的说明。

1. 如果 include 和 exclude 同时存在，则 include 优先级高于 exclude

2. include 和 exclude 支持正则表达式

3. 部分 filter_by 已固定字段，无需配置

4. source type 枚举：

- `tvbox_single`: TVBox 单仓配置（JSON）
- `tvbox_multi`: TVBox 多仓配置（JSON）
- `epg`: XMLTV 格式节目单（XML）
- `m3u`: 标准 M3U / M3U8 直播源列表
- `txt` 或 `diyp`: TXT / DIYP 文本直播源列表（支持双向转为 M3U）

5. M3U 与 TXT 过滤选项（`m3u_opt.media_playlist_filters`）：

- `filter_by`:
  - `name`: 按频道名称过滤（默认）
  - `group-title` / `group`: 按所属分组过滤
  - `tvg-name`: 按 `tvg-name` 属性过滤
  - `tvg-id`: 按 `tvg-id` 属性过滤

6. EPG 过滤选项（`epg_opt.filters`）：

- `filter_by`:
  - `channel_id` / `channel`: 按频道 ID 过滤
  - `display_name` / `channel_name`: 按频道显示名称过滤
  - `program_title` / `title`: 按节目名称过滤

7. 将 M3U 混合结果作为直播源自动挂载至 TVBox 单仓：

在 `tvbox_single_repo_opt.lives` 中添加内部源：

> 注：`internal/m3u` 为内置保留名称，系统会自动将混合后的 M3U/TXT 列表注入为 TVBox 直播源。

```yaml
tvbox_single_repo_opt:
  lives:
    - source_name: "internal/m3u"
```

8. 性能与缓存：

- 服务内置内存缓存与 HTTP 协商缓存（ETag / If-None-Match）。未变动的源与请求直接返回 HTTP 304，降低电视端耗电与延迟。
- 源加载器支持 Singleflight 协程并发收敛，多客户端同时唤醒请求时只进行一次上游抓取，避免源站点限流封禁。
## Log

> Log Level 参考：<https://docs.gofiber.io/api/log/>

output 设置为日志路径，如果设置成 `stdout` 则会输出到标准输出

## 配置示例

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
tvbox_single_repo_opt: # 单仓配置
  disable: false  # 是否禁用单仓配置
  spider:
    source_name: "main_source"  # 使用main_source的spider配置
  sites:
    - disabled: false  # 是否禁用doh配置
      source_name: "main_source"  # 使用main_source的sites配置
      filter_by: "key"  # 按key进行过滤
      include: ".*"  # 包含所有站点
      exclude: "^adult_"  # 排除以adult_开头的站点
  doh: # lives/parses/flags/ijk/doh/rules
    - disabled: false  # 是否禁用doh配置
      source_name: "main_source"  # 使用main_source的doh配置
  fallback:
    source_name: "bar_source"  # 使用bar_source的fallback配置
tvbox_multi_repo_opt:
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

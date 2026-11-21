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

8. 性能、缓存与防 OOM：

- 服务内置带 LRU 淘汰、TTL 过期与内存硬上限的统一缓存管理，彻底杜绝动态 EPG 查询导致的 OOM 风险。
- 核心路由（M3U、Live TXT、TVBox Repo、XMLTV）为常驻缓存（Pinned），不被动态请求驱逐。
- 支持 HTTP 协商缓存（ETag / If-None-Match 304 Not Modified），降低电视端耗电与网络延迟。
- 源加载器支持 Singleflight 协程并发收敛，多客户端同时唤醒请求时只进行一次上游抓取，避免源站点限流封禁。

9. 频道清洗与台标自动补齐（Cleaner & EPG Mapping）：

- 正则剥离多余画质/格式标记（如 `[IPv6]`、`[1080P]`、`4K`、`高清` 等）。
- 规范央视/卫视标准频道名（如 `CCTV1综合` 统一映射为 `CCTV-1`）。
- 自动根据规范频道名补齐高清台标（`tvg-logo`）与 EPG ID（`tvg-id`）。

10. 磁盘持久化快照与故障降级（Stale-While-Revalidate）：

- 上游源成功抓取后自动写入本地快照目录。
- 当上游源超时、网络波动或故障时，自动降级读取本地磁盘历史快照（Stale-on-error），杜绝电视端黑屏。

11. 同名频道多线路合并（Multi-line Aggregation）：

- 当聚合多个包含相同频道的源时，开启 `merge_duplicate_channels: true` 自动合并同名频道。
- 支持设置源优先级权重（`priority`），高质量源自动排在前面。
- M3U 默认采用 `merge_mode: "rename"` 生成清晰的线路后缀（`线路1`、`线路2`），电视端选台一目了然且自动共享台标与 EPG；TXT / DIYP 格式自动用 `$` 连接多线路。

12. 视频流防盗链代理（Stream Anti-hotlink Proxy）：

- 针对必须携带 `Referer`、`User-Agent` 等自定义防盗链请求头的视频流，提供 `/v1/stream/proxy` 透明代理接口。
- 支持自动重写（`auto_proxy: true`）或源级指定（`proxy_stream: true`）。
- 针对 M3U8 二级切片自动递归重写代理地址，TS 切片采用 SendStream 零积压流式中转。

## Log

> Log Level 参考：<https://docs.gofiber.io/api/log/>

output 设置为日志路径，如果设置成 `stdout` 则会输出到标准输出。支持 `max_size_mb`、`max_backups`、`max_age_days` 日志轮转配置。

## 配置示例

```yaml
server_port: 8080  # 服务器端口
external_url: "http://example.com"  # 外部访问地址

log:
  output: "stdout"  # 日志输出位置，stdout表示标准输出，或指定文件路径
  level: 2  # 日志级别，2表示Info级别
  max_size_mb: 50  # 单个日志文件最大 MB
  max_backups: 5   # 保留历史日志文件数
  max_age_days: 14 # 保留天数

# 统一缓存与防 OOM 配置
cache:
  disable: false          # 是否全局禁用缓存
  max_age: 60             # 客户端 Cache-Control max-age，秒
  ttl: 3600               # 内存缓存存活时间，秒
  max_entries: 1000       # 最大条目数（超过自动 LRU 淘汰动态查询）
  max_memory_mb: 64       # 内存硬上限 (MB)

# 全局网络抓取配置
network:
  timeout: 15             # 超时时间，秒
  retry: 2                # 失败重试次数
  proxy_url: ""           # 支持 http / socks5 代理
  headers:
    User-Agent: "Mozilla/5.0"

# 磁盘快照持久化兜底（防电视黑屏）
persistence:
  enable: true            # 是否开启持久化
  dir: "./data/cache"     # 快照保存目录
  stale_on_error: true    # 上游抓取失败时降级使用历史快照

# 视频流代理配置（防盗链与 CORS 转发）
stream_proxy:
  enable: true            # 是否启用流代理
  auto_proxy: true        # 配置了 headers 的源自动重写为代理地址
  timeout: 15             # 流拉取超时（秒）
  allowed_hosts: []       # 允许代理的目标主机名单（空表示不限制）

# 安全与鉴权
security:
  secret: "your_secret_key" # 刷新源密钥，支持请求头或 ?token= 查询参数

# 频道名称清洗引擎
cleaner:
  enable: true
  strip_patterns:
    - '(?i)\[?(?:IPv[46]|1080P|720P|2K|4K|8K|FHD|HD|SD|HEVC|60fps)\]?'
    - '(?i)(超清|高清|全高清|蓝光|标清|频道|台)'
  aliases:
    - canonical: "CCTV-1"
      matches: ["CCTV1", "CCTV1综合", "CCTV-1 综合", "中央一台"]
    - canonical: "湖南卫视"
      matches: ["湖南高清", "湖南卫视 HD"]

# 台标与 EPG 自动绑定
epg_mapping:
  enable: true
  logo_url_template: "https://live.fanmingming.com/tv/{name}.png"
  mappings:
    "CCTV-1": "cctv1.epg.id"

sources:
  - name: "main_source"  # 源名称
    url: "https://example.com/main_source.json"  # 源地址
    type: "tvbox_single"  # 源类型
    interval: 3600  # 更新间隔，单位为秒, 默认 60s, -1 表示不更新
    timeout: 10     # 单源覆盖超时
    priority: 50    # 优先级权重，多线路合并时越高越靠前
    headers:        # 单源特有防盗链头
      Referer: "https://example.com"
    proxy_stream: true # 强制此源的所有视频流经过本地代理转发
  - name: "live_source"
    url: "https://example.com/live.m3u"
    type: "m3u"
    interval: 1800
    priority: 100   # 优先线路
tvbox_single_repo_opt: # 单仓配置
  disable: false
  spider:
    source_name: "main_source"
  sites:
    - disabled: false
      source_name: "main_source"
      filter_by: "key"
      include: ".*"
      exclude: "^adult_"
  lives:
    - source_name: "internal/m3u" # 自动将聚合清洗后的 M3U/TXT 挂载为 TVBox 直播
  doh:
    - disabled: false
      source_name: "main_source"

tvbox_multi_repo_opt:
  disable: false
  include_single_repo: true

epg_opt:
  disable: false
  filters:
    - source_name: "main_source"
      filter_by: "channel_id"

m3u_opt:
  disable: false
  default_group: "其它频道" # 未分组频道的默认分组
  merge_duplicate_channels: true # 开启同名频道多线路合并
  merge_mode: "rename" # rename: 标注 (线路1)/(线路2); dollar: $连接
  epg_urls: # 自定义注入的 EPG URL（支持注入多个）
    - "{external_url}/v1/epg.xml"
  media_playlist_filters:
    - source_name: "live_source"
      filter_by: "name"
      include: ".*"
```

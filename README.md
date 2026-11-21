# TV MixProxy

![GitHub License](https://img.shields.io/github/license/wayjam/tv-mixproxy)
![GitHub Release](https://img.shields.io/github/v/release/wayjam/tv-mixproxy)
![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white&link=https://github.com/wayjam/tv-mixproxy/pkgs/container/tv-mixproxy)
![Vercel](https://img.shields.io/badge/Vercel-333?logo=vercel&link=https://github.com/wayjam/tv-mixproxy/blob/main/docs/vercel.md)
![GitHub Actions](https://github.com/wayjam/tv-mixproxy/actions/workflows/test.yml/badge.svg)

TV MixProxy 是一个用于混合不同 电视接口 并提供服务的工具。它支持单仓配置和多仓配置，可以轻松地整合多个来源配置。由于电视源的不稳定性，本项目希望提供一个统一的接口，当电视源变化时，无须再反复修改电视上的配置。

## 功能特点

- 支持 TVBox 单仓库和 TVBox 多仓库设置，支持在 `lives` 中通过 `source_name: "internal/m3u"` 自动引入内置 M3U 频道
- 支持代理与混合 XMLTV 标准 EPG，以及 DIYP 轻量级 JSON EPG 接口（`/api/diyp`）
- 支持代理与混合 M3U 媒体播放列表，自动规范化属性与注入多 EPG 地址（`x-tvg-url`），兼容主流 IPTV 客户端（TiviMate, Kodi, PotPlayer, APtv）
- 支持 TXT / DIYP 纯文本直播源格式双向互转与统一聚合（`/v1/live.txt`）
- **频道名称智能清洗引擎**：正则剥离 `[IPv6]`、`[1080P]`、`4K`、`高清` 等修饰标签，央视与卫视别名自动规范对齐
- **高清台标与 EPG 自动绑定**：缺失 `tvg-logo` 和 `tvg-id` 时自动根据规范频道名匹配注入
- **磁盘持久化快照与故障降级（Stale-While-Revalidate）**：上游超时或不可用时自动降级使用本地磁盘历史快照，杜绝电视黑屏
- **可配置缓存策略与防 OOM 内存保护**：支持 LRU 淘汰无界动态 DIYP 请求、保护核心 M3U/EPG 条目常驻（Pinned），严格限制内存与条目上限
- **全局与单源网络定制**：支持自定义超时、失败重试、HTTP/SOCKS5 代理及防盗链请求头
- 源请求并发合并（Singleflight），防止多客户端同时请求导致上游被封
- HTTP 协商缓存（ETag / If-None-Match 304 Not Modified）与 API 刷新密钥鉴权

### 二进制

- 执行编译 `make build`
- 执行 `./tv-mixproxy --config config.yaml`

### Docker

> 如果需要 mix 本地配置，请将配置也挂载到容器中

```bash
docker run -d --name tv-mixproxy \
-p 8080:8080 \
-v $(pwd)/config.yaml:/app/config.yaml \
ghcr.io/tv-mixproxy/tv-mixproxy:latest
```

### Vercel

请参阅 [Vercel 部署](docs/vercel.md)

## 接口说明

- `/logo`: 获取 Logo 图片（带缓存控制）
- `/wallpaper`: 获取壁纸图片（带缓存控制）
- `/v1/tvbox/spider`: 代理单仓的 spider 配置
- `/v1/tvbox/repo`: 获取混合后的单仓配置（支持 ETag 304 缓存）
- `/v1/tvbox/multi_repo`: 获取混合后的多仓配置（支持 ETag 304 缓存）
- `/v1/epg.xml`: 
    - 获取混合后的 XMLTV EPG XML 列表，支持 gzip 压缩（`?format=gz`），支持 ETag 304 缓存
- `/api/diyp` 或 `/v1/epg/diyp`:
    - DIYP 轻量级 EPG JSON 接口，支持参数：`ch`（频道名称，支持模糊匹配）、`date`（日期，格式 `YYYY-MM-DD`，默认为当天）
    - 响应格式：`{"channel_name":"CCTV1","date":"2026-11-21","epg_data":[{"title":"新闻","start":"08:00","end":"09:00"}]}`
- `/v1/m3u/media_playlist`: 获取混合后的 M3U 媒体播放列表（自动关联 `x-tvg-url`，规范化标签与属性）
- `/v1/live.txt` 或 `/live.txt`: 获取 TXT / DIYP 文本直播源格式（`分组,#genre#` 与 `频道名,URL`）
- `/refresh_source`: 触发源主动刷新与缓存清理（需提供 `CRON_SECRET` 或 `TV_MIXPROXY_SECRET` 鉴权）
## 配置说明

用 YAML 格式的配置文件。详细配置说明请参阅 [配置说明](docs/configuration.md)。

## 许可证

本项目采用 MIT 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 贡献

精力有限，测试环境有限，仅在自己的设备进行测试，欢迎提交问题和拉取请求。对于重大更改，请先开启一个 ISSUE 讨论想要更改的内容。

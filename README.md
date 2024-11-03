# TV MixProxy

TV MixProxy 是一个用于混合不同 电视接口 并提供服务的工具。它支持单仓配置和多仓配置，可以轻松地整合多个来源配置。

## 功能特点

- 支持TVBox单仓库和TvBox多仓库设置
- 支持代理 EPG
- 支持代理 M3U 媒体播放列表
- 可自定义不同配置字段的混合选项
- 定期更新源配置

## 部署

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

## 接口说明

- `/logo`: 获取 Logo 图片
- `/wallpaper`: 获取壁纸图片
- `/v1/tvbox/spider`: 代理单仓的 spider 配置
- `/v1/tvbox/repo`: 获取混合后的单仓配置
- `/v1/tvbox/multi_repo`: 获取混合后的多仓配置
- `/v1/epg.xml`: 
    - 获取混合后的EPG XML 列表, 支持 gzip 压缩
    - 默认返回 xml 格式, 可以通过 `format=gz` 获取 gzip 压缩的 xml 文件
- `/v1/m3u/media_playlist`: 获取混合后的 m3u 媒体播放列表

## 配置说明

用 YAML 格式的配置文件。详细配置说明请参阅 [配置说明](docs/configuration.md)。

## 许可证

本项目采用 MIT 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 贡献

精力有限，测试环境有限，仅在自己的设备进行测试，欢迎提交问题和拉取请求。对于重大更改，请先开启一个 ISSUE 讨论想要更改的内容。

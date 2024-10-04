# TVBox MixProxy

TVBox MixProxy 是一个用于混合不同 TVBox 配置并提供服务的工具。它支持单仓配置和多仓配置，可以轻松地整合多个来源的 TVBox 配置。

## 功能特点

- 支持单仓库和多仓库设置
- 可自定义不同配置字段的混合选项
- 定期更新源配置

## 部署

### Docker

> 如果需要 mix 本地配置，请将配置也挂载到容器中

```bash
docker run -d --name tvbox-mixproxy \
-p 8080:8080 \
-v $(pwd)/config.yaml:/app/config.yaml \
ghcr.io/tvbox-mixproxy/tvbox-mixproxy:latest
```

## 接口说明

TVBox MixProxy 提供以下 API 接口：

1. `/logo`: 获取 Logo 图片
2. `/wallpaper`: 获取壁纸图片
3. `/v1/repo`: 获取混合后的单仓配置
4. `/v1/multi_repo`: 获取混合后的多仓配置

## 配置说明

TVBox MixProxy 使用 YAML 格式的配置文件。以下是主要配置项的说明：

```yaml
server_port: 8080  # 服务器端口
external_url: "http://example.com"  # 外部访问地址

log:
  output: "stdout"  # 日志输出位置，stdout表示标准输出
  level: 2  # 日志级别，2表示Info级别

sources:
  - name: "main_source"  # 源名称
    url: "https://example.com/main_source.json"  # 源地址
    type: "single"  # 源类型，single表示单仓
    interval: 3600  # 更新间隔，单位为秒
  - name: "multi_source"
    url: "file:///app/multi.json"  # 本地文件源
    type: "multi"  # 多仓源
    interval: 7200

single_repo_opt:
  disable: false  # 是否禁用单仓配置
  spider:
    source_name: "main_source"  # 使用main_source的spider配置
  wallpaper:
    source_name: "main_source"  # 使用main_source的wallpaper配置
  logo:
    source_name: "main_source"  # 使用main_source的logo配置
  sites:
    source_name: "main_source"  # 使用main_source的sites配置
    filter_by: "key"  # 按key进行过滤
    include: ".*"  # 包含所有站点
    exclude: "^adult_"  # 排除以adult_开头的站点
  doh:
    source_name: "main_source"  # 使用main_source的doh配置
  lives:
    source_name: "main_source"  # 使用main_source的lives配置

multi_repo_opt:
  disable: false  # 是否禁用多仓配置
  include_single_repo: true  # 是否包含单仓配置
  repos:
  - source_name: "multi_source"  # 使用multi_source的repos配置
    field: "repos"  # 字段名
    filter_by: "name"  # 按name进行过滤
    include: ".*"  # 包含所有仓库
    exclude: "^test_"  # 排除以test_开头的仓库
```

## 许可证

本项目采用 MIT 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 贡献

欢迎提交问题和拉取请求。对于重大更改，请先开启一个问题讨论您想要更改的内容。

# AIbertAI 管理平台

本目录包含 Go 后端、Vite 管理后台与独立远程执行宿主。本机开发由 PostgreSQL、RustFS、Go HTTP 服务和 Vite 分别运行。员工端和浏览器管理台都通过 Vite 的统一入口访问后端。迁到内网服务器时，以目标机器的服务和地址完成部署配置与验收。

## 本机开发

现有开发入口为 [http://127.0.0.1:5173](http://127.0.0.1:5173)，Go 后端监听 `127.0.0.1:8080`。开发数据和配置在上一级的 `.local-dev/`。以下命令均从本目录执行，前台服务各用一个终端。

### 准备配置与依赖服务

已有 `../.local-dev/backend.env` 时继续使用。首次配置可将 [环境变量示例](.env.example) 复制到该位置，填写数据库、对象存储凭据和初始管理员密码。开发环境需要 PostgreSQL 与 `pgvector`、RustFS、`migrate` CLI、Go 和 Node.js/pnpm。下方 RustFS 命令使用本机开发目录中的 RustFS 二进制。

新配置使用 `AIBERTAI_` 前缀。升级时可沿用旧配置文件。新版变量与对应旧变量同时设置时，以新版变量为准。

启动现有开发数据库：

```bash
pg_ctl -D "$PWD/../.local-dev/pgdata" -l "$PWD/../.local-dev/postgres.log" \
  -o '-h 127.0.0.1 -p 55432 -k /tmp' -w start
```

启动现有 RustFS 服务：

```bash
. ../.local-dev/backend.env
export RUSTFS_ACCESS_KEY="$AIBERTAI_S3_ACCESS_KEY"
export RUSTFS_SECRET_KEY="$AIBERTAI_S3_SECRET_KEY"
export RUSTFS_CONSOLE_ENABLE=false
storage_link=$(mktemp -d /tmp/aibertai-rustfs.XXXXXX)
ln -s "$PWD/../.local-dev/rustfs-data" "$storage_link/data"
../.local-dev/bin/rustfs server --address 127.0.0.1:59000 \
  --console-address 127.0.0.1:59001 "$storage_link/data"
```

RustFS 或 PostgreSQL 已运行时，直接使用配置里的现有地址。新数据库及现有数据库升级均执行 [数据库迁移](backend/migrations/README.md)。数据库、对象存储、远程执行身份目录及配置需一并保留。

### 启动后端和管理后台

```bash
. ../.local-dev/backend.env
mkdir -p ../.local-dev/bin
go -C backend build -buildvcs=false -o "$PWD/../.local-dev/bin/aibertai-server" ./cmd/server
../.local-dev/bin/aibertai-server
```

初始管理员只会在空库创建，后续启动沿用已有账号。后端 `/healthz` 用于检查进程是否存活，`/readyz` 检查数据库和对象存储是否可用。

```bash
. ../.local-dev/backend.env
cd admin
pnpm install --frozen-lockfile
pnpm exec vite --host 127.0.0.1 --strictPort --port 5173
```

`AIBERTAI_PUBLIC_URL` 应设置为浏览器和员工端实际访问的根地址。后端使用它生成 OAuth 元数据和回调。员工端登录时填写该地址。Vite 的代理目标由 `AIBERTAI_DEV_BACKEND_URL` 配置。如需为 API 单独指定目标，可设置 `AIBERTAI_DEV_API_URL`。

## 业务和验证入口

- **管理后台**支持成员、模型、知识库、资源和费用设置。在「其他设置 → 知识库设置」中选择已启用的 OpenAI 兼容 Embedding 模型作为默认模型。已有绑定的知识库可在详情页切换模型并重建索引。Reranker 通过兼容的 `/rerank` 接口调用，失败时沿用原排序。文档视觉解析使用支持图片输入的 Chat Completions 模型，解析 PDF 需要服务端 PATH 中的 Poppler `pdftoppm`。界面开发和验证见[管理后台说明](admin/README.md)。
- **设计资源**通过后台导入资源包，或由部署方提供私有后端的本地目录缓存。设计案例、设计系统和媒体从已登录的 AIbertAI 服务读取。设计风格的目录与缩略图随员工端提供。[资源导入与缓存说明](design/design-resource-import.md)
- **远程执行**使用后端中转和设备审批，身份保存在 `AIBERTAI_RUNTIME_STATE_DIR`。独立执行宿主另见 [Engine 说明](engine/README.md)，后端路由和授权见[远程执行中转](backend/README.md#远程执行中转)。
- **桌面端开发**常规改动用源码检查和针对性联调验证。安装包在需要跨设备验收或发版时构建。[桌面端构建与验收](../../AIbertAI/packaging/README.md)

常规检查：

```bash
make -C backend check
pnpm -C admin test
pnpm -C admin build
```

接口契约见 [OpenAPI](backend/api/README.md)，后端技术说明见 [Backend README](backend/README.md)。`design/` 中的设计和实施计划用于查看背景，操作步骤以本页及链接的指南为准。

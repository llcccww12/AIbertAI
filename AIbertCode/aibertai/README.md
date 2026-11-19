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

## 内网服务器部署

新服务器安装 Docker Engine、Docker Compose v2、Bash 和 OpenSSL 后，从本目录执行。把示例地址换成管理员和桌面端实际访问的完整地址，把邮箱换成自己的管理员邮箱。

```bash
bash deploy/stack.sh init http://192.168.1.20:5173 admin@example.com
bash deploy/stack.sh up
```

`init` 首次生成 `deploy/.env` 和独立密码。管理员初始密码在该文件的 `INITIAL_ADMIN_PASSWORD`，只在空库初始化时生效。已有部署保留该文件，拉取更新后直接执行 `up`。该命令构建管理后台和 Go 后端，启动 PostgreSQL、RustFS，完成数据库迁移，再启动支持设备 WebSocket 的统一入口。数据库、对象存储和设备签名密钥各自保存在 Docker 数据卷中。默认只对客户端开放 `5173`。升级迁移失败时，后端保持停止，先检查迁移错误再恢复服务。

外面另有 HTTPS 反向代理时，将 `deploy/.env` 中的 `PUBLIC_URL` 填为外部 HTTPS 地址，按需设置 `LISTEN_HOST=127.0.0.1` 和 `LISTEN_PORT=5173`。外层代理须转发 API 及设备桥接、远程会话的 WebSocket Upgrade。桌面端登录地址、`PUBLIC_URL` 和 OAuth 元数据的 `issuer` 应完全一致。

部署成功后检查以下入口。若任一服务未启动，可从本目录执行 `docker compose --env-file deploy/.env -f deploy/compose.yaml logs --tail=100`。

- 统一入口的 `/readyz` 返回 HTTP 200，`/.well-known/oauth-authorization-server` 返回正确的 `issuer`。
- 登录管理后台后配置并启用文本模型，授权给桌面账号或所在分组。用桌面端登录，检查「设置 → 模型」并发送一条测试消息。
- `stack.sh up` 从当前源码构建 AMD64 和 ARM64 宿主安装器，随后台镜像提供，并在停止已有服务前校验两个架构的发布清单和 SHA-256。登录后台，在「云电脑 → 添加宿主机」生成命令。执行命令的目标 Linux 主机需要预先安装 Docker 并配置 gVisor，详见 [宿主安装说明](engine/INSTALL.md)。
- 发布标签形如 `host-v26010101` 的 GitHub Release 时，工作流会附上两种架构的安装资源及压缩包校验和。使用固定发布版本时，将已校验的资源解压到 `deploy/data/releases/host-engine`，并将 `ENGINE_RELEASE_DIR=/data/releases/host-engine` 和 `ENGINE_VERSION=v26010101` 写入 `deploy/.env`。保留旧版本目录可继续分发旧安装命令需要的文件。默认镜像只提供当前构建版本，更新镜像后可重新生成安装命令。需要使用后台的宿主版本比较时，使用此类编号发布版本。
- 云电脑工作负载使用独立的运行时发布件。按 [运行时说明](engine/RUNTIME.md) 准备该目录，在 `deploy/.env` 配置 `ENVIRONMENT_RUNTIME_DIR` 和 `ENVIRONMENT_RUNTIME_VERSION`。目录变量使用容器内的 `/data/releases/...` 路径。

迁移已有平台前，备份并恢复 PostgreSQL、RustFS、运行时状态目录和部署私密配置。迁移时尽量保留统一域名和端口，以延续原设备身份。更换登录地址时，桌面端会按新地址重新注册设备。首次初始化数据库只能得到一个空团队。若曾使用手工部署的 5173 静态代理，切换到此入口前先为新实例选择独立监听端口，避免和正在运行的服务冲突。保持已有服务运行时，也可优先将代理换成仓库内的 `admin/scripts/serve-admin.mjs`。

### 升级默认资源桶

新配置默认使用 `aibertai-resources`。Compose 的 `stack.sh up` 会在后端停服期间迁移旧默认桶。自定义桶保持原配置。手工运行后端时，先停止所有使用这两个桶的后端进程，保持 RustFS 运行，使用已配置 S3 环境变量的新版服务执行：

```bash
aibertai-server migrate-resource-bucket
```

命令保留旧桶，按原对象键复制到新桶，相同对象复用，遇到同名异内容时保留两份并报错。复制完成后，取消显式的旧桶配置或设置 `AIBERTAI_S3_BUCKET=aibertai-resources`，再启动后端。对象存储凭据须能列出、读取和写入相应的桶。

回滚时同样先停服，执行 `aibertai-server migrate-resource-bucket --rollback`，成功后设置 `AIBERTAI_S3_BUCKET=monkeyai-resources` 再启动。回滚会把新版新增对象同步到旧桶。若同一键的两个对象已各自更新，先确认应保留哪个版本再重试。Compose 可用 `docker compose --env-file deploy/.env -f deploy/compose.yaml run --rm --no-deps backend migrate-resource-bucket --rollback`，并把桶配置写入 `deploy/.env`。数据库里的资源引用保持原对象键。

### 部署设计案例缓存

在服务器上准备完整的设计缓存目录，将已验收缓存中的 `catalog.json` 和 `assets/` 一起复制过去。也可以在服务器上使用[设计资源同步工具](design/design-resource-import.md#私有后端缓存)，从部署方选定的资源目录生成缓存。

若用上面的 Docker Compose 部署，把缓存放在 `deploy/data/design-cache/`，在 `deploy/.env` 加入 `DESIGN_PUBLIC_CACHE_DIR=/data/design-cache`，再执行 `bash deploy/stack.sh up`。若沿用手工启动，在服务器的 `backend.env` 中加入下面的配置，把路径换成服务器上的实际绝对路径。Go 后端的运行用户须能读取该目录和文件。

```bash
export AIBERTAI_DESIGN_PUBLIC_CACHE_DIR=/srv/aibertai/design-cache
```

保存后重启 Go 后端。通过统一入口访问 `/api/v1/design-resources/public-cache/catalog/templates?page_size=1`，应返回 HTTP 200，且 `data.items` 中有案例。再用桌面端登录同一个入口，打开「海报 → 设计案例」，检查案例和缩略图是否显示。升级时复用服务器上的缓存目录和环境配置。

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

## 文档入口

- 部署、地址、环境变量与升级由本页维护。宿主安装与运行时路径见 [宿主说明](engine/README.md)。
- 接口以 [OpenAPI](backend/api/README.md) 为准，排障、数据库和后端配置见 [后端说明](backend/README.md)。
- 页面组件和浏览器验证见 [管理后台说明](admin/README.md)，桌面开发与构建见 [桌面端说明](../../AIbertAI/packaging/README.md)。

`design/` 中带日期的方案和实施记录供查阅历史。交付改动直接更新对应的现行入口，验证截图和本机编译缓存留在临时目录。已由现行指南覆盖的过程清单可从仓库移除，历史在 Git 中查询。

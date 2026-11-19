# AIbertAI

AIbert 品牌下的企业 AI 办公平台。员工在 macOS 工作台发起会话，管理员在 Web 后台管理团队成员、模型、配额、资源和反馈。

## 目录

| 路径 | 用途 |
| --- | --- |
| `AIbertAI/AIbertAI.app` | macOS 独立员工端，本机双击启动 |
| `AIbertAI/启动员工端.command` | 打开同目录的独立员工端 |
| `AIbertAI/code/app/` | 员工端 Electron 主进程、preload、运行时管理和更新入口 |
| `AIbertAI/code/web/` | 员工工作台前端。当前目录包含可运行的构建产物 |
| `AIbertAI/brand/` | AIbert Logo 和图标 |
| `AIbertAI/native/` | 员工端源码运行时 |
| `AIbertAI/packaging/` | [桌面端构建与验收入口](AIbertAI/packaging/README.md) |
| `AIbertAI/.local-runtime/` | 本机 Electron、原生依赖、用户配置、会话和日志 |
| `AIbertCode/aibertai/admin/` | React、TypeScript 和 Vite 管理后台，包含公开首页和桌面端的浏览器授权页 |
| `AIbertCode/aibertai/backend/` | Go API、身份验证、资源与模型服务、数据库迁移和接口定义 |
| `AIbertCode/aibertai/engine/` | 可单独部署的远程执行引擎源码 |
| `AIbertCode/aibertai/design/`、`AIbertCode/docs/` | 架构、业务设计与历史实现记录 |
| `AIbertCode/.local-dev/` | 本地后台环境配置、服务二进制、PostgreSQL 数据、对象存储数据和日志 |

`AIbertCode/` 是后台工程目录。当前 Go 后端使用本地模块 `aibertai.local/backend`，构建和运行无需 Git。根层的 `backend/`、`mobile/`、`browser-extension/` 等目录属于保留的工程，当前 macOS 员工端和管理后台的本地启动入口见上表。

GitHub 仓库包含当前后端、管理端、工作台构建文件及品牌资源。本机数据库、配置、登录资料、已安装依赖和生成的 macOS 应用由本地环境提供。新检出目录按打包说明准备依赖和构建员工端，再按下方步骤启动服务。

## 架构

```text
员工端 Electron + 本机运行时
  ├─ 读取包内员工界面
  └─ 登录与业务 API ──┐
                      ├─ Vite 入口 / 管理后台  127.0.0.1:5173
管理员浏览器 ──────────┘          │
                            Go API  127.0.0.1:8080
                              ├─ PostgreSQL  127.0.0.1:55432
                              ├─ RustFS      127.0.0.1:59000
                              └─ Ollama      127.0.0.1:11434（本地模型）
```

桌面登录经浏览器授权后回到本机运行时。员工端反馈通过 Go API 保存，管理员在 [用户反馈](http://127.0.0.1:5173/console/settings/feedback) 查看。后端开发时可查看 `AIbertCode/aibertai/backend/api/` 的 OpenAPI 定义。

## 本机开发

管理入口是 [http://127.0.0.1:5173](http://127.0.0.1:5173)。本地管理员为 `admin@aibertai.local`，密码由 `AIbertCode/.local-dev/backend.env` 保存。已有服务运行时，直接双击 `AIbertAI/AIbertAI.app` 即可打开员工端。

服务停止时，从工作区根目录依次启动现有的 PostgreSQL 和 RustFS。命令复用本地已有数据目录。每个前台服务使用单独的终端。

```bash
cd AIbertCode
pg_ctl -D "$PWD/.local-dev/pgdata" -l "$PWD/.local-dev/postgres.log" \
  -o '-h 127.0.0.1 -p 55432 -k /tmp' -w start
```

```bash
cd AIbertCode
. .local-dev/backend.env
export RUSTFS_ACCESS_KEY="$MONKEYAI_S3_ACCESS_KEY"
export RUSTFS_SECRET_KEY="$MONKEYAI_S3_SECRET_KEY"
export RUSTFS_CONSOLE_ENABLE=false
storage_link=$(mktemp -d /tmp/aibertai-rustfs.XXXXXX)
ln -s "$PWD/.local-dev/rustfs-data" "$storage_link/data"
.local-dev/bin/rustfs server --address 127.0.0.1:59000 \
  --console-address 127.0.0.1:59001 "$storage_link/data"
```

启动 Go 后端和 Vite 前端。

```bash
cd AIbertCode/aibertai/backend
go build -buildvcs=false -o ../../.local-dev/bin/aibertai-server ./cmd/server
cd ../..
. .local-dev/backend.env
.local-dev/bin/aibertai-server
```

```bash
cd AIbertCode/aibertai/admin
pnpm exec vite --host 127.0.0.1 --strictPort --port 5173
```

使用 Ollama 模型时，保持其服务运行，并在管理后台配置兼容模型。首次登录时输入员工设备可访问的后端地址，安装包本身与后端地址无关。

桌面端的设计案例和设计系统优先读取本机的公开目录缓存。由工作区根目录运行以下命令可以首次下载或更新缓存。将 `MONKEYAI_DESIGN_PUBLIC_CACHE_DIR` 指向该目录，后端就会提供缓存接口。同步完成后会原子切换目录索引。桌面端仅从登录的 AIbertAI 服务器读取目录和已缓存媒体。容器部署时，把整个缓存目录放在 `AIBERTAI_DESIGN_CACHE_PATH`，并授予容器运行用户目录遍历及文件读取权限。此同步命令是手动导入流程，已完成缓存的服务器和客户端在日常读取时不需要访问来源网站。

```bash
python3 AIbertCode/aibertai/backend/tools/designcache/sync.py "$PWD/AIbertCode/.local-dev/design-cache" \
  --source "https://catalog.example.com/openapi/v1"
```

管理后台的「其他设置 → 知识库设置」可配置 Embedding、Reranker 和文档解析视觉模型。Embedding 使用模型管理中已启用的 OpenAI 兼容 `/embeddings` 模型。默认模型应用于新建和未绑定的知识库，已有知识库可在详情页单独切换并重建索引。Reranker 使用 Cohere 或 Jina 兼容的 `/rerank` 接口，调用失败时保留原有排序。

文档解析需要支持图片输入的 OpenAI 兼容 Chat Completions API。配置后，后台知识库可导入 PDF 和 PNG/JPEG 图片。PDF 导入需要服务端 PATH 中的 Poppler `pdftoppm`，后端容器已包含该工具，本机可用 `brew install poppler` 安装。现有数据库需应用迁移至 `000050_knowledge_models_settings`。

## 文档入口

- [桌面端构建、升级与回滚](AIbertAI/packaging/README.md)
- [后端部署与业务配置](AIbertCode/aibertai/README.md)
- [管理后台开发和验证](AIbertCode/aibertai/admin/README.md)
- [接口契约](AIbertCode/aibertai/backend/api/README.md)

`design/` 保存专题设计及实施记录。实际操作以这些入口和当前接口定义为准。

## 品牌与发布

- 品牌为 **AIbert**，当前产品显示名为 **AIbertAI**。授权完成页按钮使用「打开AIbert办公」。
- 品牌原图在 `AIbertAI/brand/logo.png`。管理后台从 `/api/auth/v1/branding` 读取当前配置。管理端和员工端都内置了同一份回退 Logo。
- 官网是 [世纪艾伯特官网](https://www.century-albert.com/)。员工端支持区使用项目中的微信二维码及电话 **0592-2626666**。
- 正式客户端构建与公证按[桌面端打包说明](AIbertAI/packaging/README.md)执行。发布后再填入 `AIbertCode/aibertai/admin/public/downloads.json`。自动更新仍在发布阶段，需准备签名安装包和 AIbertAI 的 HTTPS 更新源。
- 管理首页和桌面工作台使用 AIbertAI 品牌。已移除旧 Astro 官网工程。OAuth 客户端 ID、协议名和数据库中的历史身份值仍用于兼容现有数据。旧二进制保留在本机回滚归档。
- `backend.env`、数据库、对象存储、桌面用户资料和日志仅用于本机开发。

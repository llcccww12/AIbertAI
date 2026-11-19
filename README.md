# AIbertAI

AIbert 品牌下的企业 AI 办公平台，包含桌面员工端与 Web 管理后台。管理后台负责团队成员、模型、知识库、配额及使用记录。

## 目录

| 路径 | 用途 |
| --- | --- |
| `AIbertAI/AIbertAI.app` | macOS 独立员工端，本机双击启动 |
| `AIbertAI/启动员工端.command` | 打开同目录的独立员工端 |
| `AIbertAI/code/app/` | 员工端 Electron 主进程、preload、运行时管理和更新入口 |
| `AIbertAI/code/web/` | 员工工作台前端。当前目录包含可运行的构建产物 |
| `AIbertAI/brand/` | AIbert Logo 和图标 |
| `AIbertAI/native/` | 员工端源码运行时、设计专家和浏览器桥接 |
| `AIbertAI/browser-extension/` | Chrome 和 Edge 的 AIbertAI 浏览器助手 |
| `AIbertAI/packaging/` | [桌面端构建与验收入口](AIbertAI/packaging/README.md) |
| `AIbertAI/.local-runtime/` | 本机 Electron、原生依赖、用户配置、会话和日志 |
| `AIbertCode/aibertai/admin/` | React、TypeScript 和 Vite 管理后台，包含公开首页和桌面端的浏览器授权页 |
| `AIbertCode/aibertai/backend/` | Go API、身份验证、资源与模型服务、数据库迁移和接口定义 |
| `AIbertCode/aibertai/engine/` | 可单独部署的远程执行引擎源码 |
| `AIbertCode/aibertai/design/` | 架构、业务设计和实施记录 |
| `AIbertCode/.local-dev/` | 本地后台环境配置、服务二进制、PostgreSQL 数据、对象存储数据和日志 |

`AIbertCode/aibertai/` 是当前后台源码目录，Go 后端模块为 `aibertai.local/backend`。桌面端与管理后台的开发入口见上表。

GitHub 仓库包含当前后端、管理端、工作台构建文件及品牌资源。本机数据库、配置、登录资料、已安装依赖和生成的 macOS 应用由本地环境提供。开发环境和本地数据位置由下方启动指南统一维护。

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
                              └─ 管理后台配置的模型服务
```

桌面登录经浏览器授权后回到本机运行时。员工端反馈通过 Go API 保存，管理员在 [用户反馈](http://127.0.0.1:5173/console/settings/feedback) 查看。后端开发时可查看 `AIbertCode/aibertai/backend/api/` 的 OpenAPI 定义。

## 本机开发

按[管理平台启动指南](AIbertCode/aibertai/README.md#本机开发)启动 PostgreSQL、RustFS、Go 后端和 Vite。服务启动后可双击现有 `AIbertAI/AIbertAI.app`，登录时填写该设备可访问的后端地址。开发期间按改动范围运行代码检查与联调，跨设备验收和发布时再构建安装包。

## 服务器部署

新服务器使用仓库的 [Docker Compose 部署入口](AIbertCode/aibertai/README.md#内网服务器部署)。部署文档包含数据库迁移、桌面 WebSocket 代理、设计资源缓存和迁移已有数据所需的配置。

## 文档入口

- [桌面端构建、升级与回滚](AIbertAI/packaging/README.md)
- [管理平台启动与业务配置](AIbertCode/aibertai/README.md)
- [管理后台开发和验证](AIbertCode/aibertai/admin/README.md)
- [接口契约](AIbertCode/aibertai/backend/api/README.md)

`design/` 保存专题设计及实施记录。实际操作以这些入口和当前接口定义为准。

## 品牌与发布

- 品牌为 **AIbert**，当前产品显示名为 **AIbertAI**。授权完成页按钮使用「打开AIbert办公」。
- 品牌原图在 `AIbertAI/brand/logo.png`。管理后台从 `/api/auth/v1/branding` 读取当前配置。管理端和员工端都内置了同一份回退 Logo。
- 官网是 [世纪艾伯特官网](https://www.century-albert.com/)。员工端支持区使用项目中的微信二维码及电话 **0592-2626666**。
- 正式客户端构建与公证按[桌面端打包说明](AIbertAI/packaging/README.md)执行。发布后再填入 `AIbertCode/aibertai/admin/public/downloads.json`。自动更新仍在发布阶段，需准备签名安装包和 AIbertAI 的 HTTPS 更新源。
- 管理首页和桌面工作台使用 AIbertAI 品牌。OAuth 客户端 ID、协议名和数据库中的历史身份值仍用于兼容现有数据。
- `backend.env`、数据库、对象存储、桌面用户资料和日志仅用于本机开发。

# AIbertAI

AIbert 品牌下的企业 AI 办公平台，包含桌面员工端与 Web 管理后台。管理后台负责团队成员、模型、知识库、配额及使用记录。

## 目录

| 路径 | 用途 |
| --- | --- |
| `AIbertAI/AIbertAI.app` | macOS 独立员工端，本机双击启动 |
| `AIbertAI/启动员工端.command` | 打开员工端，使用 `--dev` 启动当前源码和 DSH SDK |
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

按[管理平台启动指南](AIbertCode/aibertai/README.md#本机开发)启动 PostgreSQL、RustFS、Go 后端和 Vite。运行数据库迁移至 `000053` 后，管理员可在专家编辑页设置「DSH 会话插件」。保留「使用 DSH 默认组合」可继承 SDK 预设，也可以为专家单独选择工具。在「画布工具」中可逐项授权读取画布、创建文本等操作，再为专家配置提示词、技能和连接器。

按[桌面端构建说明](AIbertAI/packaging/README.md)准备 SDK，再运行 `bash AIbertAI/启动员工端.command --dev`。员工桌面端新建的本机会话使用 DSH，历史会话按创建时的运行时打开。要体验后台目标，在新会话设置目标并关闭聊天窗口，保持 AIbert 运行。重新打开同一会话可查看期间的进度、工具调用和最终回复。点击输入框下方的上下文百分比，可查看 DSH 报告的总用量及系统提示词、工具定义和对话消息的估算。

在员工端点输入框右下角的模型名称，可切换「启用智能体团队协作」。新会话默认关闭。开启后，智能体可用共享任务板分工；关闭后保留已有任务和队友。管理员可以通过专家的 DSH 会话插件预设限制可用工具。

同一员工在同一设备、同一服务器上的会话共享本机插件配置。通过会话中的插件管理工具可以列出插件和插件包，并经员工单次确认后启停时间感知、待办、文件搜索、向用户提问或文件交付。配置变更在各会话下一轮执行前加载，会话历史保存在磁盘。更换插件组合会重启对应的 SDK 子进程，原进程中运行的 Bash 作业会结束。连续执行的 Goal 由桌面宿主管理；通用定时和周期任务仍待接入统一调度。

### 员工端的 DSH 能力

| 使用场景 | 当前行为与范围 |
| --- | --- |
| 查看或追问团队成员 | 点会话顶部的子智能体数量，或点创建记录、成员通知里的「打开成员会话」。完成后仍可查看本地成员的记录。可继续对话的成员有独立的文字输入框，点顶部主会话标题即可返回。一次性成员仅提供记录，外部智能体由其原服务管理。 |
| 成员中断 | 主管收到 DSH 的成员结果通知，可以读取已有文件、继续验证、重新分工。是否需要再派成员由主管根据任务判断。员工也可以在可继续对话的成员里补充说明。需要员工工作区写入审批的编辑可由成员通过消息把相对路径和改动交给主管执行。 |
| 模型网关失败 | 会话保留失败原因和已返回的内容，与 DSH 一样不锁会话，随后可直接在原会话继续；只有上一轮的模型调用仍待网关回执时需要先点「重新核验」。结果未知的工具调用由模型先核对现状再决定是否重做。限流、额度或访问权限等已确认的错误会提示处理方式。回执暂缺时可点「重新核验」，只查询上一次调用的结果。 |
| 专家和全局插件 | 专家的角色、技能和会话工具由预设决定，团队能力也受该专家可用工具限制。后台的全局插件页展示此版本的插件清单和加载条件，个别会话的实际可用工具以其预设、授权及宿主能力为准。 |
| 计划和上下文 | 输入框可查看模型上下文用量，会话支持手动压缩。员工端还缺少 DSH 原生的独立计划审阅侧栏（提交、批准、重新打开计划）。 |
| 工作区与后台执行 | 与 DSH 一样，`~/Documents/AIbertAI/default-workspace` 作为「默认工作区」项目列在侧栏，新会话默认归入它，删除会话不删除其中文件；文件面板默认隐藏以 `.` 开头的内部目录。权限与 DSH 对齐为「仅可查看」「工作区内修改」（默认）「完全权限」三档，分别对应 DSH 的 `read-only`、`workspace-write`、`danger-full-access` 预设，会话内可随时切换：新档位立即作用于宿主的文件写入检查，并在下一轮开始时通过 DSH 的 `/permission` 命令写进正在运行的会话；远程中转、后台目标和没有写文件权限的访问者始终只读。文件交付、预览、待办和运行用时可在会话中查看。后台目标要求员工端进程保持运行。当前成员输入支持文字；通用定时任务和运行时插件代码热重载尚未开放。 |

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

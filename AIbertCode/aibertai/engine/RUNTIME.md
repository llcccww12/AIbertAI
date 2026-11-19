# 云电脑运行时编排

新建云电脑使用仓库内 `AIbertAI/native` 编译的 Linux `aibertai-native`。同一进程负责设备授权、远程会话和工具执行，容器自行保存独立的设备授权。

## 镜像与目录契约

管理员提供固定摘要的 Linux OCI 镜像，必须包含：

| 路径 | 用途 |
| --- | --- |
| `/usr/bin/tini` | PID 1，负责回收孤儿进程 |
| `/bin/sleep` | 支持 `infinity` 的驻留进程 |

镜像根只读；程序和动态库必须允许 UID/GID 1000 执行。托管规格忽略镜像 Entrypoint/Cmd，固定运行 `tini -g -- /bin/sleep infinity`，并使用固定的 PATH、HOME、USER、LOGNAME、LANG，不继承镜像的身份及执行环境变量。宿主按规格下载独立的、带版本和 SHA-256 的运行时。镜像提供系统依赖和工作负载工具。

新容器的 runtime 卷挂载在 `/var/lib/aibertai`，实际传入 `--state-dir /var/lib/aibertai/state`。设备身份、会话和待补传记录在该状态卷持久化。项目与用户目录分别为 `/workspace` 和 `/home/aibertai`。已有容器继续使用 `/var/lib/monkeyai` 和 `/home/monkey`。原容器的可写层和持久卷继续使用，运行时根据容器的挂载布局选择状态目录。

托管规格必须使用 `public-only`，并确保固定 DNS、后台 HTTP(S)、中转 WS(S) 和所需服务可达。后台位于同一 Linux 宿主时，可启用[同机内网后台网关](NETWORK.md#同机内网后台)，仅允许设备运行时访问后台。`isolated` 保留给本地容器执行，不可用于客户端云电脑登录。

默认镜像可运行 `bash engine/image/prepare.sh /绝对路径/cloud-oci` 构建。
脚本校验两种架构的清单、配置和每个镜像层。用后台的 `environment-images` 命令导入并做
`--check` 校验。安装后设置 `AIBERTAI_ENVIRONMENT_IMAGE_STATE_DIR` 为该镜像仓库的持久目录。
同机部署可在后台与宿主进程同时设置 `AIBERTAI_CLOUD_IMAGE_REGISTRY=127.0.0.1:8080`（必须等于后台回环监听地址），让宿主 Docker 通过本机回环拉取镜像。
镜像引用始终由控制面固定到已验证的摘要。

## 启动流程

1. 后台冻结 `runtime.server/server_id/user_id` 到规范化实例规格；宿主必须报告 `runtime_protocol=1` 与 `runtime_update_protocol=3`，并校验发行物后再启动容器。
2. engine 创建并启动受限容器，在同一实例内调用固定 runtime 的 `status`。仅解析公开 machine ID、公钥和账号状态，不读取私钥文件。
3. 首次初始化或凭据不可用时，运行 `login --device-code --state-dir … --server … --server-id … --user …`；收到 authorize 事件后再次调用 status 取公开身份。
4. 设备申请通过既有签名 exchange 的 runtime 数组提交，绑定宿主、环境、实例、代次、公钥及用户操作。后台只批准现有固定 scopes，拒绝身份变化、旧实例、已停用用户、已撤销设备和已结束操作。首次绑定必须有设备申请；之后允许复用同一持久身份。
5. runtime 自行轮询、保存凭据。engine 核对登录后的账号与身份，再使用固定 exec ID 启动 `serve --state-dir …`。进程由 Docker 管理，stdio 使用 NullIO，engine 重启不终止它。
6. engine 仅认领仍运行的 serve，不自动重新执行已退出进程。观察阶段不重新登录。启动失败锁定本代次，用户需要停止清理后显式启动新实例。

登录和 status 的输出只进入有界内存，设备码不写入控制日志或用户界面。状态查询最长 15 秒，登录最长 11 分钟，整个编排最长 12 分钟。临时 exec 完成或取消后回收；退出不能确认时报告失败。

## 连接与授权

容器 running 不表示可连接。列表和详情返回 machine_id、runtime_ready、connectable、stale；只有当前代次观察新鲜、容器运行、持久身份属于本实例、设备证明有效且中转控制／数据双通道在线时才可连接。托管创建操作在该连接条件满足后才完成。客户端沿用既有 endpoint connection ticket，不新增另一套连接协议。

普通停止立即禁止新的设备绑定、执行访问和连接签发，关闭中转并撤销连接票据，保留 runtime 授权族与整个状态卷。暂时不可执行返回可重试的授权状态错误，不向 runtime 发送永久撤销信号。重启复核当前实例和代次的设备身份后恢复；旧实例的迟到报告不能恢复授权。

删除在控制事务内撤销与该 machine 关联的 runtime 授权族和 token，再通知中转与执行密钥治理。显式撤销设备不自动恢复。用户停用、宿主撤销和执行失败同样禁止访问；失联隔离、跨宿主恢复及租约 fencing 仍属于后续运维能力。

## 发布与验收

在后端配置 `AIBERTAI_ENVIRONMENT_RUNTIME_DIR`，目录使用 `<version>/<arch>/manifest.json` 和 `<version>/<arch>/aibertai-native`。用 `AIBERTAI_ENVIRONMENT_RUNTIME_VERSION` 或 `latest.json` 选择版本。清单记录版本、架构、源码修订，以及单个静态 ELF 的 SHA-256 和大小。 从 `AIbertAI/native` 使用 `CGO_ENABLED=0 GOOS=linux GOARCH=<架构> go build -trimpath -o <输出目录>/aibertai-native .` 构建。新建云电脑要求对应架构的源码运行时和协议 3 宿主。已有设备跨格式升级需要当前原设备的签名。

Docker Linux 已验证设备授权、代次复核、远程文件工具、重启恢复、旧格式会话读取和工具上报补传。生产宿主的 gVisor、镜像与网络隔离从实际宿主验收，`workload_verified` 仍保持 false。

## 已绑定设备的原地迁移

环境须先停止并由宿主确认。保留运行时、工作区和用户目录的备份。核对原设备 ID、公钥、环境 ID、后台 ID、用户 ID 和已确认停止的代次。原 Ed25519 私钥在 Linux 宿主本地以受限权限文件提供，支持 32 字节原始种子或 PKCS#8 PEM。

以状态卷所属用户运行源码运行时自带的准备命令。原路径仍由宿主挂载，`--state-dir` 指向原运行时的 `state` 目录。

```bash
aibertai-native prepare-cloud-migration \
  --state-dir "$STATE_DIR" --key-file "$ORIGINAL_ED25519_KEY_FILE" \
  --server "$BACKEND_ORIGIN" --server-id "$BACKEND_ID" --user "$OWNER_ID" \
  --machine "$ORIGINAL_MACHINE_ID" --public-key "$ORIGINAL_PUBLIC_KEY" \
  --environment "$ENVIRONMENT_ID" --generation "$STOPPED_GENERATION" \
  --version "$SOURCE_RUNTIME_VERSION"
```

把命令输出的 JSON 提交到 `POST /api/v1/environments/{id}/runtime/upgrade`。只提交签名和升级参数，私钥文件留在宿主。后台校验原公钥、环境归属、代次和目标版本。升级成功后启动环境，使用同一设备 ID 和公钥重新授权，再核对会话和工作区。重复提交同一 JSON 会返回同一操作。保留旧密钥格式和运行时资源，直到实机验证通过。

# AIbertAI 宿主安装和升级

宿主需要运行 systemd 的 Linux（amd64 或 arm64）、本机 Docker 和已注册的 `aibertai-runsc` gVisor runtime。宿主服务启动后会预检这些依赖。Docker 和 gVisor 由宿主管理。已有的 `monkeyai-runsc` 注册及旧 runsc 二进制路径仍可用于现有宿主。

## 添加宿主

在管理后台的云电脑页面选择「添加宿主机」，在目标 Linux 宿主执行生成的短期命令。后台按架构提供 `installer` 和 `aibertai-engine`，二者是摘要一致的同一个自有 Go 程序。脚本校验安装器和发行清单，再通过 systemd 启动安装。

新宿主使用 `/etc/aibertai-engine`、`/var/lib/aibertai-engine` 和 `aibertai-engine.service`。程序安装在 `/opt/aibertai-engine/aibertai-engine`。云电脑运行时通过独立发行提供，只加载 `aibertai-native`。

## 升级宿主

从管理后台发起升级，或执行管理后台生成的固定版本 `/api/host/v1/upgrade.sh?version=...` 命令。升级沿用已有后台地址、宿主 ID、私钥、配置和持久卷。升级程序校验目标版本，保存原程序与配置并切换 systemd 服务。只有后台接收到新版本签名心跳后，宿主才恢复任务同步。

服务启动或心跳确认失败时会恢复原配置、程序和服务。升级中断后重跑同一命令可先恢复原状态。新布局的日志可通过 `journalctl -u aibertai-engine` 查看。

已有云电脑如需从旧执行程序转换，先停止环境并完成原设备身份签名迁移，再选择新版运行时。迁移继续使用原有持久卷。宿主升级与云电脑运行时升级分别记录状态。

## 迁移已有宿主名称

先用常规升级把已登记宿主升级到支持新布局的版本。升级会继续使用旧路径，设备无需重新授权。等待宿主升级任务结束后，在维护窗口执行：

```bash
sudo /opt/aibertai-engine/aibertai-engine migrate-host-brand
```

命令暂停旧宿主服务，原地移动配置和状态目录，保留宿主私钥、设备绑定、卷映射和工作区，然后启用新服务。旧状态根路径保留为指向同一目录的链接，供已有 Docker 容器重新挂载。自定义数据目录继续使用原位置。若目标已有另一份数据，命令会停止。中断后可重跑相同命令。

要恢复旧布局，继续用同一兼容版本执行 `sudo /opt/aibertai-engine/aibertai-engine migrate-host-brand --rollback`。回滚移动当前数据并启用旧服务名，保留迁移后写入的数据。Docker 和 gVisor 的安装文件由宿主管理，准备新宿主时使用 `/opt/aibertai-engine/current/bin/runsc`。已有运行中的容器继续使用原挂载目标，避免替换容器导致可写层丢失。

## 本地发布资源

`scripts/package.py` 从当前 Go 源码生成指定架构的固定版本宿主目录。后台生成命令前会检查两个架构的清单。现有发行版本保持可复查。

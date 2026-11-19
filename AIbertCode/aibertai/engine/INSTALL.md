# AIbertAI 宿主安装和升级

宿主需要运行 systemd 的 Linux（amd64 或 arm64）、本机 Docker 和已注册的 `monkeyai-runsc` gVisor runtime。宿主服务启动后会预检这些依赖。Docker 和 gVisor 由宿主管理。

## 添加宿主

在管理后台的云电脑页面选择「添加宿主机」，在目标 Linux 宿主执行生成的短期命令。后台按架构提供 `installer` 和 `monkeyai-engine`，二者是摘要一致的同一个自有 Go 程序。脚本校验安装器和发行清单，再通过 systemd 启动安装。

宿主私钥、配置和状态路径沿用原有布局。新版程序安装在 `/opt/aibertai-engine/aibertai-engine`，由已有的 `monkeyai-engine.service` 服务名管理。云电脑运行时通过独立发行提供，只加载 `aibertai-native`。

## 升级宿主

从管理后台发起升级，或执行管理后台生成的固定版本 `/api/host/v1/upgrade.sh?version=...` 命令。升级沿用已有后台地址、宿主 ID、私钥、配置和持久卷。升级程序校验目标版本，保存原程序与配置并切换 systemd 服务。只有后台接收到新版本签名心跳后，宿主才恢复任务同步。

服务启动或心跳确认失败时会恢复原配置、程序和服务。升级中断后重跑同一命令可先恢复原状态。日志可通过 `journalctl -u monkeyai-engine` 查看。

已有云电脑如需从旧执行程序转换，先停止环境并完成原设备身份签名迁移，再选择新版运行时。迁移继续使用原有持久卷。宿主升级与云电脑运行时升级分别记录状态。

## 本地发布资源

`scripts/package.py` 从当前 Go 源码生成指定架构的固定版本宿主目录。后台生成命令前会检查两个架构的清单。现有发行版本保持可复查。

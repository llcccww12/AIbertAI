# 设计资源导入与离线分发

后端支持设计资源 v1 发布包，四类目录独立于 Agent 资源保存。控制面与本地资源完整 base 均为 `<AIBERTAI_PUBLIC_URL>/api/v1/design-resources`；hub base 由配置原样下发。实现计划与验证状态见 [实施方案](design-resource-import-plan.md)，包格式见 [本地契约](design-resource-contract-v1.md)。

## 私有后端缓存

没有导入发布包时，可先把管理员选定的目录与媒体同步到私有后端的本地磁盘。在平台目录运行：

```bash
python3 backend/tools/designcache/sync.py "$PWD/../.local-dev/design-cache" \
  --source "https://catalog.example.com/openapi/v1"
```

在后端环境设置 `AIBERTAI_DESIGN_PUBLIC_CACHE_DIR` 为该目录。设计案例与设计系统由桌面端从登录时填写的后端读取。同步脚本由部署方运行，来源地址由 `--source` 指定。

## 发布包部署配置

升级前按[数据库迁移说明](../backend/migrations/README.md)备份数据库与对象存储并执行迁移。

| 环境变量 | 用途 |
| --- | --- |
| `AIBERTAI_DESIGN_PUBLISHER_ID` | 可选。配置时只接受匹配的发布者 |
| `AIBERTAI_DESIGN_RESPUB_URL` | 可选。部署方提供的外部发布目录完整地址，默认留空 |
| `AIBERTAI_DESIGN_RESPUB_JWT_SECRET` | 可选。由对接发布目录的服务端共享 |
| `AIBERTAI_DESIGN_MEDIA_JWT_SECRET` | 可选。独立的本地发布快照媒体签名密钥 |

签名密钥的源码默认值为空，由部署方通过运行环境提供。发布快照的签发接口还要求宿主通过 `server.InstanceIdentity` 提供实例身份。缺少签发配置时，相应凭据接口返回 503。本地目录缓存单独配置。无发布快照时，快照 API 返回 `available=false`、`snapshot=null`，快照目录请求返回 `404 snapshot_not_found`。

每个部署只激活一个发布者。未配置预期发布者时，首次激活绑定发布包中的发布者。实际发布者从快照读取。

## 管理台流程

1. 打开“资源管理 → 导入资源包”，上传 ZIP、字节分卷或选择解压目录。总大小上限2 GiB，封存上传按原保留期保存24小时。
2. 服务端先发现可用分区。选择 Agent、设计或二者，再解析并预览；旧未版本化设计分区显示不支持，不算设计成功。
3. 核对 Agent 新增/更新/停用，以及设计发布者、源 release ID、四类计数和快照变化。选择设计不计划 Agent 退役；选择 Agent 表示确认该发布者的完整 Agent 清单。
4. 确认后固定 Agent→设计分别提交。部分成功保留成功结果并显示失败原因，不把HTTP200等同全部成功；失败分区需重新预览后重试。
5. 同一上传在有效期内可补导另一分区。成功回执重放只返回原结果，不重新应用或激活；处理完成后点击“完成”，在历史中查看逐分区状态与源 release。

设计严格校验整包 CHECKSUMS、发布身份、产物 SHA、业务摘要、逻辑源文件和 taxonomy；未选 Agent 的业务内容不解析，但其外包原始字节仍包含在完整性校验内。Agent-only 继续使用旧宽松业务解析。校验和证明完整性，不提供发布者签名认证；包来源由已授权管理员确认。

同 publisher/release 的不同设计快照指纹返回409；指纹包含 taxonomy、排序、来源和产物引用，不仅比较单个资源摘要。旧快照、源文件和已生成 ZIP 保持不可变。损坏内容、过期计划或事务失败不会替换 current。

## 客户端协议

- config/token：使用现有 Agent OAuth Bearer，成功返回裸对象，错误沿用控制面 error 信封，no-store。
- JSON/ZIP：使用现有 Agent OAuth Bearer；资源响应使用 `{code:0,message:"",data:...}`，错误 code 为字符串，data=null。
- 媒体：先以自己的已登记 machine_id 请求 target=monkeyai-media，使用独立本地媒体 JWT 的 Bearer 或临时 query token。存在 Authorization 时优先验证它，失败不降级；hub JWT 和 Agent access token 不能代替本地媒体 JWT。
- hub：target=respub 返回 HS256/iss=monkeyai/aud=respub-design 的短期JWT，client_id取服务端验证后的MachineID，instance_id 由宿主提供。使用配置的发布目录地址和对应签名密钥。
- 从 config 读取完整 base，再请求 `/catalog/snapshot`，后续固定 snapshot/publisher_id/content_digest。明确指定的未知快照404、摘要不符409，不静默回落 current；401最多刷新一次，403不切源绕过。
- 模板/系统下载直接读取冻结 ZIP，不302到S3。Range、HEAD、If-None-Match、If-Range 使用真实字节强ETag；鉴权先于304/206。媒体只允许image/video/audio，SVG加sandbox CSP，缓存私有。
- 原 metadata.skillPath 保留；模板详情 source_skill_path 是独立解析的源文件逻辑路径，无技能时null，不参与业务摘要。客户端可以按需读取该字段。

源文件按 catalog 记录的逻辑路径恢复，ZIP包含原源文件，不加入生成的发布元数据或避让副本。可恢复已记录的包装路径，不能重现未记录的原 ZIP 容器字节。业务摘要及逐文件字节与发布源保持一致，ZIP容器SHA不承诺跨渠道相同。

## 临时文件与对象存储

设计导入流式处理，外包最多2 GiB，整次实际读取展开字节累计4 GiB；不再单独限制ZIP条目数、元数据文件大小、嵌套包展开大小或累计保留的发布元数据。拼接/解析/预览/确认的并发请求最多2个，预算在下载上传块或解析multipart前取得，取消等待会释放请求；服务端应为上传拼接、源对象及预生成ZIP预留足够临时磁盘空间。

可将 `TMPDIR` 设置为后端运行用户可写的本地磁盘目录。请求成功/失败正常退出时清理临时文件；进程异常退出留下的临时目录需在确认没有活跃导入后清理，不应删除已有快照对象。

S3复用现有bucket，需要条件创建、HeadObject和字节Range读取能力。对象按SHA寻址，已存在同key必须核对SHA/大小；事务失败不立即删除共享对象。当前保留未引用的内容地址对象，自动回收尚未实现。过期上传会清理传输块，分区成功/失败历史与回执保留。

## 本地验证

普通测试读取 `backend/testdata/design-contract/v1` 的冻结夹具。修改夹具时同步核对 `SHA256SUMS`。大包测试通过 `MONKEYAI_DESIGN_FIXTURE_ZIP` 显式启用。

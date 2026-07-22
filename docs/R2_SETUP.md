# Cloudflare R2 文件存储配置

耳语把文件正文直接从浏览器上传到私有 Cloudflare R2 Bucket。应用服务器只保存文件元数据、生成短期签名并检查权限，不中转文件内容，因此服务器的 2 GB 本地空间不会被聊天文件持续占用。

## 1. 数据流与安全边界

```text
浏览器 ──申请上传──> Whisper Go 服务 ──生成 5 分钟签名──> 浏览器
浏览器 ───────────────带签名 PUT───────────────> 私有 R2 Bucket
浏览器 ──确认完成──> Whisper Go 服务 ──HEAD 校验大小和类型──> R2
浏览器 ──发送消息──> Whisper Go 服务 ──事务绑定附件──> SQLite
浏览器 ──打开附件──> Whisper Go 服务 ──鉴权并 307 跳转──> R2 短期下载地址
```

- R2 Bucket 必须保持私有，不需要启用公开开发 URL 或自定义公开域名。
- R2 Access Key 和 Secret Key 只配置在 Go 服务进程中，绝不能写进 `app.js`、反向代理配置或浏览器环境。
- 浏览器拿到的只是单个对象、短时间有效的预签名 URL，不能列出或访问其他对象。
- 私聊附件只允许消息双方访问；群附件还会检查当前群成员身份、历史消息边界和个人清空记录边界。

实现依据可参考 Cloudflare 官方的 [AWS SDK for Go V2 示例](https://developers.cloudflare.com/r2/examples/aws/aws-sdk-go/)、[预签名 URL 文档](https://developers.cloudflare.com/r2/api/s3/presigned-urls/) 和 [用户上传内容参考架构](https://developers.cloudflare.com/reference-architecture/diagrams/storage/storing-user-generated-content/)。

## 2. 创建 Bucket 和 API Token

1. 在 Cloudflare Dashboard 进入 **R2 Object Storage**，创建一个 Bucket，例如 `whisper-files`。
2. 创建 R2 API Token，权限选择可对该 Bucket 执行对象读写的权限，并尽量把授权范围限制到这个 Bucket。
3. 创建后保存 `Access Key ID` 和 `Secret Access Key`。Secret 只显示一次。
4. 在 Dashboard 右侧或 R2 概览中找到 Cloudflare `Account ID`。

应用使用的 S3 API Endpoint 由 Account ID 自动组成：

```text
https://<ACCOUNT_ID>.r2.cloudflarestorage.com
```

这里不需要填写 Endpoint 环境变量，也不需要配置 Bucket 的公开地址。

## 3. 配置浏览器上传 CORS

在 Bucket 的 CORS 设置中填写：

```json
[
  {
    "AllowedOrigins": [
      "https://chat.example.com",
      "http://localhost:8080",
      "http://127.0.0.1:8080"
    ],
    "AllowedMethods": ["PUT"],
    "AllowedHeaders": ["Content-Type"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3600
  }
]
```

`AllowedOrigins` 的语义是“用户浏览器地址栏里打开耳语页面时的 Origin”。Origin 只包含协议、主机和非默认端口：

| 页面地址 | 应填写的 Origin |
|---|---|
| `https://chat.example.com/room` | `https://chat.example.com` |
| `https://chat.example.com:8443/` | `https://chat.example.com:8443` |
| `http://localhost:8080/` | `http://localhost:8080` |

不要填写 R2 Endpoint、Bucket 名、服务器内网 IP（除非浏览器就是通过它访问页面），也不要带 `/`、路径或查询参数。`http` 和 `https`、`localhost` 和 `127.0.0.1`、不同端口都属于不同 Origin，实际会使用哪些就分别列出哪些。生产环境不建议使用 `*`。

当前前端直传只使用 `PUT` 和 `Content-Type`。`ETag` 暂未参与业务校验，但保留后便于以后支持分片或完整性校验。Cloudflare 的 [JavaScript S3 示例](https://developers.cloudflare.com/r2/examples/aws/aws-sdk-js/) 也说明了预签名浏览器请求需要匹配 Bucket CORS。

## 4. 配置服务端环境变量

程序读取以下四项，示例也见项目根目录的 `.env.example`：

| 环境变量 | 示例 | 意义 |
|---|---|---|
| `WHISPER_R2_ACCOUNT_ID` | `012345...` | Cloudflare Account ID，用于构造私有 S3 API Endpoint |
| `WHISPER_R2_ACCESS_KEY_ID` | `abc...` | R2 API Token 的 Access Key ID |
| `WHISPER_R2_SECRET_ACCESS_KEY` | `xyz...` | R2 API Token 的 Secret Access Key |
| `WHISPER_R2_BUCKET` | `whisper-files` | 保存聊天文件的 Bucket 名 |

本地临时运行：

```bash
export WHISPER_R2_ACCOUNT_ID='你的 Account ID'
export WHISPER_R2_ACCESS_KEY_ID='你的 Access Key ID'
export WHISPER_R2_SECRET_ACCESS_KEY='你的 Secret Access Key'
export WHISPER_R2_BUCKET='whisper-files'

go run ./cmd/whisper -addr 127.0.0.1:8080 \
  -db data/whisper.db \
  -users-backup data/users-backup.json \
  -static .
```

若使用 systemd，把四项写到仅 root 可读的环境文件，例如 `/etc/whisper/whisper.env`：

```ini
WHISPER_R2_ACCOUNT_ID=你的AccountID
WHISPER_R2_ACCESS_KEY_ID=你的AccessKeyID
WHISPER_R2_SECRET_ACCESS_KEY=你的SecretAccessKey
WHISPER_R2_BUCKET=whisper-files
```

权限设为 `0600`，然后在 service 的 `[Service]` 中加入：

```ini
EnvironmentFile=/etc/whisper/whisper.env
```

重载并重启服务：

```bash
sudo systemctl daemon-reload
sudo systemctl restart whisper
sudo journalctl -u whisper -n 50 --no-pager
```

启动日志中的 `r2_enabled=true` 表示配置已启用。四项全部不设置时，文字和 Emoji 仍可用，文件页签及自定义表情包添加入口显示为不可用；只设置一部分会让服务拒绝启动，避免误以为上传已经生效。

## 5. 应用内限制

- 单文件最大 `50 MB`。
- 单条消息最多 `5` 个文件。
- 单条消息附件总大小最大 `100 MB`。
- 支持一次继续添加多个文件，发送前可删除；失败卡片保留，可再次发送重试。
- GIF、JPEG、PNG、WebP 图片不超过 `10 MB` 时直接展示；更大的图片以文件卡片显示，用户主动查看时才加载。
- 自定义表情包只支持上述四种图片，单张最大 `10 MB`，每位用户最多收藏 `120` 张。
- MP4、WebM 和 Ogg 视频使用站内播放器，其他类型显示下载卡片。
- 上传和普通下载签名有效期为 `5 分钟`；视频读取签名为 `2 小时`，供播放和 Range 请求持续使用。
- 未发送草稿超过 `24 小时`后，在服务启动或下一次申请上传时清理 R2 对象和 SQLite 元数据。撤回、解散群聊等操作产生且不再被任何有效消息或个人表情包引用的对象也进入同一可重试清理流程。

这些是耳语自身的产品限制，不是 R2 的单文件硬上限。要修改它们，需要同步调整 `internal/chat/attachments.go` 和 `app.js` 中的限制；只改前端不能绕过服务端校验。

## 6. 本次代码改动及意义

| 位置 | 改动 | 意义 |
|---|---|---|
| `internal/blob` | R2 配置、S3 客户端、PUT/GET 预签名、HEAD 和删除 | 把对象存储封装为可替换接口，密钥仅留在服务端 |
| `internal/chat/attachments.go` | 附件草稿、绑定、访问权限和大小限制 | 用 SQLite 维护对象归属和消息可见性 |
| `internal/chat/store.go` | schema v7、撤回时间与媒体引用表 | 消息、附件、表情包及转发引用在同一事务中提交，旧数据库自动迁移 |
| `internal/chat/messages.go` | 两分钟撤回事务与接收者计算 | 撤回后同步通知消息当前可见成员 |
| `internal/web/attachments.go` | 申请、完成、下载、删除 API | 浏览器直传，服务端仍控制对象生命周期和下载权限 |
| `internal/web/stickers.go` / `messages.go` | 收藏、移除与撤回接口 | 在服务端执行归属、可见性和时间窗口校验 |
| `index.html` / `styles.css` / `app.js` | 三栏内容弹层、媒体消息、查看器和自定义播放器 | 完成表情包收藏、图片缩放切换、视频播放、下载、转发和撤回交互 |
| `cmd/whisper/main.go` | 读取四项 R2 环境变量 | 未配置时保持原功能，配置完整时自动启用上传 |

服务会自动把私有 R2 S3 Origin 加入页面的 `connect-src`、`img-src` 和 `media-src` CSP。反向代理不需要放宽聊天 API 的请求体限制，因为文件内容不经过 Go 服务或代理，只传少量 JSON 元数据。

## 7. 排错

浏览器提示 CORS：检查地址栏 Origin 是否和 `AllowedOrigins` 完全一致，尤其是协议和端口；修改 R2 CORS 后重新打开页面再试。

服务启动报“R2 配置不完整”：四个变量必须同时存在，检查 systemd 的 `EnvironmentFile` 路径和权限。

上传返回 `403 SignatureDoesNotMatch`：确认浏览器发送的 `Content-Type` 未被代理或扩展改写，并确认 API Token 属于同一个 Account 和 Bucket。

文件上传成功但确认失败：应用会通过 HEAD 比较申请时的大小和类型；不一致的对象会被删除，卡片保留为失败状态供重试。

下载返回 `403`：该地址经过耳语权限检查。确认当前账户仍能看到所属消息，仍是群成员，并且没有清空该会话记录。

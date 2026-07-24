# 后端架构

## 目标

该架构针对单机、四五人长期运行的场景，使用嵌入式 SQLite 保证状态一致性，不引入 Redis、消息队列或多节点协调。

## 组件

```text
浏览器
  ├─ HTTP JSON API：认证、资料、密码、设置、附件签名与鉴权
  ├─ WebSocket：群聊、私聊、在线状态、离线消息送达
  └─ Web Push：Service Worker 系统通知
           │
           ▼
internal/web
  ├─ Server：路由、会话 Cookie、安全响应头
  └─ Hub：连接管理、用户广播、在线状态
           │
           ▼
internal/chat
  └─ Store：账户、好友、消息、投递状态、SQLite 事务
           ├─ data/whisper.db：全部业务数据
           └─ data/users-backup.json：用户信息与明文密码备份

浏览器 ──预签名 PUT / 短期 GET──> 私有 Cloudflare R2 Bucket
```

## 数据模型

- `User`：稳定 ID、名称、bcrypt 密码哈希、签名、设置、好友、好友消息颜色和各会话清除时间。
- `Group`：稳定 ID、群名、群个签、群主、历史可见策略和系统群标记。
- `GroupMember`：群 ID、用户 ID 和该成员可见消息的 `history_from` 边界。
- `Message`：稳定 ID、群聊/私聊类型、群 ID、发送者 ID、接收者 ID、正文、旧版表情 ID、发送/送达时间和可选撤回时间。
- `Attachment`：稳定 ID、上传者 ID、私有对象键、原文件名、内容类型、大小和 `pending` / `ready` / `attached` 状态。
- `MessageAttachment`：消息与附件的多对多引用及展示顺序。转发只新增引用，不复制 R2 对象。
- `UserSticker`：用户收藏的图片附件及排序位置。
- `MessageSticker`：自定义表情消息与图片附件的一对一引用。
- `Session`：随机令牌只通过 HttpOnly Cookie 发给浏览器，SQLite 只保存 token 哈希、用户 ID、创建时间和过期时间。
- `PushSubscription`：用户设备的 Web Push endpoint、公钥和认证密钥；只用于向已授权设备发送加密通知。
- `ConversationRead`：用户在每个会话最后实际看到的消息时间和消息 ID，用于持久化未读位置。
- `ServerInstance`：每次服务启动生成的实例标识，通过 bootstrap 返回，用于客户端检测更新后自动刷新资源。

用户名可以修改，但消息和好友关系使用稳定用户 ID，因此改名不会破坏历史会话。

## coco 隔离

`coco` 使用保留 ID，不存在可登录账户。发给 coco 的消息以当前用户 ID 和 coco ID 组成私有会话键。其他用户在查询消息时无法匹配该参与者关系，因此看不到该会话。

## 离线投递

私聊对象离线时，消息的 `deliveredAt` 为空，发送端显示“待送达”。接收者调用启动数据接口或建立 WebSocket 时，服务标记消息已送达，并向仍在线的发送者推送 `delivered` 事件。

启用 VAPID 配置后，私聊接收者和群聊成员会同时收到 Web Push。Service Worker 在没有可见前台页面时显示系统通知，点击通知会打开对应会话。Push endpoint 返回 `404` / `410` 时会自动从 SQLite 清理。Web Push 依赖浏览器厂商的外部推送服务，内网环境需要允许浏览器和服务端访问这些服务。

未读数由 SQLite 中的 read cursor 计算，而不是只保存在当前标签页。前端仅上报最后已经渲染的消息 ID，服务端校验消息属于该用户可见的会话后单调推进 cursor；同一账号的其他在线标签页通过 WebSocket `read` 事件同步清除未读状态。

启动数据接口只返回每个会话最近 50 条可见消息，并单独计算完整未读数。浏览器通过消息时间与稳定消息 ID 组成的游标向前分页，每次再取 50 条；查询同时遵守群成员历史边界和用户清空记录边界。前端进入会话时定位到最新消息，只有主动加载旧消息时才保留原阅读位置。

## 持久化

业务数据使用 `database/sql` 和纯 Go `modernc.org/sqlite` 驱动，主要表包括：

- `users`：账户、bcrypt 哈希、签名和设置。
- `friends`、`friend_colors`：好友关系和本地消息颜色。
- `groups`、`group_members`：公共大厅和普通群聊、成员关系及历史可见边界。
- `messages`：各群聊、私聊、离线送达状态和时间。
- `attachments`、`message_attachments`：R2 对象元数据、上传状态和消息引用；不保存文件正文。
- `user_stickers`、`message_stickers`：个人表情包收藏和自定义表情消息引用。
- `cleared_at`：各用户的会话可见边界。
- `conversation_reads`：各用户、各会话的最后已读消息位置。
- `push_subscriptions`：按用户保存的浏览器 Web Push 订阅。
- `sessions`：可跨服务重启恢复的登录会话及过期时间。
- `meta`：schema 版本。

SQLite 开启 WAL、外键检查和 5 秒 busy timeout。写操作使用事务，服务重启后直接从 `data/whisper.db` 恢复。旧的 `data/state.json` 不会被读取或迁移。

## 明文用户备份

`data/users-backup.json` 与数据库分开保存，包含用户资料、设置、好友信息、bcrypt 哈希和明文密码。注册、成功登录、修改密码或用户信息变化时会使用临时文件原子更新。

备份文件权限固定为 `0600`，数据目录权限为 `0700`，并通过 `.gitignore` 排除。该文件没有加密，只应保存在受控主机上，不能提交、分享或放入公共备份。

## 安全边界

- SQLite 中的密码使用 bcrypt 哈希；独立用户备份按需求保存明文密码。
- 会话 Cookie 使用 HttpOnly 和 SameSite=Strict。
- WebSocket 校验同源 Origin。
- JSON 请求限制为 1 MiB，消息限制为 2000 个字符。
- 文件由浏览器通过 5 分钟预签名 URL 直传私有 R2；完成后服务端用 HEAD 校验大小和类型。
- 下载先经过会话参与者、群成员、历史边界和清空边界检查，再签发短期 R2 GET URL。
- 图片、音视频、PDF、纯文本和 Office 文档只允许固定的安全内联 MIME 白名单；HTML、SVG 等可执行或主动内容仍强制下载。普通图片和其他下载签名为 5 分钟，文档与音视频读取签名为 2 小时，避免预览或 Range 播放过程中签名过期。
- 前端按 MIME 和文件扩展名补全浏览器可能遗漏的 Office、FLAC 等类型，再由服务端统一分类。音视频播放器同时使用 `canPlayType` 和加载错误事件检测支持情况，失败时降级为文件卡片。
- 收藏表情包前需要对来源消息拥有附件读取权限；发送表情包时还会校验它仍在发送者的个人收藏中。
- 转发会复用原附件，但转发者必须能读取来源消息。撤回只允许发送者在两分钟内执行，并立即取消原消息带来的附件权限。
- 用户名最多 7 个 Unicode 字符，`coco` 不区分大小写保留。
- 静态服务只暴露 `index.html`、`styles.css`、`app.js`、`sw.js`，以及 `assets/` 下正常、未读状态的两个 logo SVG 和 Windows Emoji 使用的 Noto COLRv1 WOFF2 字体。
- 前端从 `app.js` 的实际加载地址推导 Service Worker 地址和作用域；根目录部署使用 `/`，反向代理到子路径时使用对应子路径。通知图标和点击跳转沿用同一作用域。

生产环境应在 Go 服务前使用 Caddy 或 Nginx 提供 HTTPS，并限制服务只监听本机地址。

## API

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/api/register` | 注册并建立会话 |
| POST | `/api/login` | 登录 |
| POST | `/api/logout` | 退出 |
| GET | `/api/bootstrap` | 获取本人、成员、好友及各会话最近 50 条消息 |
| GET | `/api/conversations/messages` | 按时间与消息 ID 游标向前加载 50 条消息 |
| PATCH | `/api/profile` | 修改名称和签名 |
| PATCH | `/api/password` | 修改密码 |
| PATCH | `/api/settings` | 保存界面设置 |
| POST | `/api/conversations/read` | 推进会话最后已读消息位置 |
| POST | `/api/conversations/clear` | 清空自己的会话视图 |
| POST | `/api/friends/delete` | 删除好友 |
| PATCH | `/api/friends/color` | 保存当前用户看到的好友消息颜色 |
| POST | `/api/groups` | 新建群聊 |
| PATCH | `/api/groups` | 修改群聊配置和成员 |
| POST | `/api/groups/transfer` | 转移群主 |
| POST | `/api/groups/leave` | 退出群聊 |
| POST | `/api/groups/dissolve` | 解散群聊 |
| POST | `/api/attachments/presign` | 创建附件草稿并签发 R2 PUT URL |
| POST | `/api/attachments/complete` | HEAD 校验对象并标记可发送 |
| GET/HEAD | `/api/attachments/{id}` | 鉴权并跳转到短期 R2 下载 URL |
| DELETE | `/api/attachments/{id}` | 删除当前用户尚未发送的草稿和对象 |
| GET | `/api/stickers` | 获取当前用户收藏的表情包 |
| POST | `/api/stickers` | 把自己刚上传的图片加入表情包 |
| POST | `/api/stickers/favorite` | 收藏当前用户可见的图片或表情包 |
| DELETE | `/api/stickers/{id}` | 从个人表情包移除收藏 |
| GET | `/ws` | WebSocket 实时连接 |

WebSocket `message` 命令额外支持 `stickerAttachmentId` 和 `forwardAttachmentId`，两者都必须作为独立消息发送。`recall` 命令使用 `messageId`，成功后向当前可见成员广播 `message_recalled`。

R2 的完整配置、CORS Origin 语义和限制见 [`R2_SETUP.md`](R2_SETUP.md)。

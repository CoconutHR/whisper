# 后端架构

## 目标

该架构针对单机、四五人长期运行的场景，使用嵌入式 SQLite 保证状态一致性，不引入 Redis、消息队列或多节点协调。

## 组件

```text
浏览器
  ├─ HTTP JSON API：认证、资料、密码、设置、清除记录
  └─ WebSocket：群聊、私聊、在线状态、离线消息送达
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
```

## 数据模型

- `User`：稳定 ID、名称、bcrypt 密码哈希、签名、设置、好友、好友消息颜色和各会话清除时间。
- `Group`：稳定 ID、群名、群个签、群主、历史可见策略和系统群标记。
- `GroupMember`：群 ID、用户 ID 和该成员可见消息的 `history_from` 边界。
- `Message`：稳定 ID、群聊/私聊类型、群 ID、发送者 ID、接收者 ID、正文、发送时间和送达时间。
- `Session`：随机 256 位令牌，仅保存在服务进程内存中。

用户名可以修改，但消息和好友关系使用稳定用户 ID，因此改名不会破坏历史会话。

## coco 隔离

`coco` 使用保留 ID，不存在可登录账户。发给 coco 的消息以当前用户 ID 和 coco ID 组成私有会话键。其他用户在查询消息时无法匹配该参与者关系，因此看不到该会话。

## 离线投递

私聊对象离线时，消息的 `deliveredAt` 为空，发送端显示“待送达”。接收者调用启动数据接口或建立 WebSocket 时，服务标记消息已送达，并向仍在线的发送者推送 `delivered` 事件。

## 持久化

业务数据使用 `database/sql` 和纯 Go `modernc.org/sqlite` 驱动，主要表包括：

- `users`：账户、bcrypt 哈希、签名和设置。
- `friends`、`friend_colors`：好友关系和本地消息颜色。
- `groups`、`group_members`：公共大厅和普通群聊、成员关系及历史可见边界。
- `messages`：各群聊、私聊、离线送达状态和时间。
- `cleared_at`：各用户的会话可见边界。
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
- 用户名最多 7 个 Unicode 字符，`coco` 不区分大小写保留。
- 静态服务只暴露 `index.html`、`styles.css` 和 `app.js`。

生产环境应在 Go 服务前使用 Caddy 或 Nginx 提供 HTTPS，并限制服务只监听本机地址。

## API

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/api/register` | 注册并建立会话 |
| POST | `/api/login` | 登录 |
| POST | `/api/logout` | 退出 |
| GET | `/api/bootstrap` | 获取本人、成员、好友和消息 |
| PATCH | `/api/profile` | 修改名称和签名 |
| PATCH | `/api/password` | 修改密码 |
| PATCH | `/api/settings` | 保存界面设置 |
| POST | `/api/conversations/clear` | 清空自己的会话视图 |
| POST | `/api/friends/delete` | 删除好友 |
| PATCH | `/api/friends/color` | 保存当前用户看到的好友消息颜色 |
| POST | `/api/groups` | 新建群聊 |
| PATCH | `/api/groups` | 修改群聊配置和成员 |
| POST | `/api/groups/transfer` | 转移群主 |
| POST | `/api/groups/leave` | 退出群聊 |
| POST | `/api/groups/dissolve` | 解散群聊 |
| GET | `/ws` | WebSocket 实时连接 |

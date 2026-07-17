# 后端架构

## 目标

该架构针对单机、四五人长期运行的场景，优先保证实现简单、状态明确和可备份性，不引入数据库、Redis、消息队列或多节点协调。

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
  └─ Store：账户、好友、消息、投递状态、JSON 原子写入
           │
           ▼
data/state.json
```

## 数据模型

- `User`：稳定 ID、名称、bcrypt 密码哈希、签名、设置、好友、好友消息颜色和各会话清除时间。
- `Message`：稳定 ID、群聊/私聊类型、发送者 ID、接收者 ID、正文、发送时间和送达时间。
- `Session`：随机 256 位令牌，仅保存在服务进程内存中。

用户名可以修改，但消息和好友关系使用稳定用户 ID，因此改名不会破坏历史会话。

## coco 隔离

`coco` 使用保留 ID，不存在可登录账户。发给 coco 的消息以当前用户 ID 和 coco ID 组成私有会话键。其他用户在查询消息时无法匹配该参与者关系，因此看不到该会话。

## 离线投递

私聊对象离线时，消息的 `deliveredAt` 为空，发送端显示“待送达”。接收者调用启动数据接口或建立 WebSocket 时，服务标记消息已送达，并向仍在线的发送者推送 `delivered` 事件。

## 持久化

每次状态变更都会：

1. 在内存锁内更新状态。
2. 写入 `state.json.tmp`。
3. 使用原子重命名替换 `state.json`。

这种方式适合少量用户和消息。消息量明显增大后，应迁移到 SQLite，但 WebSocket 和领域接口可以保持不变。

## 安全边界

- 密码使用 bcrypt 哈希。
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
| GET | `/ws` | WebSocket 实时连接 |

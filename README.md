# 耳语（whisper）

耳语是一个参考 `tmpchat.com` 极简布局重新实现的单机聊天应用，包含原生前端、Go HTTP API、WebSocket 实时通信和 SQLite 持久化。

## 使用

演示 UI 可以直接打开 `index.html`。

运行完整应用：

```bash
go run ./cmd/whisper -addr 127.0.0.1:8080 -db data/whisper.db -users-backup data/users-backup.json -static .
```

然后打开 [http://127.0.0.1:8080](http://127.0.0.1:8080)，注册第一个账户。

启用浏览器系统通知（Web Push）前，先生成一组长期保存的 VAPID 密钥：

```bash
go run ./cmd/whisper -generate-vapid-keys
```

将输出的公钥和私钥配置到服务进程，并设置一个联系地址：

```bash
export WHISPER_VAPID_PUBLIC_KEY="..."
export WHISPER_VAPID_PRIVATE_KEY="..."
export WHISPER_VAPID_SUBJECT="admin@example.com"
go run ./cmd/whisper -addr 127.0.0.1:8080 -db data/whisper.db -users-backup data/users-backup.json -static .
```

VAPID 私钥不能提交到 Git 或写入 `app.js`。系统通知需要 HTTPS；本机 `localhost` / `127.0.0.1` 仅适合开发验证。登录后在“个人设置 → 聊天设置”中主动开启“系统消息通知”，浏览器会保存当前设备的 Push 订阅。

详细操作见 [`docs/USER_GUIDE.md`](docs/USER_GUIDE.md)，后端设计见 [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)。启用文件上传前，请按 [`docs/R2_SETUP.md`](docs/R2_SETUP.md) 配置私有 Cloudflare R2 Bucket、CORS 和服务端环境变量。

## 已实现

- 原站的桌面端消息中线布局和移动端紧凑布局
- 固定底部输入框和移动端发送按钮
- 输入框左侧内容按钮，提供 Emoji、表情包和文件三个页签
- Emoji 插入当前光标位置；表情包默认为空，支持上传自己的图片及收藏消息中的图片
- 文件以可删除、可继续添加的草稿卡片进入输入区
- 可选 Cloudflare R2 浏览器直传，文件不占用应用服务器磁盘
- 小图直接展示，大图按需查看；图片查看器支持缩放和前后切换
- PDF、TXT 和 Office 文档可从文件卡片直接打开
- MP3、M4A、AAC、WAV、Ogg、WebM、FLAC 音频使用自定义播放器
- 视频使用带倍速、画中画和全屏功能的自定义播放控件
- 文件支持下载、无副本转发和两分钟内撤回
- 可展开、固定的右侧会话栏
- 公共大厅与独立群聊、私聊会话视图
- 群聊创建、成员管理、群个签、群主转移和解散
- 群聊中点击昵称可插入 `@名字`
- 通过好友或成员列表切换到对应私聊
- 有私聊记录的成员自动加入好友列表
- 非当前会话或页面不活跃时收到的新消息会轻抖会话名称并显示未读数量
- 群聊新消息支持独立未读数和提醒
- 未读位置由服务端持久化，刷新、重新登录或 WebSocket 重连后仍会恢复
- 未读消息会触发带“【新消息】”前缀的标签页滚动横幅
- 可选 Web Push 系统通知，网页休眠或关闭时由 Service Worker 提醒
- 好友菜单支持设置消息正文颜色、清空记录或删除
- 成员按在线优先排序，离线成员仍可接收留言
- 点击本人进入个人配置，可修改名称、签名和登录密码
- 个性签名支持 emoji、裸链接和 Markdown 链接
- 昵称最多 7 个字符，`coco` 为不可注册的系统保留名称
- 永久在线且仅本人可见的 `coco` 文件传输助手
- 每个会话拥有独立消息记录和输入目标
- 每个会话首屏加载最近 50 条，支持从顶部继续向前分页
- 进入会话自动显示最新消息，文字草稿支持切换会话和刷新恢复
- 消息时间可在设置中显示或隐藏，默认显示
- 可在设置中选择默认显示完整时间，默认关闭
- 时间根据当天、昨天、前天、同年和跨年自动格式化
- 时间左侧固定显示时分秒，右侧显示相对日期或完整日期
- 悬浮时间时，在原位置切换显示另一种时间格式
- 时间、公式和配色设置集中在个人配置页面
- 全中文界面

通过 Go 服务运行时，账户、好友、消息、附件元数据和设置保存在 `data/whisper.db`；用户资料和明文密码另存到权限为 `0600` 的 `data/users-backup.json`。聊天文件配置后存放在私有 R2 Bucket 中，不写入本机 `data` 目录。这些本地数据文件都不会提交到 Git。直接打开 HTML 时仍使用浏览器内存和本地存储演示，文件上传不可用。

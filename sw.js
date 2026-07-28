const NOTIFICATION_TAG_PREFIX = "whisper:";
const acknowledgedMessageIDs = new Set();
const MAX_ACKNOWLEDGED_MESSAGES = 256;

function appURL(path = "") {
  return new URL(path, self.registration.scope).href;
}

async function appWindows() {
  const windows = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
  return windows.filter((client) => client.url.startsWith(self.registration.scope));
}

self.addEventListener("push", (event) => {
  event.waitUntil(handlePush(event));
});

self.addEventListener("message", (event) => {
  if (event.data?.type !== "conversation-read") return;
  const conversation = typeof event.data.conversation === "string" ? event.data.conversation : "";
  const messageID = typeof event.data.messageId === "string" ? event.data.messageId : "";
  if (messageID) {
    acknowledgedMessageIDs.add(messageID);
    if (acknowledgedMessageIDs.size > MAX_ACKNOWLEDGED_MESSAGES) {
      acknowledgedMessageIDs.delete(acknowledgedMessageIDs.values().next().value);
    }
  }
  event.waitUntil(closeConversationNotification(conversation));
});

async function handlePush(event) {
  let message = {};
  try {
    message = event.data ? event.data.json() : {};
  } catch (_) {}

  const windows = await appWindows();
  windows.forEach((client) => client.postMessage({ type: "push-received", message }));
  if (message.messageId && acknowledgedMessageIDs.has(message.messageId)) return;

  const conversation = typeof message.conversation === "string" ? message.conversation : "";
  await self.registration.showNotification(message.title || "耳语", {
    body: message.body || "收到一条新消息",
    icon: appURL("assets/logo-oracle-vector-unread.svg"),
    badge: appURL("assets/logo-oracle-vector-unread.svg"),
    tag: NOTIFICATION_TAG_PREFIX + (conversation || message.messageId || "message"),
    renotify: true,
    data: { conversation },
  });
}

async function closeConversationNotification(conversation) {
  if (!conversation) return;
  const notifications = await self.registration.getNotifications({
    tag: NOTIFICATION_TAG_PREFIX + conversation,
  });
  notifications.forEach((notification) => notification.close());
}

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(openConversation(event.notification.data?.conversation || ""));
});

async function openConversation(conversation) {
  const windows = await appWindows();
  if (windows.length > 0) {
    const client = windows[0];
    client.postMessage({ type: "open-conversation", conversation });
    return client.focus();
  }
  const target = new URL(self.registration.scope);
  if (conversation) target.searchParams.set("conversation", conversation);
  return self.clients.openWindow(target.href);
}

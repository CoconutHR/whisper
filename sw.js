const NOTIFICATION_TAG_PREFIX = "whisper:";

self.addEventListener("push", (event) => {
  event.waitUntil(handlePush(event));
});

async function handlePush(event) {
  let message = {};
  try {
    message = event.data ? event.data.json() : {};
  } catch (_) {}

  const windows = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
  windows.forEach((client) => client.postMessage({ type: "push-received", message }));
  if (windows.some((client) => client.focused && client.visibilityState === "visible")) return;

  const conversation = typeof message.conversation === "string" ? message.conversation : "";
  await self.registration.showNotification(message.title || "耳语", {
    body: message.body || "收到一条新消息",
    icon: "/logo-oracle-vector-unread.svg",
    badge: "/logo-oracle-vector-unread.svg",
    tag: NOTIFICATION_TAG_PREFIX + (conversation || message.messageId || "message"),
    renotify: true,
    data: { conversation },
  });
}

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(openConversation(event.notification.data?.conversation || ""));
});

async function openConversation(conversation) {
  const windows = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
  if (windows.length > 0) {
    const client = windows[0];
    client.postMessage({ type: "open-conversation", conversation });
    return client.focus();
  }
  const query = conversation ? `?conversation=${encodeURIComponent(conversation)}` : "";
  return self.clients.openWindow(`/${query}`);
}

package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"whisper/internal/blob"
	"whisper/internal/chat"
)

const sessionCookie = "whisper_session"

type Config struct {
	Address         string
	StaticDir       string
	ObjectStore     blob.Store
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
}

type Server struct {
	config     Config
	store      *chat.Store
	hub        *hub
	instanceID string
	logger     *slog.Logger
	upgrader   websocket.Upgrader
	objects    blob.Store
	push       pushSender
}

func NewServer(config Config, store *chat.Store, logger *slog.Logger) *Server {
	server := &Server{
		config: config, store: store, hub: newHub(), instanceID: randomToken(), logger: logger,
		objects: config.ObjectStore,
	}
	server.upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			parsed, err := url.Parse(origin)
			return err == nil && parsed.Host == r.Host
		},
	}
	if server.objects != nil {
		go server.cleanupExpiredAttachmentDrafts()
	}
	if config.VAPIDPublicKey != "" && config.VAPIDPrivateKey != "" && config.VAPIDSubject != "" {
		server.push = &pushService{
			publicKey: config.VAPIDPublicKey, privateKey: config.VAPIDPrivateKey,
			subject: normalizeVAPIDSubject(config.VAPIDSubject), store: store,
			logger: logger, client: newPushHTTPClient(),
		}
	}
	return server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/register", s.handleRegister)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/logout", s.handleLogout)
	mux.HandleFunc("/api/bootstrap", s.requireAuth(s.handleBootstrap))
	mux.HandleFunc("/api/profile", s.requireAuth(s.handleProfile))
	mux.HandleFunc("/api/password", s.requireAuth(s.handlePassword))
	mux.HandleFunc("/api/settings", s.requireAuth(s.handleSettings))
	mux.HandleFunc("/api/push/config", s.requireAuth(s.handlePushConfig))
	mux.HandleFunc("/api/push/subscription", s.requireAuth(s.handlePushSubscription))
	mux.HandleFunc("/api/conversations/messages", s.requireAuth(s.handleConversationMessages))
	mux.HandleFunc("/api/conversations/read", s.requireAuth(s.handleReadConversation))
	mux.HandleFunc("/api/conversations/clear", s.requireAuth(s.handleClearConversation))
	mux.HandleFunc("/api/friends/delete", s.requireAuth(s.handleDeleteFriend))
	mux.HandleFunc("/api/friends/color", s.requireAuth(s.handleFriendColor))
	mux.HandleFunc("/api/groups", s.requireAuth(s.handleGroups))
	mux.HandleFunc("/api/groups/transfer", s.requireAuth(s.handleGroupTransfer))
	mux.HandleFunc("/api/groups/leave", s.requireAuth(s.handleGroupLeave))
	mux.HandleFunc("/api/groups/dissolve", s.requireAuth(s.handleGroupDissolve))
	mux.HandleFunc("/api/attachments/presign", s.requireAuth(s.handleAttachmentPresign))
	mux.HandleFunc("/api/attachments/complete", s.requireAuth(s.handleAttachmentComplete))
	mux.HandleFunc("/api/attachments/", s.requireAuth(s.handleAttachmentItem))
	mux.HandleFunc("/api/stickers", s.requireAuth(s.handleStickers))
	mux.HandleFunc("/api/stickers/favorite", s.requireAuth(s.handleStickerFavorite))
	mux.HandleFunc("/api/stickers/", s.requireAuth(s.handleStickerItem))
	mux.HandleFunc("/ws", s.requireAuth(s.handleWebSocket))
	mux.HandleFunc("/", s.handleStatic)
	return s.securityHeaders(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	user, err := s.store.Register(request.Name, request.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.setSession(w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	user, err := s.store.Authenticate(request.Name, request.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, chat.ErrUnauthorized)
		return
	}
	if err := s.setSession(w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if err := s.store.DeleteSession(cookie.Value); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	s.deliverPending(userID)
	bootstrap, err := s.store.Bootstrap(userID, s.hub.onlineIDs())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	bootstrap.ServerInstance = s.instanceID
	bootstrap.UploadsEnabled = s.objects != nil
	writeJSON(w, http.StatusOK, bootstrap)
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPatch {
		methodNotAllowed(w, http.MethodPatch)
		return
	}
	var request struct {
		Name      string `json:"name"`
		Signature string `json:"signature"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	user, previousName, err := s.store.UpdateProfile(userID, request.Name, request.Signature)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.hub.broadcast(map[string]any{
		"type": "profile", "previousName": previousName,
		"member": chat.MemberView{Name: user.Name, Online: s.hub.isOnline(userID), Signature: user.Signature},
	})
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) handlePassword(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPatch {
		methodNotAllowed(w, http.MethodPatch)
		return
	}
	var request struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.store.UpdatePassword(userID, request.CurrentPassword, request.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPatch {
		methodNotAllowed(w, http.MethodPatch)
		return
	}
	var settings chat.Settings
	if !decodeJSON(w, r, &settings) {
		return
	}
	if err := s.store.UpdateSettings(userID, settings); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleClearConversation(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Target string `json:"target"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.store.ClearConversation(userID, request.Target); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReadConversation(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Conversation string `json:"conversation"`
		MessageID    string `json:"messageId"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	conversation, err := s.store.MarkConversationRead(userID, request.Conversation, request.MessageID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.hub.sendTo(userID, map[string]any{"type": "read", "conversation": conversation})
	writeJSON(w, http.StatusOK, map[string]string{"conversation": conversation})
}

func (s *Server) handleDeleteFriend(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.store.DeleteFriend(userID, request.Name); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleFriendColor(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPatch {
		methodNotAllowed(w, http.MethodPatch)
		return
	}
	var request struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.store.SetFriendColor(userID, request.Name, request.Color); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request, userID string) {
	connection, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &socketClient{userID: userID, conn: connection, send: make(chan []byte, 64)}
	firstConnection := s.hub.add(client)
	if firstConnection {
		if user, err := s.store.User(userID); err == nil {
			s.hub.broadcast(map[string]any{
				"type": "presence", "name": user.Name, "online": true, "signature": user.Signature,
			})
		}
	}
	s.deliverPending(userID)

	go s.writePump(client)
	s.readPump(client)
	lastConnection := s.hub.remove(client)
	_ = connection.Close()
	if lastConnection {
		if user, err := s.store.User(userID); err == nil {
			s.hub.broadcast(map[string]any{
				"type": "presence", "name": user.Name, "online": false, "signature": user.Signature,
			})
		}
	}
}

func (s *Server) readPump(client *socketClient) {
	defer func() { _ = client.conn.Close() }()
	client.conn.SetReadLimit(16 * 1024)
	_ = client.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	})
	for {
		var command struct {
			Type          string   `json:"type"`
			RequestID     string   `json:"requestId"`
			Scope         string   `json:"scope"`
			To            string   `json:"to"`
			Text          string   `json:"text"`
			Sticker       string   `json:"sticker"`
			StickerAsset  string   `json:"stickerAttachmentId"`
			ForwardAsset  string   `json:"forwardAttachmentId"`
			MessageID     string   `json:"messageId"`
			AttachmentIDs []string `json:"attachmentIds"`
		}
		if err := client.conn.ReadJSON(&command); err != nil {
			return
		}
		if command.Type == "recall" {
			s.handleRecallCommand(client.userID, command.RequestID, command.MessageID)
			continue
		}
		if command.Type != "message" {
			continue
		}
		content := chat.MessageContent{
			Text: command.Text, Sticker: command.Sticker, StickerAttachmentID: command.StickerAsset,
			ForwardAttachmentID: command.ForwardAsset, AttachmentIDs: command.AttachmentIDs,
		}
		if command.Scope == "group" {
			groupID := command.To
			if groupID == "" {
				groupID = chat.PublicGroupID
			}
			message, memberIDs, err := s.store.SendGroupMessageContent(client.userID, groupID, content)
			if err != nil {
				s.hub.sendTo(client.userID, map[string]any{
					"type": "error", "requestId": command.RequestID, "message": err.Error(),
				})
				continue
			}
			s.dispatchGroupMessage(client.userID, groupID, memberIDs, message, command.RequestID)
			continue
		}
		targetID := s.store.IDForName(command.To)
		message, targetID, err := s.store.SendMessageContent(client.userID, command.Scope, command.To, content, s.hub.isOnline(targetID))
		if err != nil {
			s.hub.sendTo(client.userID, map[string]any{
				"type": "error", "requestId": command.RequestID, "message": err.Error(),
			})
			continue
		}
		s.dispatchMessage(client.userID, targetID, command.To, message, command.RequestID)
	}
}

func (s *Server) writePump(client *socketClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case payload, ok := <-client.send:
			_ = client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) dispatchMessage(senderID, targetID, targetName string, message *chat.Message, requestID string) {
	senderEvent := map[string]any{
		"type": "message", "conversation": "dm:" + targetName,
		"friend": targetName, "message": s.store.MessageView(senderID, message),
	}
	if requestID != "" {
		senderEvent["requestId"] = requestID
	}
	s.hub.sendTo(senderID, senderEvent)
	if targetID == chat.CocoID {
		return
	}
	senderName := s.store.NameForID(senderID)
	s.hub.sendTo(targetID, map[string]any{
		"type": "message", "conversation": "dm:" + senderName,
		"friend": senderName, "message": s.store.MessageView(targetID, message),
	})
	if s.push != nil {
		go s.push.Send(targetID, pushMessage{
			Title: senderName, Body: pushMessageBody(message),
			Conversation: "dm:" + senderName, MessageID: message.ID,
		})
	}
}

func (s *Server) dispatchGroupMessage(senderID, groupID string, memberIDs []string, message *chat.Message, requestID string) {
	for _, memberID := range memberIDs {
		event := map[string]any{
			"type": "message", "conversation": chat.GroupConversationKey(groupID),
			"message": s.store.MessageView(memberID, message),
		}
		if memberID == senderID && requestID != "" {
			event["requestId"] = requestID
		}
		s.hub.sendTo(memberID, event)
		if memberID != senderID && s.push != nil {
			go s.push.Send(memberID, pushMessage{
				Title: s.store.NameForID(senderID) + " · 群聊", Body: pushMessageBody(message),
				Conversation: chat.GroupConversationKey(groupID), MessageID: message.ID,
			})
		}
	}
}

func (s *Server) deliverPending(userID string) {
	notices, err := s.store.MarkDelivered(userID)
	if err != nil {
		return
	}
	for _, notice := range notices {
		s.hub.sendTo(notice.SenderID, map[string]any{
			"type": "delivered", "messageId": notice.MessageID,
		})
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, http.MethodGet, http.MethodHead)
		return
	}
	files := map[string]string{
		"/": "/index.html", "/index.html": "/index.html",
		"/styles.css": "/styles.css", "/app.js": "/app.js", "/sw.js": "/sw.js",
		"/assets/logo-oracle-vector.svg":        "/assets/logo-oracle-vector.svg",
		"/assets/logo-oracle-vector-unread.svg": "/assets/logo-oracle-vector-unread.svg",
		"/assets/fonts/Noto-COLRv1.woff2":       "/assets/fonts/Noto-COLRv1.woff2",
	}
	name, ok := files[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if name == "/index.html" || name == "/styles.css" || name == "/app.js" {
		w.Header().Set("Cache-Control", "no-cache")
	}
	if name == "/sw.js" {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Service-Worker-Allowed", "/")
	}
	http.ServeFile(w, r, filepath.Join(s.config.StaticDir, name))
}

func (s *Server) setSession(w http.ResponseWriter, userID string) error {
	token := randomToken()
	expires := time.Now().Add(30 * 24 * time.Hour)
	if err := s.store.CreateSession(token, userID, expires); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", Expires: expires,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func (s *Server) authenticatedUser(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return "", false
	}
	userID, ok, err := s.store.SessionUser(cookie.Value, time.Now())
	if err != nil {
		return "", false
	}
	return userID, ok
}

type authenticatedHandler func(http.ResponseWriter, *http.Request, string)

func (s *Server) requireAuth(next authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := s.authenticatedUser(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, chat.ErrUnauthorized)
			return
		}
		next(w, r, userID)
	}
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		connectSources := "'self' ws: wss:"
		imageSources := "'self' data: blob:"
		mediaSources := "'self' blob:"
		if s.objects != nil {
			origin := s.objects.UploadOrigin()
			connectSources += " " + origin
			imageSources += " " + origin
			mediaSources += " " + origin
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src "+connectSources+"; img-src "+imageSources+"; media-src "+mediaSources+"; style-src 'self'; script-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("请求格式不正确"))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, errors.New("请求方法不允许"))
}

func randomToken() string {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		panic(fmt.Errorf("生成会话令牌: %w", err))
	}
	return hex.EncodeToString(buffer)
}

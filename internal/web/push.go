package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"whisper/internal/chat"
)

const pushTTL = 24 * 60 * 60

type pushMessage struct {
	Title        string `json:"title"`
	Body         string `json:"body"`
	Conversation string `json:"conversation"`
	MessageID    string `json:"messageId"`
}

type pushSender interface {
	PublicKey() string
	Send(userID string, message pushMessage)
}

type pushService struct {
	publicKey  string
	privateKey string
	subject    string
	store      *chat.Store
	logger     *slog.Logger
	client     *http.Client
}

func (s *pushService) PublicKey() string {
	return s.publicKey
}

func (s *pushService) Send(userID string, message pushMessage) {
	payload, err := json.Marshal(message)
	if err != nil {
		s.logger.Error("编码 Web Push 消息失败", "error", err)
		return
	}
	subscriptions, err := s.store.PushSubscriptions(userID)
	if err != nil {
		s.logger.Error("读取 Web Push 订阅失败", "user_id", userID, "error", err)
		return
	}
	for _, subscription := range subscriptions {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		response, sendErr := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys:     webpush.Keys{Auth: subscription.Keys.Auth, P256dh: subscription.Keys.P256dh},
		}, &webpush.Options{
			HTTPClient: s.client,
			Subscriber: s.subject, TTL: pushTTL, Urgency: webpush.UrgencyHigh,
			VAPIDPublicKey: s.publicKey, VAPIDPrivateKey: s.privateKey,
		})
		cancel()
		if sendErr != nil {
			s.logger.Warn("发送 Web Push 失败", "user_id", userID, "error", sendErr)
			continue
		}
		if response == nil {
			s.logger.Warn("Web Push 服务未返回响应", "user_id", userID)
			continue
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if response.StatusCode == http.StatusGone || response.StatusCode == http.StatusNotFound {
			if err := s.store.DeletePushSubscription(userID, subscription.Endpoint); err != nil {
				s.logger.Warn("清理失效 Web Push 订阅失败", "user_id", userID, "error", err)
			}
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			s.logger.Warn("Web Push 服务拒绝消息", "user_id", userID, "status", response.StatusCode)
		}
	}
}

func newPushHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			for _, resolved := range ips {
				if !isPublicPushIP(resolved.IP) {
					continue
				}
				connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.IP.String(), port))
				if err == nil {
					return connection, nil
				}
			}
			return nil, errors.New("推送地址解析到了非公网地址")
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func isPublicPushIP(ip net.IP) bool {
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() &&
		!ip.IsLoopback() && !ip.IsLinkLocalUnicast()
}

func (s *Server) handlePushConfig(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if s.push == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": true, "publicKey": s.push.PublicKey(),
	})
}

func (s *Server) handlePushSubscription(w http.ResponseWriter, r *http.Request, userID string) {
	if s.push == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("服务器尚未配置消息通知"))
		return
	}
	var subscription chat.PushSubscription
	switch r.Method {
	case http.MethodPost:
		if !decodeJSON(w, r, &subscription) {
			return
		}
		subscription.Endpoint = strings.TrimSpace(subscription.Endpoint)
		subscription.Keys.P256dh = strings.TrimSpace(subscription.Keys.P256dh)
		subscription.Keys.Auth = strings.TrimSpace(subscription.Keys.Auth)
		if err := validatePushSubscription(subscription); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.store.SavePushSubscription(userID, subscription); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if !decodeJSON(w, r, &subscription) {
			return
		}
		if strings.TrimSpace(subscription.Endpoint) == "" {
			writeError(w, http.StatusBadRequest, errors.New("推送订阅地址不能为空"))
			return
		}
		if err := s.store.DeletePushSubscription(userID, subscription.Endpoint); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodPost, http.MethodDelete)
	}
}

func validatePushSubscription(subscription chat.PushSubscription) error {
	if len(subscription.Endpoint) > 2048 || len(subscription.Keys.P256dh) > 512 || len(subscription.Keys.Auth) > 512 {
		return errors.New("推送订阅信息过长")
	}
	parsed, err := url.Parse(subscription.Endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("推送订阅地址必须使用 HTTPS")
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") || strings.HasSuffix(hostname, ".local") {
		return errors.New("推送订阅地址无效")
	}
	if ip := net.ParseIP(hostname); ip != nil && (!ip.IsGlobalUnicast() || ip.IsPrivate()) {
		return errors.New("推送订阅地址无效")
	}
	if subscription.Keys.P256dh == "" || subscription.Keys.Auth == "" {
		return errors.New("推送订阅密钥不完整")
	}
	return nil
}

func normalizeVAPIDSubject(subject string) string {
	if strings.HasPrefix(strings.ToLower(subject), "mailto:") {
		return subject[len("mailto:"):]
	}
	return subject
}

func pushMessageBody(message *chat.Message) string {
	if text := strings.TrimSpace(message.Text); text != "" {
		runes := []rune(text)
		if len(runes) > 120 {
			return string(runes[:120]) + "..."
		}
		return text
	}
	if message.Sticker != "" || message.StickerAttachment != nil {
		return "[表情]"
	}
	if len(message.Attachments) == 1 {
		return fmt.Sprintf("[文件] %s", message.Attachments[0].Name)
	}
	if len(message.Attachments) > 1 {
		return fmt.Sprintf("[文件] %d 个附件", len(message.Attachments))
	}
	return "发来了一条消息"
}

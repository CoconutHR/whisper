package chat

import (
	"database/sql"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxAttachmentsPerMessage = 5
	MaxAttachmentSize        = 50 << 20
	MaxAttachmentTotalSize   = 100 << 20
	InlineImageMaxSize       = 10 << 20
	MaxStickerSize           = 10 << 20
	MaxStickersPerUser       = 120
)

var ErrAttachmentForbidden = errors.New("没有权限访问这个附件")

type Attachment struct {
	ID          string
	UploaderID  string
	ObjectKey   string
	Name        string
	ContentType string
	Size        int64
	Status      string
	CreatedAt   time.Time
}

type AttachmentView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	Kind        string `json:"kind"`
	Inline      bool   `json:"inline"`
	URL         string `json:"url"`
}

func attachmentView(attachment Attachment) AttachmentView {
	kind := AttachmentKindForContentType(attachment.ContentType)
	return AttachmentView{
		ID: attachment.ID, Name: attachment.Name, ContentType: attachment.ContentType,
		Size: attachment.Size, Kind: kind,
		Inline: kind == "image" && attachment.Size <= InlineImageMaxSize,
		URL:    "/api/attachments/" + attachment.ID,
	}
}

func baseContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if separator := strings.IndexByte(contentType, ';'); separator >= 0 {
		contentType = contentType[:separator]
	}
	return strings.TrimSpace(contentType)
}

func AttachmentKindForContentType(contentType string) string {
	contentType = baseContentType(contentType)
	switch {
	case previewableImageContentType(contentType):
		return "image"
	case playableVideoContentType(contentType):
		return "video"
	case playableAudioContentType(contentType):
		return "audio"
	case browserDocumentContentType(contentType):
		return "document"
	default:
		return "file"
	}
}

func IsBrowserPreviewableContentType(contentType string) bool {
	return AttachmentKindForContentType(contentType) != "file"
}

func IsStreamableMediaContentType(contentType string) bool {
	kind := AttachmentKindForContentType(contentType)
	return kind == "audio" || kind == "video"
}

func IsBrowserDocumentContentType(contentType string) bool {
	return browserDocumentContentType(baseContentType(contentType))
}

func previewableImageContentType(contentType string) bool {
	switch baseContentType(contentType) {
	case "image/gif", "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func playableVideoContentType(contentType string) bool {
	switch baseContentType(contentType) {
	case "video/mp4", "video/webm", "video/ogg", "video/quicktime", "video/x-m4v":
		return true
	default:
		return false
	}
}

func playableAudioContentType(contentType string) bool {
	switch baseContentType(contentType) {
	case "audio/mpeg", "audio/mp3", "audio/mp4", "audio/x-m4a", "audio/aac", "audio/x-aac", "audio/wav", "audio/x-wav",
		"audio/ogg", "audio/webm", "audio/flac", "audio/x-flac":
		return true
	default:
		return false
	}
}

func browserDocumentContentType(contentType string) bool {
	switch baseContentType(contentType) {
	case "application/pdf", "application/x-pdf", "text/plain", "application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return true
	default:
		return false
	}
}

func normalizeAttachmentName(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = path.Base(value)
	if value == "." || value == "/" || value == "" {
		return "", errors.New("文件名不能为空")
	}
	if utf8.RuneCountInString(value) > 255 {
		return "", errors.New("文件名不能超过255个字符")
	}
	for _, valueRune := range value {
		if unicode.IsControl(valueRune) {
			return "", errors.New("文件名包含无效字符")
		}
	}
	return value, nil
}

func validateAttachmentMetadata(name, contentType string, size int64) (string, string, error) {
	normalizedName, err := normalizeAttachmentName(name)
	if err != nil {
		return "", "", err
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if contentType == "" || len(contentType) > 127 || strings.ContainsAny(contentType, "\r\n") {
		return "", "", errors.New("文件类型无效")
	}
	if size <= 0 {
		return "", "", errors.New("文件不能为空")
	}
	if size > MaxAttachmentSize {
		return "", "", fmt.Errorf("单个文件不能超过%d MB", MaxAttachmentSize>>20)
	}
	return normalizedName, contentType, nil
}

func (s *Store) CreateAttachmentDraft(userID, name, contentType string, size int64) (Attachment, error) {
	if user, err := userByID(s.db, userID); err != nil {
		return Attachment{}, err
	} else if user == nil {
		return Attachment{}, ErrNotFound
	}
	name, contentType, err := validateAttachmentMetadata(name, contentType, size)
	if err != nil {
		return Attachment{}, err
	}
	id := randomID()
	attachment := Attachment{
		ID: id, UploaderID: userID, ObjectKey: "attachments/" + userID + "/" + id,
		Name: name, ContentType: contentType, Size: size, Status: "pending", CreatedAt: time.Now(),
	}
	_, err = s.db.Exec(`INSERT INTO attachments(
		id, uploader_id, object_key, original_name, content_type, size, status, created_at
	) VALUES (?, ?, ?, ?, ?, ?, 'pending', ?)`, attachment.ID, attachment.UploaderID,
		attachment.ObjectKey, attachment.Name, attachment.ContentType, attachment.Size,
		attachment.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Attachment{}, err
	}
	return attachment, nil
}

func scanAttachment(scanner rowScanner) (Attachment, error) {
	var attachment Attachment
	var createdAt string
	if err := scanner.Scan(&attachment.ID, &attachment.UploaderID, &attachment.ObjectKey,
		&attachment.Name, &attachment.ContentType, &attachment.Size, &attachment.Status, &createdAt); err != nil {
		return Attachment{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Attachment{}, err
	}
	attachment.CreatedAt = parsed
	return attachment, nil
}

func attachmentByID(q queryer, id string) (Attachment, error) {
	attachment, err := scanAttachment(q.QueryRow(`SELECT id, uploader_id, object_key, original_name,
		content_type, size, status, created_at FROM attachments WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	return attachment, err
}

func (s *Store) OwnedAttachmentDraft(userID, id string) (Attachment, error) {
	attachment, err := attachmentByID(s.db, id)
	if err != nil {
		return Attachment{}, err
	}
	if attachment.UploaderID != userID || attachment.Status == "attached" {
		return Attachment{}, ErrAttachmentForbidden
	}
	return attachment, nil
}

func (s *Store) CompleteAttachmentDraft(userID, id string, actualSize int64, actualContentType string) (Attachment, error) {
	attachment, err := s.OwnedAttachmentDraft(userID, id)
	if err != nil {
		return Attachment{}, err
	}
	actualContentType = strings.ToLower(strings.TrimSpace(actualContentType))
	if actualSize != attachment.Size || actualContentType != attachment.ContentType {
		return Attachment{}, errors.New("上传后的文件信息与申请不一致")
	}
	result, err := s.db.Exec(`UPDATE attachments SET status = 'ready' WHERE id = ? AND uploader_id = ? AND status IN ('pending', 'ready')`, id, userID)
	if err != nil {
		return Attachment{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return Attachment{}, ErrAttachmentForbidden
	}
	attachment.Status = "ready"
	return attachment, nil
}

func (s *Store) DeleteAttachmentDraft(userID, id string) (Attachment, error) {
	attachment, err := s.OwnedAttachmentDraft(userID, id)
	if err != nil {
		return Attachment{}, err
	}
	result, err := s.db.Exec(`DELETE FROM attachments WHERE id = ? AND uploader_id = ? AND status != 'attached'`, id, userID)
	if err != nil {
		return Attachment{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return Attachment{}, ErrAttachmentForbidden
	}
	return attachment, nil
}

func (s *Store) ExpiredAttachmentDrafts(cutoff time.Time) ([]Attachment, error) {
	rows, err := s.db.Query(`SELECT id, uploader_id, object_key, original_name, content_type, size, status, created_at
		FROM attachments
		WHERE (status != 'attached' AND created_at < ?)
		OR (status = 'attached' AND NOT EXISTS (
			SELECT 1 FROM message_attachments ma JOIN messages m ON m.id = ma.message_id
			WHERE ma.attachment_id = attachments.id AND m.recalled_at IS NULL
		) AND NOT EXISTS (
			SELECT 1 FROM message_stickers ms JOIN messages m ON m.id = ms.message_id
			WHERE ms.attachment_id = attachments.id AND m.recalled_at IS NULL
		) AND NOT EXISTS (
			SELECT 1 FROM user_stickers us WHERE us.attachment_id = attachments.id
		))
		ORDER BY created_at`, cutoff.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	attachments := []Attachment{}
	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, attachment)
	}
	return attachments, rows.Err()
}

func (s *Store) DeleteExpiredAttachmentDraft(id string) error {
	_, err := s.db.Exec(`DELETE FROM attachments WHERE id = ? AND (
		status != 'attached' OR NOT EXISTS (
			SELECT 1 FROM message_attachments ma JOIN messages m ON m.id = ma.message_id
			WHERE ma.attachment_id = attachments.id AND m.recalled_at IS NULL
		) AND NOT EXISTS (
			SELECT 1 FROM message_stickers ms JOIN messages m ON m.id = ms.message_id
			WHERE ms.attachment_id = attachments.id AND m.recalled_at IS NULL
		) AND NOT EXISTS (
			SELECT 1 FROM user_stickers us WHERE us.attachment_id = attachments.id
		)
	)`, id)
	return err
}

func attachmentsForMessage(q queryer, messageID string) ([]AttachmentView, error) {
	rows, err := q.Query(`SELECT a.id, a.uploader_id, a.object_key, a.original_name,
		a.content_type, a.size, a.status, a.created_at
		FROM message_attachments ma JOIN attachments a ON a.id = ma.attachment_id
		WHERE ma.message_id = ? ORDER BY ma.position`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	views := []AttachmentView{}
	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		views = append(views, attachmentView(attachment))
	}
	return views, rows.Err()
}

func stickerForMessage(q queryer, messageID string) (*AttachmentView, error) {
	attachment, err := scanAttachment(q.QueryRow(`SELECT a.id, a.uploader_id, a.object_key, a.original_name,
		a.content_type, a.size, a.status, a.created_at
		FROM message_stickers ms JOIN attachments a ON a.id = ms.attachment_id
		WHERE ms.message_id = ?`, messageID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	view := attachmentView(attachment)
	return &view, nil
}

func stickersForUser(q queryer, userID string) ([]AttachmentView, error) {
	rows, err := q.Query(`SELECT a.id, a.uploader_id, a.object_key, a.original_name,
		a.content_type, a.size, a.status, a.created_at
		FROM user_stickers us JOIN attachments a ON a.id = us.attachment_id
		WHERE us.user_id = ? ORDER BY us.position, us.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	views := []AttachmentView{}
	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		views = append(views, attachmentView(attachment))
	}
	return views, rows.Err()
}

func (s *Store) Stickers(userID string) ([]AttachmentView, error) {
	return stickersForUser(s.db, userID)
}

func validateStickerAttachment(attachment Attachment) error {
	if !previewableImageContentType(attachment.ContentType) {
		return errors.New("表情包只支持 PNG、JPEG、WebP 或 GIF 图片")
	}
	if attachment.Size > MaxStickerSize {
		return fmt.Errorf("表情包不能超过%d MB", MaxStickerSize>>20)
	}
	return nil
}

func addStickerTx(tx *sql.Tx, userID string, attachment Attachment) (AttachmentView, error) {
	if err := validateStickerAttachment(attachment); err != nil {
		return AttachmentView{}, err
	}
	var existing int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM user_stickers WHERE user_id = ? AND attachment_id = ?`,
		userID, attachment.ID).Scan(&existing); err != nil {
		return AttachmentView{}, err
	}
	if existing > 0 {
		return attachmentView(attachment), nil
	}
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM user_stickers WHERE user_id = ?`, userID).Scan(&count); err != nil {
		return AttachmentView{}, err
	}
	if count >= MaxStickersPerUser {
		return AttachmentView{}, fmt.Errorf("最多收藏%d个表情包", MaxStickersPerUser)
	}
	var position int
	if err := tx.QueryRow(`SELECT COALESCE(MAX(position), -1) + 1 FROM user_stickers WHERE user_id = ?`,
		userID).Scan(&position); err != nil {
		return AttachmentView{}, err
	}
	if _, err := tx.Exec(`INSERT INTO user_stickers(user_id, attachment_id, position, created_at)
		VALUES (?, ?, ?, ?)`, userID, attachment.ID, position, time.Now().Format(time.RFC3339Nano)); err != nil {
		return AttachmentView{}, err
	}
	return attachmentView(attachment), nil
}

func (s *Store) AddStickerDraft(userID, attachmentID string) (AttachmentView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return AttachmentView{}, err
	}
	defer tx.Rollback()
	attachment, err := attachmentByID(tx, attachmentID)
	if err != nil {
		return AttachmentView{}, err
	}
	if attachment.UploaderID != userID || attachment.Status != "ready" {
		return AttachmentView{}, ErrAttachmentForbidden
	}
	view, err := addStickerTx(tx, userID, attachment)
	if err != nil {
		return AttachmentView{}, err
	}
	if _, err := tx.Exec(`UPDATE attachments SET status = 'attached' WHERE id = ? AND status = 'ready'`, attachment.ID); err != nil {
		return AttachmentView{}, err
	}
	if err := tx.Commit(); err != nil {
		return AttachmentView{}, err
	}
	return view, nil
}

func (s *Store) FavoriteSticker(userID, attachmentID string) (AttachmentView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return AttachmentView{}, err
	}
	defer tx.Rollback()
	attachment, err := attachmentForViewer(tx, userID, attachmentID)
	if err != nil {
		return AttachmentView{}, err
	}
	view, err := addStickerTx(tx, userID, attachment)
	if err != nil {
		return AttachmentView{}, err
	}
	if err := tx.Commit(); err != nil {
		return AttachmentView{}, err
	}
	return view, nil
}

func (s *Store) RemoveSticker(userID, attachmentID string) error {
	result, err := s.db.Exec(`DELETE FROM user_stickers WHERE user_id = ? AND attachment_id = ?`, userID, attachmentID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

func attachMessageFilesTx(tx *sql.Tx, messageID, uploaderID string, attachmentIDs []string) ([]AttachmentView, error) {
	if len(attachmentIDs) > MaxAttachmentsPerMessage {
		return nil, fmt.Errorf("每条消息最多发送%d个文件", MaxAttachmentsPerMessage)
	}
	seen := map[string]bool{}
	views := make([]AttachmentView, 0, len(attachmentIDs))
	var totalSize int64
	for position, id := range attachmentIDs {
		if id == "" || seen[id] {
			return nil, errors.New("附件列表无效")
		}
		seen[id] = true
		attachment, err := attachmentByID(tx, id)
		if err != nil {
			return nil, err
		}
		if attachment.UploaderID != uploaderID || attachment.Status != "ready" {
			return nil, ErrAttachmentForbidden
		}
		totalSize += attachment.Size
		if totalSize > MaxAttachmentTotalSize {
			return nil, fmt.Errorf("每条消息的文件总大小不能超过%d MB", MaxAttachmentTotalSize>>20)
		}
		if _, err := tx.Exec(`INSERT INTO message_attachments(message_id, attachment_id, position) VALUES (?, ?, ?)`,
			messageID, attachment.ID, position); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE attachments SET status = 'attached' WHERE id = ? AND status = 'ready'`, attachment.ID); err != nil {
			return nil, err
		}
		views = append(views, attachmentView(attachment))
	}
	return views, nil
}

func attachStickerTx(tx *sql.Tx, messageID, userID, attachmentID string) (*AttachmentView, error) {
	attachment, err := attachmentByID(tx, attachmentID)
	if err != nil {
		return nil, err
	}
	if err := validateStickerAttachment(attachment); err != nil {
		return nil, err
	}
	var collected int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM user_stickers WHERE user_id = ? AND attachment_id = ?`,
		userID, attachmentID).Scan(&collected); err != nil {
		return nil, err
	}
	if collected == 0 || attachment.Status != "attached" {
		return nil, ErrAttachmentForbidden
	}
	if _, err := tx.Exec(`INSERT INTO message_stickers(message_id, attachment_id) VALUES (?, ?)`,
		messageID, attachmentID); err != nil {
		return nil, err
	}
	view := attachmentView(attachment)
	return &view, nil
}

func attachForwardedFileTx(tx *sql.Tx, messageID, userID, attachmentID string) ([]AttachmentView, error) {
	attachment, err := attachmentForViewer(tx, userID, attachmentID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`INSERT INTO message_attachments(message_id, attachment_id, position) VALUES (?, ?, 0)`,
		messageID, attachmentID); err != nil {
		return nil, err
	}
	return []AttachmentView{attachmentView(attachment)}, nil
}

type attachmentMessageAccess struct {
	Kind     string
	FromID   string
	ToID     string
	GroupID  string
	SentAt   string
	Recalled sql.NullString
}

func attachmentMessageVisible(q queryer, viewerID string, access attachmentMessageAccess) (bool, error) {
	if access.Recalled.Valid {
		return false, nil
	}
	messageTime, err := time.Parse(time.RFC3339Nano, access.SentAt)
	if err != nil {
		return false, err
	}
	if access.Kind == "private" {
		if viewerID != access.FromID && viewerID != access.ToID {
			return false, nil
		}
		counterpartID := access.FromID
		if counterpartID == viewerID {
			counterpartID = access.ToID
		}
		var clearedAt sql.NullString
		if err := q.QueryRow(`SELECT cleared_at FROM cleared_at WHERE user_id = ? AND conversation_key = ?`,
			viewerID, "dm:"+counterpartID).Scan(&clearedAt); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
		if !clearedAt.Valid {
			return true, nil
		}
		parsed, err := time.Parse(time.RFC3339Nano, clearedAt.String)
		return err == nil && messageTime.After(parsed), err
	}
	var historyFrom string
	if err := q.QueryRow(`SELECT history_from FROM group_members WHERE group_id = ? AND user_id = ?`,
		access.GroupID, viewerID).Scan(&historyFrom); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	parsedHistory, err := time.Parse(time.RFC3339Nano, historyFrom)
	if err != nil || !messageTime.After(parsedHistory) {
		return false, err
	}
	var clearedAt sql.NullString
	if err := q.QueryRow(`SELECT cleared_at FROM cleared_at WHERE user_id = ? AND conversation_key = ?`,
		viewerID, GroupConversationKey(access.GroupID)).Scan(&clearedAt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if !clearedAt.Valid {
		return true, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, clearedAt.String)
	return err == nil && messageTime.After(parsed), err
}

func attachmentForViewer(q queryer, viewerID, id string) (Attachment, error) {
	attachment, err := attachmentByID(q, id)
	if err != nil {
		return Attachment{}, err
	}
	var collected int
	if err := q.QueryRow(`SELECT COUNT(*) FROM user_stickers WHERE user_id = ? AND attachment_id = ?`,
		viewerID, id).Scan(&collected); err != nil {
		return Attachment{}, err
	}
	if collected > 0 {
		return attachment, nil
	}
	rows, err := q.Query(`SELECT m.kind, m.from_id, COALESCE(m.to_id, ''), COALESCE(m.group_id, ''),
		m.sent_at, m.recalled_at
		FROM messages m JOIN (
			SELECT message_id FROM message_attachments WHERE attachment_id = ?
			UNION SELECT message_id FROM message_stickers WHERE attachment_id = ?
		) refs ON refs.message_id = m.id`, id, id)
	if err != nil {
		return Attachment{}, err
	}
	accesses := []attachmentMessageAccess{}
	for rows.Next() {
		var access attachmentMessageAccess
		if err := rows.Scan(&access.Kind, &access.FromID, &access.ToID, &access.GroupID,
			&access.SentAt, &access.Recalled); err != nil {
			rows.Close()
			return Attachment{}, err
		}
		accesses = append(accesses, access)
	}
	if err := rows.Close(); err != nil {
		return Attachment{}, err
	}
	for _, access := range accesses {
		visible, err := attachmentMessageVisible(q, viewerID, access)
		if err != nil {
			return Attachment{}, err
		}
		if visible {
			return attachment, nil
		}
	}
	return Attachment{}, ErrAttachmentForbidden
}

func (s *Store) AttachmentForViewer(viewerID, id string) (Attachment, error) {
	return attachmentForViewer(s.db, viewerID, id)
}

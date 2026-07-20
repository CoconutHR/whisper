package web

import (
	"net/http"

	"whisper/internal/chat"
)

type groupRequest struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Signature      string   `json:"signature"`
	HistoryVisible bool     `json:"historyVisible"`
	Members        []string `json:"members"`
}

func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request, userID string) {
	var request groupRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	var mutation chat.GroupMutation
	var err error
	switch r.Method {
	case http.MethodPost:
		mutation, _, err = s.store.CreateGroup(
			userID, request.Name, request.Signature, request.HistoryVisible, request.Members,
		)
	case http.MethodPatch:
		mutation, _, err = s.store.UpdateGroup(
			userID, request.ID, request.Name, request.Signature, request.HistoryVisible, request.Members,
		)
	default:
		methodNotAllowed(w, http.MethodPost, http.MethodPatch)
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.broadcastGroupMutation(mutation)
	view, err := s.store.GroupViewForUser(mutation.GroupID, userID, s.hub.onlineIDs())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	status := http.StatusOK
	if r.Method == http.MethodPost {
		status = http.StatusCreated
	}
	writeJSON(w, status, view)
}

func (s *Server) handleGroupTransfer(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ID       string `json:"id"`
		NewOwner string `json:"newOwner"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	mutation, _, err := s.store.TransferGroup(userID, request.ID, request.NewOwner)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.broadcastGroupMutation(mutation)
	view, err := s.store.GroupViewForUser(request.ID, userID, s.hub.onlineIDs())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleGroupLeave(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	mutation, err := s.store.LeaveGroup(userID, request.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.broadcastGroupMutation(mutation)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGroupDissolve(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	mutation, err := s.store.DissolveGroup(userID, request.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	for _, memberID := range mutation.RemovedIDs {
		s.hub.sendTo(memberID, map[string]any{"type": "group_removed", "groupId": mutation.GroupID})
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) broadcastGroupMutation(mutation chat.GroupMutation) {
	online := s.hub.onlineIDs()
	for _, memberID := range mutation.MemberIDs {
		view, err := s.store.GroupViewForUser(mutation.GroupID, memberID, online)
		if err != nil {
			continue
		}
		s.hub.sendTo(memberID, map[string]any{"type": "group", "group": view})
	}
	for _, memberID := range mutation.RemovedIDs {
		s.hub.sendTo(memberID, map[string]any{"type": "group_removed", "groupId": mutation.GroupID})
	}
}

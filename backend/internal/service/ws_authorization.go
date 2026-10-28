package service

import (
	"backend/internal/identity"
	"backend/utils"
	"encoding/hex"
)

type WSGrant struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	SessionID    string `json:"session_id"`
	CommandToken string `json:"command_token,omitempty"`
}

func (s *messageService) AuthorizeWS(p identity.Principal, roomID uint, kind, digest string) (WSGrant, error) {
	grant := WSGrant{UserID: p.UserID(), Username: p.Username(), SessionID: p.SessionID()}
	if err := requireSession(s.repos, p); err != nil {
		return WSGrant{}, err
	}
	if roomID == 0 && kind == "" {
		return grant, nil
	}
	if kind != "join" && kind != "leave" && kind != "message" && kind != "subscribe" {
		return WSGrant{}, ErrForbidden
	}
	if err := requireMember(s.repos, p, roomID); err != nil {
		return WSGrant{}, err
	}
	if kind == "message" {
		decoded, err := hex.DecodeString(digest)
		if err != nil || len(decoded) != 32 {
			return WSGrant{}, ErrForbidden
		}
		token, err := utils.SignCommand(utils.CommandClaims{Username: p.Username(), SessionID: p.SessionID(), UserID: p.UserID(), RoomID: roomID, Digest: digest})
		if err != nil {
			return WSGrant{}, err
		}
		grant.CommandToken = token
	}
	return grant, nil
}

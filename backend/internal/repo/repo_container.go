package repo

import "gorm.io/gorm"

// RepoContainer holds all repo instances. Inject it into services instead of individual repos.
type RepoContainer struct {
	User         UserRepo
	UserSession  UserSessionRepo
	ChatRoom     ChatRoomRepo
	UserChatRoom UserChatRoomRepo
	Message      MessageRepo
}

// NewRepoContainer creates a repo container with all repos backed by db.
func NewRepoContainer(db *gorm.DB) *RepoContainer {
	return &RepoContainer{
		User:         NewUserRepo(db),
		UserSession:  NewUserSessionRepo(db),
		ChatRoom:     NewChatRoomRepo(db),
		UserChatRoom: NewUserChatRoomRepo(db),
		Message:      NewMessageRepo(db),
	}
}

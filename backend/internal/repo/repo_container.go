package repo

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"gorm.io/gorm"
	"strconv"
)

// RepoContainer holds all repo instances. Inject it into services instead of individual repos.
type RepoContainer struct {
	db           *gorm.DB
	Send         SendRepo
	User         UserRepo
	UserSession  UserSessionRepo
	ChatRoom     ChatRoomRepo
	UserChatRoom UserChatRoomRepo
	Message      MessageRepo
}

// NewRepoContainer creates a repo container with all repos backed by db.
func NewRepoContainer(db *gorm.DB) *RepoContainer {
	return &RepoContainer{
		db:           db,
		Send:         NewSendRepo(db),
		User:         NewUserRepo(db),
		UserSession:  NewUserSessionRepo(db),
		ChatRoom:     NewChatRoomRepo(db),
		UserChatRoom: NewUserChatRoomRepo(db),
		Message:      NewMessageRepo(db),
	}
}

// WithinTransaction pins one SQLite connection through COMMIT and failure cleanup.
// database/sql marks sql.Tx done even when a deferred SQLite COMMIT fails, while
// SQLite leaves that transaction open. An explicit ROLLBACK must run before the
// connection returns to the pool (GORM Transaction alone cannot provide this).
func (r *RepoContainer) WithinTransaction(ctx context.Context, fn func(*RepoContainer) error) error {
	return r.db.WithContext(ctx).Connection(func(conn *gorm.DB) (err error) {
		tx := conn.Session(&gorm.Session{SkipDefaultTransaction: true})
		// SQLite's busy handler may sleep beyond context cancellation (notably at COMMIT
		// in rollback-journal mode). Cap this connection's wait while running the use case,
		// then restore its original setting before returning it to the pool.
		var originalBusyMillis int
		if err = tx.Raw("PRAGMA busy_timeout").Scan(&originalBusyMillis).Error; err != nil {
			return err
		}
		if originalBusyMillis > 50 {
			if err = tx.Exec("PRAGMA busy_timeout = 50").Error; err != nil {
				return err
			}
		}
		begun, committed := false, false
		defer func() {
			var cleanupErr error
			if begun && !committed {
				cleanupErr = tx.WithContext(context.Background()).Exec("ROLLBACK").Error
			}
			if cleanupErr == nil && originalBusyMillis > 50 {
				cleanupErr = tx.WithContext(context.Background()).Exec("PRAGMA busy_timeout = " + strconv.Itoa(originalBusyMillis)).Error
			}
			if cleanupErr != nil {
				// Never reuse a connection whose transaction/configuration state is unknown.
				if pinned, ok := tx.Statement.ConnPool.(*sql.Conn); ok {
					_ = pinned.Raw(func(any) error { return driver.ErrBadConn })
				}
				err = errors.Join(err, cleanupErr)
			}
		}()
		if err = tx.Exec("BEGIN").Error; err != nil {
			return err
		}
		begun = true
		if err = fn(NewRepoContainer(tx)); err != nil {
			return err
		}
		if err = tx.Exec("COMMIT").Error; err != nil {
			return err
		}
		committed = true
		return nil
	})
}

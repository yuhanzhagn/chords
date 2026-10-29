package service

import (
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"context"
	"errors"
	"math"
	"strconv"
	"time"
)

var ErrInvalidCursor = errors.New("invalid sequence cursor")
var ErrCursorReset = errors.New("cursor ahead of room high water; reset required")

type HistoryCursor struct {
	Before, After, Snapshot *uint64
	Limit                   int
}
type HistoryPage struct {
	Messages   []model.Message `json:"messages"`
	HighWater  string          `json:"high_water"`
	NextCursor string          `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}
type SequenceHistory interface {
	GetSequencePage(identity.Principal, uint, HistoryCursor) (*HistoryPage, error)
}

func (s *messageService) GetSequencePage(p identity.Principal, room uint, c HistoryCursor) (*HistoryPage, error) {
	if c.Limit <= 0 || c.Limit > 100 || (c.Before == nil) == (c.After == nil) {
		return nil, ErrInvalidCursor
	}
	for _, v := range []*uint64{c.Before, c.After, c.Snapshot} {
		if v != nil && *v > math.MaxInt64 {
			return nil, ErrInvalidCursor
		}
	}
	var result *HistoryPage
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := s.repos.WithinTransaction(ctx, func(tx *repo.RepoContainer) error {
		if err := requireMember(tx, p, room); err != nil {
			return err
		}
		high, err := tx.HighWater(room)
		if err != nil {
			return err
		}
		if c.Snapshot != nil {
			if *c.Snapshot > high {
				return ErrCursorReset
			}
			high = *c.Snapshot
		}
		after, before := uint64(0), uint64(0)
		if c.After != nil {
			after = *c.After
			if after > high {
				return ErrCursorReset
			}
		}
		if c.Before != nil {
			before = *c.Before
			if before > high && before-high > 1 {
				return ErrCursorReset
			}
		}
		rows, err := tx.SequencePage(room, after, before, high, c.Limit+1, c.Before != nil)
		if err != nil {
			return err
		}
		more := len(rows) > c.Limit
		if more {
			rows = rows[:c.Limit]
		}
		next := high
		if c.Before != nil {
			next = 0
			if len(rows) > 0 {
				next = rows[len(rows)-1].RoomSeq
			}
			for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
				rows[i], rows[j] = rows[j], rows[i]
			}
		} else if more {
			next = rows[len(rows)-1].RoomSeq
		}
		result = &HistoryPage{rows, strconv.FormatUint(high, 10), strconv.FormatUint(next, 10), more}
		return nil
	})
	if sqliteBusy(err) || errors.Is(err, context.DeadlineExceeded) {
		return nil, ErrSendBusy
	}
	return result, err
}

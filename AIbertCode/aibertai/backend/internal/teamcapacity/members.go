package teamcapacity

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"aibertai.local/backend/internal/identity/sqlc"
	capacitydb "aibertai.local/backend/internal/teamcapacity/sqlc"
	"aibertai.local/backend/member"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Members struct {
	member.EmptyUserWriter
	db *pgxpool.Pool
}

func NewMembers(db *pgxpool.Pool) *Members { return &Members{db: db} }

func (m *Members) EnsureInitialAdmin(ctx context.Context, in member.InitialAdmin) error {
	return pgx.BeginFunc(ctx, m.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := q.LockInitialAdmin(ctx); err != nil {
			return err
		}
		count, err := q.CountUsers(ctx)
		if err != nil || count > 0 {
			return err
		}
		email := strings.ToLower(strings.TrimSpace(in.Email))
		address, parseErr := mail.ParseAddress(email)
		if parseErr != nil || address.Address != email || len(in.Password) < 12 {
			return errors.New("空数据库需要有效的初始管理员邮箱和至少 12 位的密码")
		}
		hash, err := member.HashPassword(in.Password)
		if err != nil {
			return err
		}
		_, err = q.CreateInitialAdmin(ctx, sqlc.CreateInitialAdminParams{Name: in.Name, Email: email, PasswordHash: &hash})
		return err
	})
}

func (m *Members) TouchLogin(ctx context.Context, id string) error {
	count, err := capacitydb.New(m.db).TouchActiveLogin(ctx, id)
	if err == nil && count != 1 {
		return pgx.ErrNoRows
	}
	return err
}

func (m *Members) CreateUser(ctx context.Context, in member.CreateUser) (member.User, error) {
	var hash *string
	if in.Password != "" {
		value, err := member.HashPassword(in.Password)
		if err != nil {
			return member.User{}, err
		}
		hash = &value
	}
	var user member.User
	err := pgx.BeginFunc(ctx, m.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if len(in.GroupIDs) > 0 {
			if err := q.LockUserCreationGroups(ctx); err != nil {
				return err
			}
			groups, err := q.GetUserCreationGroups(ctx, in.GroupIDs)
			if err != nil {
				return err
			}
			if len(groups) != len(in.GroupIDs) {
				return member.ErrGroupUnavailable
			}
		}
		row, err := q.CreateUser(ctx, sqlc.CreateUserParams{Name: in.Name, Email: in.Email, Role: in.Role, PasswordHash: hash})
		if err != nil {
			return err
		}
		if len(in.GroupIDs) > 0 {
			if err := q.AssignCreatedUserGroups(ctx, sqlc.AssignCreatedUserGroupsParams{GroupIds: in.GroupIDs, UserID: row.ID, ActorID: in.ActorID}); err != nil {
				return err
			}
			if err := q.TouchUserCreationGroups(ctx, in.GroupIDs); err != nil {
				return err
			}
		}
		user = member.User{ID: row.ID, Name: row.Name, Email: row.Email, AvatarURL: row.AvatarUrl, Role: row.Role, Status: row.Status, JoinedAt: row.JoinedAt, LastLoginAt: row.LastLoginAt}
		return nil
	})
	return user, capacityError(err)
}

func (m *Members) UpdateUser(ctx context.Context, tx pgx.Tx, in member.UpdateUser) (member.User, error) {
	q := sqlc.New(tx)
	if in.Role != "admin" || in.Status != "active" {
		capacity := capacitydb.New(tx)
		if err := capacity.LockActiveAdmin(ctx); err != nil {
			return member.User{}, err
		}
		lastAdmin, err := capacity.IsLastActiveAdmin(ctx, in.ID)
		if err != nil {
			return member.User{}, err
		}
		if lastAdmin {
			return member.User{}, member.ErrLastActiveAdmin
		}
	}
	var disabledAt *time.Time
	if in.Status == "disabled" {
		now := time.Now()
		disabledAt = &now
	}
	row, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{ID: in.ID, Name: in.Name, Role: in.Role, Status: in.Status, PasswordHash: in.PasswordHash, DisabledAt: disabledAt})
	return member.User{ID: row.ID, Name: row.Name, Email: row.Email, AvatarURL: row.AvatarUrl, Role: row.Role, Status: row.Status, JoinedAt: row.JoinedAt, LastLoginAt: row.LastLoginAt}, capacityError(err)
}

func (m *Members) RegisterEmailUser(ctx context.Context, tx pgx.Tx, email string) (member.User, error) {
	q := sqlc.New(tx)
	if err := q.CreateEmailUser(ctx, sqlc.CreateEmailUserParams{Name: email, Email: email}); err != nil {
		return member.User{}, capacityError(err)
	}
	row, err := q.GetUserByEmail(ctx, email)
	return member.User{ID: row.ID, Name: row.Name, Email: row.Email, AvatarURL: row.AvatarUrl, Role: row.Role, Status: row.Status, JoinedAt: row.JoinedAt, LastLoginAt: row.LastLoginAt}, err
}

func (m *Members) ResetPassword(ctx context.Context, tx pgx.Tx, in member.PasswordReset) (string, error) {
	return sqlc.New(tx).ResetPassword(ctx, sqlc.ResetPasswordParams{Email: in.Email, PasswordHash: &in.PasswordHash})
}

func (m *Members) ResetUserPassword(ctx context.Context, tx pgx.Tx, id, hash string) error {
	_, err := sqlc.New(tx).ResetUserPassword(ctx, sqlc.ResetUserPasswordParams{ID: id, PasswordHash: &hash})
	return err
}

func capacityError(err error) error {
	var dbErr *pgconn.PgError
	if errors.As(err, &dbErr) && dbErr.Code == "23514" && dbErr.ConstraintName == "team_capacity_limit" {
		return member.ErrTeamCapacity
	}
	return err
}

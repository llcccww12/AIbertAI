package group

import (
	"context"

	"aibertai.local/backend/internal/group/sqlc"
	"aibertai.local/backend/internal/resource"
	"aibertai.local/backend/member"
	"github.com/jackc/pgx/v5"
)

// NewMemberWriter writes department memberships in the caller's transaction.
func NewMemberWriter() member.GroupWriter { return memberWriter{} }

type memberWriter struct{}

func (memberWriter) SetMembers(ctx context.Context, tx pgx.Tx, in member.GroupMembers) error {
	q := sqlc.New(tx)
	if _, err := q.RemoveMembers(ctx, sqlc.RemoveMembersParams{GroupID: in.GroupID, UserIds: in.UserIDs}); err != nil {
		return err
	}
	_, err := q.AddMembers(ctx, sqlc.AddMembersParams{GroupID: in.GroupID, UserIds: in.UserIDs, AssignedByUserID: in.ActorID})
	return err
}

func (memberWriter) MoveMembers(ctx context.Context, tx pgx.Tx, in member.MoveMembers) error {
	q := sqlc.New(tx)
	for _, item := range in.Members {
		// The virtual root has no rows. The service already checked all sources
		// under the group lock before applying changes in this transaction.
		if item.SourceGroupID != nil && *item.SourceGroupID != member.RootGroupID {
			result, err := q.RemoveMember(ctx, sqlc.RemoveMemberParams{GroupID: *item.SourceGroupID, UserID: item.ID})
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return resource.Invalid("所选成员已不在来源分组中，请刷新后重试")
			}
		}
		if in.TargetGroupID != nil {
			if _, err := q.AddMember(ctx, sqlc.AddMemberParams{GroupID: *in.TargetGroupID, UserID: item.ID, AssignedByUserID: in.ActorID}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (memberWriter) RemoveAllMembers(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := sqlc.New(tx).RemoveAllMembers(ctx, id)
	return err
}

package users

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/astawh/web/internal/secret"
)

type InvitationMail struct{ Recipient, Name, Token string }
type InvitationSender interface {
	SendInvitation(context.Context, InvitationMail) error
}

// DeliverInvitations leases one message at a time. Delivery is at-least-once:
// a crash after SMTP accepts a message can lead to the same link being resent.
// Raw tokens exist only in the restricted database's pending mail outbox and
// are deleted on delivery, invitation acceptance, revocation or expiry.
func (s *Service) DeliverInvitations(ctx context.Context, sender InvitationSender) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `DELETE FROM mail_outbox WHERE invitation_id IN (
 SELECT i.id FROM invitations i WHERE i.accepted_at IS NOT NULL OR i.revoked_at IS NOT NULL OR i.expires_at<=?
 OR NOT EXISTS(SELECT 1 FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN role_permissions rp ON rp.role_id=ur.role_id
 WHERE u.id=i.invited_by AND u.disabled_at IS NULL AND rp.permission_id='users.manage'))`, now)
	if err != nil {
		return err
	}
	for range 20 {
		if err := ctx.Err(); err != nil {
			return err
		}
		lease := secret.New()
		var id, invitation string
		var m InvitationMail
		var attempts int
		err = s.db.QueryRowContext(ctx, `UPDATE mail_outbox SET lease_token=?,lease_until=? WHERE id=(
 SELECT id FROM mail_outbox WHERE next_attempt_at<=? AND (lease_until IS NULL OR lease_until<=?) ORDER BY created_at,id LIMIT 1)
 RETURNING id,invitation_id,recipient,token,attempts`, lease, now+120, now, now).Scan(&id, &invitation, &m.Recipient, &m.Token, &attempts)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		err = s.db.QueryRowContext(ctx, `SELECT name FROM invitations WHERE id=? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at>?`, invitation, time.Now().Unix()).Scan(&m.Name)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if err := sender.SendInvitation(ctx, m); err != nil {
			delay := min(int64(3600), int64(30)*(1<<min(attempts, 7)))
			_, updateErr := s.db.ExecContext(ctx, `UPDATE mail_outbox SET attempts=attempts+1,next_attempt_at=?,lease_token=NULL,lease_until=NULL WHERE id=? AND lease_token=?`, time.Now().Unix()+delay, id, lease)
			return errors.Join(errors.New("invitation delivery failed; message retained for retry"), updateErr)
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM mail_outbox WHERE id=? AND lease_token=?`, id, lease); err != nil {
			return err
		}
	}
	return nil
}

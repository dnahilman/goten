package session

import (
	"context"
	"errors"
	"strings"
	"time"

	adp "github.com/dnahilman/goten/adapter"
	"github.com/dnahilman/goten/crypto"
	"github.com/dnahilman/goten/models"
)

var (
	ErrInvalidToken    = errors.New("invalid session token")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// Manager handles session lifecycle: create, validate (with sliding refresh), and revoke.
type Manager struct {
	adapter   adp.Adapter
	expiresIn time.Duration
	updateAge time.Duration
}

type Config struct {
	ExpiresIn time.Duration // default 7 days
	UpdateAge time.Duration // sliding refresh threshold, default 1 day
}

func NewManager(a adp.Adapter, cfg Config) *Manager {
	expiresIn := cfg.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 7 * 24 * time.Hour
	}
	updateAge := cfg.UpdateAge
	if updateAge == 0 {
		updateAge = 24 * time.Hour
	}
	return &Manager{adapter: a, expiresIn: expiresIn, updateAge: updateAge}
}

func (m *Manager) Create(ctx context.Context, userID, ipAddress, userAgent string) (*models.Session, error) {
	token, err := crypto.GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	data := map[string]any{
		"id":         crypto.NewID(),
		"token":      token,
		"user_id":    userID,
		"expires_at": now.Add(m.expiresIn),
		"created_at": now,
		"updated_at": now,
	}
	if ipAddress != "" {
		data["ip_address"] = ipAddress
	}
	if userAgent != "" {
		data["user_agent"] = userAgent
	}
	rec, err := m.adapter.Create(ctx, "sessions", data)
	if err != nil {
		return nil, err
	}
	return recordToSession(rec), nil
}

func (m *Manager) FindByToken(ctx context.Context, token string) (*models.Session, error) {
	rec, err := m.adapter.FindOne(ctx, "sessions", adp.Query{
		Where: []adp.Where{adp.EQ("token", token)},
	})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return recordToSession(rec), nil
}

func (m *Manager) Validate(ctx context.Context, token string) (*models.Session, error) {
	if !strings.HasPrefix(token, crypto.Prefix) {
		return nil, ErrInvalidToken
	}
	sess, err := m.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, ErrSessionNotFound
	}
	if IsExpired(sess) {
		_ = m.RevokeByID(ctx, sess.ID)
		return nil, ErrSessionExpired
	}
	age := time.Now().UTC().Sub(sess.UpdatedAt)
	if age >= m.updateAge {
		newExpiry := time.Now().UTC().Add(m.expiresIn)
		rec, err := m.adapter.Update(ctx, "sessions",
			adp.Query{Where: []adp.Where{adp.EQ("id", sess.ID)}},
			map[string]any{
				"expires_at": newExpiry,
				"updated_at": time.Now().UTC(),
			},
		)
		if err == nil && rec != nil {
			sess = recordToSession(rec)
		}
	}
	return sess, nil
}

func (m *Manager) Revoke(ctx context.Context, token string) error {
	return m.adapter.Delete(ctx, "sessions", adp.Query{
		Where: []adp.Where{adp.EQ("token", token)},
	})
}

func (m *Manager) RevokeByID(ctx context.Context, sessionID string) error {
	return m.adapter.Delete(ctx, "sessions", adp.Query{
		Where: []adp.Where{adp.EQ("id", sessionID)},
	})
}

func (m *Manager) ListByUserID(ctx context.Context, userID string) ([]*models.Session, error) {
	recs, err := m.adapter.FindMany(ctx, "sessions", adp.Query{
		Where: []adp.Where{adp.EQ("user_id", userID)},
	})
	if err != nil {
		return nil, err
	}
	out := make([]*models.Session, 0, len(recs))
	for _, r := range recs {
		out = append(out, recordToSession(r))
	}
	return out, nil
}

func (m *Manager) RevokeAllForUser(ctx context.Context, userID, exceptToken string) error {
	sessions, err := m.ListByUserID(ctx, userID)
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if s.Token == exceptToken {
			continue
		}
		if err := m.RevokeByID(ctx, s.ID); err != nil {
			return err
		}
	}
	return nil
}

func IsExpired(sess *models.Session) bool {
	return time.Now().UTC().After(sess.ExpiresAt)
}

func recordToSession(r map[string]any) *models.Session {
	s := &models.Session{}
	s.ID, _ = r["id"].(string)
	s.Token, _ = r["token"].(string)
	s.UserID, _ = r["user_id"].(string)
	if v, ok := r["expires_at"].(time.Time); ok {
		s.ExpiresAt = v
	}
	if v, ok := r["ip_address"].(string); ok {
		s.IPAddress = &v
	}
	if v, ok := r["user_agent"].(string); ok {
		s.UserAgent = &v
	}
	if v, ok := r["created_at"].(time.Time); ok {
		s.CreatedAt = v
	}
	if v, ok := r["updated_at"].(time.Time); ok {
		s.UpdatedAt = v
	}
	return s
}

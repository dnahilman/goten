package goten

import (
	"context"
	"fmt"
	"time"

	adp "github.com/dnahilman/goten/adapter"
	"github.com/dnahilman/goten/crypto"
	"github.com/dnahilman/goten/models"
)

// InternalAdapter provides typed CRUD methods on top of the raw Adapter interface.
type InternalAdapter struct {
	adapter adp.Adapter
}

func NewInternalAdapter(a adp.Adapter) *InternalAdapter {
	return &InternalAdapter{adapter: a}
}

// Adapter returns the raw underlying adapter for plugins that need direct access.
func (ia *InternalAdapter) Adapter() adp.Adapter { return ia.adapter }

// --- Users ---

func (ia *InternalAdapter) CreateUser(ctx context.Context, email, name string, emailVerified bool) (*models.User, error) {
	return ia.CreateUserWithExtra(ctx, email, name, emailVerified, nil)
}

func (ia *InternalAdapter) CreateUserWithExtra(ctx context.Context, email, name string, emailVerified bool, extra map[string]any) (*models.User, error) {
	now := time.Now().UTC()
	data := map[string]any{
		"id":             crypto.NewID(),
		"email":          email,
		"name":           name,
		"email_verified": emailVerified,
		"created_at":     now,
		"updated_at":     now,
	}
	for k, v := range extra {
		data[k] = v
	}
	rec, err := ia.adapter.Create(ctx, "users", data)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	return recordToUser(rec), nil
}

func (ia *InternalAdapter) FindUserByID(ctx context.Context, id string) (*models.User, error) {
	rec, err := ia.adapter.FindOne(ctx, "users", adp.Query{Where: []adp.Where{adp.EQ("id", id)}})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return recordToUser(rec), nil
}

func (ia *InternalAdapter) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	rec, err := ia.adapter.FindOne(ctx, "users", adp.Query{Where: []adp.Where{adp.EQ("email", email)}})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return recordToUser(rec), nil
}

func (ia *InternalAdapter) UpdateUser(ctx context.Context, id string, data map[string]any) (*models.User, error) {
	data["updated_at"] = time.Now().UTC()
	rec, err := ia.adapter.Update(ctx, "users", adp.Query{Where: []adp.Where{adp.EQ("id", id)}}, data)
	if err != nil {
		return nil, err
	}
	return recordToUser(rec), nil
}

func (ia *InternalAdapter) DeleteUser(ctx context.Context, id string) error {
	return ia.adapter.Delete(ctx, "users", adp.Query{Where: []adp.Where{adp.EQ("id", id)}})
}

// --- Accounts ---

func (ia *InternalAdapter) CreateAccount(ctx context.Context, userID, accountID, providerID string, extra map[string]any) (*models.Account, error) {
	now := time.Now().UTC()
	data := map[string]any{
		"id":          crypto.NewID(),
		"user_id":     userID,
		"account_id":  accountID,
		"provider_id": providerID,
		"created_at":  now,
		"updated_at":  now,
	}
	for k, v := range extra {
		data[k] = v
	}
	rec, err := ia.adapter.Create(ctx, "accounts", data)
	if err != nil {
		return nil, fmt.Errorf("creating account: %w", err)
	}
	return recordToAccount(rec), nil
}

func (ia *InternalAdapter) FindAccountByProviderAndID(ctx context.Context, providerID, accountID string) (*models.Account, error) {
	rec, err := ia.adapter.FindOne(ctx, "accounts", adp.Query{
		Where: []adp.Where{adp.EQ("provider_id", providerID), adp.EQ("account_id", accountID)},
	})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return recordToAccount(rec), nil
}

func (ia *InternalAdapter) FindAccountsByUserID(ctx context.Context, userID string) ([]*models.Account, error) {
	recs, err := ia.adapter.FindMany(ctx, "accounts", adp.Query{Where: []adp.Where{adp.EQ("user_id", userID)}})
	if err != nil {
		return nil, err
	}
	out := make([]*models.Account, 0, len(recs))
	for _, r := range recs {
		out = append(out, recordToAccount(r))
	}
	return out, nil
}

func (ia *InternalAdapter) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	acc, err := ia.FindAccountByProviderAndID(ctx, "credential", userID)
	if err != nil {
		return fmt.Errorf("finding credential account: %w", err)
	}
	if acc == nil {
		_, err = ia.CreateAccount(ctx, userID, userID, "credential", map[string]any{
			"password": hashedPassword,
		})
		return err
	}
	_, err = ia.adapter.Update(ctx, "accounts", adp.Query{Where: []adp.Where{adp.EQ("id", acc.ID)}}, map[string]any{
		"password":   hashedPassword,
		"updated_at": time.Now().UTC(),
	})
	return err
}

// --- Record converters ---

func recordToUser(r map[string]any) *models.User {
	u := &models.User{}
	u.ID, _ = r["id"].(string)
	u.Email, _ = r["email"].(string)
	u.Name, _ = r["name"].(string)
	u.EmailVerified, _ = r["email_verified"].(bool)
	if v, ok := r["image"].(string); ok {
		u.Image = &v
	}
	if v, ok := r["created_at"].(time.Time); ok {
		u.CreatedAt = v
	}
	if v, ok := r["updated_at"].(time.Time); ok {
		u.UpdatedAt = v
	}
	return u
}

func recordToAccount(r map[string]any) *models.Account {
	a := &models.Account{}
	a.ID, _ = r["id"].(string)
	a.UserID, _ = r["user_id"].(string)
	a.AccountID, _ = r["account_id"].(string)
	a.ProviderID, _ = r["provider_id"].(string)
	if v, ok := r["password"].(string); ok {
		a.Password = &v
	}
	if v, ok := r["created_at"].(time.Time); ok {
		a.CreatedAt = v
	}
	if v, ok := r["updated_at"].(time.Time); ok {
		a.UpdatedAt = v
	}
	return a
}

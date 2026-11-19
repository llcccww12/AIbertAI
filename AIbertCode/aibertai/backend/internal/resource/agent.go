package resource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"uuid"

	"aibertai.local/backend/internal/identity"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func owned(o Object, actor string) bool {
	return actor != "" && o.String("ownership_type") == "user" && o.String("owner_user_id") == actor
}

func (c *CRUD) GetUser(ctx context.Context, q Queryer, id, actor string) (Object, error) {
	o, err := DecodeObject(c.Def.Repository(q).GetResource(ctx, id))
	if err != nil {
		return nil, err
	}
	if !owned(o, actor) {
		return nil, NotFound
	}
	o, err = c.decorate(ctx, q, o)
	if err != nil {
		return nil, err
	}
	if c.Def.UserDecorate != nil {
		c.Def.UserDecorate(o)
	}
	return o, nil
}

func (c *CRUD) SaveUser(ctx context.Context, actor, id, match string, in Object) (Object, error) {
	return c.saveUser(ctx, actor, id, match, "", in)
}

// ImportUser reuses the normal personal-resource validation. The key belongs to
// the source record, so retries also work after a lost response or on another device.
func (c *CRUD) ImportUser(ctx context.Context, actor, key string, in Object) (Object, error) {
	sourceKey, err := hex.DecodeString(key)
	if err != nil || len(sourceKey) != sha256.Size || (c.Def.Kind != "expert" && c.Def.Kind != "connector") {
		return nil, Invalid("导入标识无效")
	}
	sum := sha256.Sum256([]byte("aibertai/desktop-import/v1\x00" + actor + "\x00" + c.Def.Kind + "\x00" + hex.EncodeToString(sourceKey)))
	var id uuid.UUID
	copy(id[:], sum[:16])
	id[6] = id[6]&0x0f | 0x80
	id[8] = id[8]&0x3f | 0x80
	item, err := c.saveUser(ctx, actor, "", "", id.String(), in)
	var conflict *pgconn.PgError
	if errors.As(err, &conflict) && conflict.Code == "23505" {
		// A concurrent import may have committed the same ID since our lookup.
		if existing, readErr := c.GetUser(ctx, c.Store.Pool, id.String(), actor); readErr == nil {
			return existing, nil
		}
	}
	return item, err
}

func (c *CRUD) saveUser(ctx context.Context, actor, id, match, importID string, in Object) (Object, error) {
	if actor == "" || len(c.Def.UserFields) == 0 {
		return nil, NotFound
	}
	input := Object{}
	for _, key := range c.Def.UserFields {
		if value, ok := in[key]; ok {
			input[key] = value
		}
	}
	return c.save(ctx, actor, id, match, input, true, importID)
}

func (c *CRUD) DeleteUser(ctx context.Context, actor, id, match string) error {
	return c.delete(ctx, actor, id, match, true)
}

func (c *CRUD) LockOwned(ctx context.Context, tx pgx.Tx, id, actor string) error {
	o, err := DecodeObject(c.Def.Repository(tx).LockResource(ctx, id))
	if err != nil {
		return err
	}
	if !owned(o, actor) {
		return NotFound
	}
	return nil
}

func (c *CRUD) TouchShared(ctx context.Context, tx pgx.Tx, id string) error {
	return c.Def.Repository(tx).TouchResource(ctx, id)
}

func (c *CRUD) RegisterAgent(router chi.Router) {
	path := c.Def.Path
	router.Get(path+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		user, _ := identity.UserFromContext(r.Context())
		item, err := c.GetUser(r.Context(), c.Store.Pool, chi.URLParam(r, "id"), user.ID)
		if err != nil {
			Fail(r.Context(), w, err)
			return
		}
		ETag(w, item)
		JSON(r.Context(), w, http.StatusOK, item)
	})
	save := func(w http.ResponseWriter, r *http.Request) {
		var input Object
		if err := Decode(w, r, &input); err != nil {
			Fail(r.Context(), w, err)
			return
		}
		user, _ := identity.UserFromContext(r.Context())
		item, err := c.SaveUser(r.Context(), user.ID, chi.URLParam(r, "id"), r.Header.Get("If-Match"), input)
		if err != nil {
			Fail(r.Context(), w, err)
			return
		}
		status := http.StatusOK
		if r.Method == http.MethodPost {
			status = http.StatusCreated
		}
		ETag(w, item)
		JSON(r.Context(), w, status, item)
	}
	router.Post(path, save)
	if c.Def.Kind == "expert" {
		router.Put(path+"/desktop-import/{key}", func(w http.ResponseWriter, r *http.Request) {
			var input Object
			if err := Decode(w, r, &input); err != nil {
				Fail(r.Context(), w, err)
				return
			}
			user, _ := identity.UserFromContext(r.Context())
			delete(input, "avatar")
			item, err := c.ImportUser(r.Context(), user.ID, chi.URLParam(r, "key"), input)
			if err != nil {
				Fail(r.Context(), w, err)
				return
			}
			ETag(w, item)
			JSON(r.Context(), w, http.StatusOK, item)
		})
	}
	router.Put(path+"/{id}", save)
	router.Delete(path+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		user, _ := identity.UserFromContext(r.Context())
		if err := c.DeleteUser(r.Context(), user.ID, chi.URLParam(r, "id"), r.Header.Get("If-Match")); err != nil {
			Fail(r.Context(), w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

package etcdstore

import (
	"context"
	"encoding/base32"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"go.etcd.io/etcd/client/v3"
)

// EtcdStore stores sessions in a etcd backend.
type EtcdStore struct {
	Client  *clientv3.Client
	Context context.Context
	Codecs  []securecookie.Codec
	Options *sessions.Options

	keyPrefix string
}

func NewEtcdStore(config clientv3.Config, ctx context.Context, prefix string, keyPairs ...[]byte) (*EtcdStore, error) {
	client, err := clientv3.New(config)
	if err != nil {
		return nil, err
	}

	if prefix == "" {
		prefix = "/sessions"
	}

	return &EtcdStore{
		Client:    client,
		Context:   ctx,
		keyPrefix: prefix,
		Codecs:    securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
	}, nil
}

func (s *EtcdStore) load(session *sessions.Session) error {
	key := s.keyPrefix + "/" + session.ID
	resp, err := s.Client.Get(s.Context, key)
	if err != nil {
		return err
	}

	if resp.Count == 0 {
		return fmt.Errorf("key: %s is not found in etcd", key)
	}

	if err = securecookie.DecodeMulti(session.Name(), string(resp.Kvs[0].Value), &session.Values, s.Codecs...); err != nil {
		return err
	}

	return nil
}

func (s *EtcdStore) delete(session *sessions.Session) error {
	key := s.keyPrefix + "/" + session.ID
	resp, err := s.Client.Delete(s.Context, key)
	if err != nil {
		return err
	}

	if resp.Deleted == 0 {
		return fmt.Errorf("key: %s is not found in etcd", key)
	}

	return nil
}

// save writes encoded session.Values to etcd.
func (s *EtcdStore) save(session *sessions.Session) error {
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values,
		s.Codecs...)
	if err != nil {
		return err
	}

	key := s.keyPrefix + "/" + session.ID

	grant, err := s.Client.Grant(s.Context, int64(session.Options.MaxAge+1))
	if err != nil {
		return err
	}

	_, err = s.Client.Put(s.Context, key, encoded, clientv3.WithLease(grant.ID))
	if err != nil {
		return err
	}

	return nil
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting Options.MaxAge
// = -1 for that session.
func (s *EtcdStore) MaxAge(age int) {
	s.Options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range s.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

// New returns a session for the given name without adding it to the registry.
//
// See gorilla/sessions CookieStore.New().
func (s *EtcdStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)

	options := *s.Options
	session.Options = &options
	session.IsNew = true

	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			err = s.load(session)
			if err == nil {
				session.IsNew = false
			}
		}
	}

	return session, err
}

// Get returns a session for the given name after adding it to the registry.
//
// See gorilla/sessions CookieStore.Get().
func (s *EtcdStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// Save adds a single session to the response.
func (s *EtcdStore) Save(_ *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	if session.Options.MaxAge <= 0 {
		if err := s.delete(session); err != nil {
			return err
		}

		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	if session.ID == "" {
		session.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(32)), "=")
	}

	if err := s.save(session); err != nil {
		return err
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
	if err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

// Close the etcd client
func (s *EtcdStore) Close() error {
	return s.Client.Close()
}

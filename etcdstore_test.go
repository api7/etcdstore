package etcdstore

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/client/v3"
)

const (
	_defaultEtcd = "http://127.0.0.1:2379"
)

var (
	store *EtcdStore
)

func init() {
	var err error
	store, err = NewEtcdStore(clientv3.Config{Endpoints: []string{_defaultEtcd}}, context.Background(), "/sessions", []byte("secret"))
	if err != nil {
		panic(err)
	}

}

func TestEtcdStore_New(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	assert.Nil(t, err, "http new request")

	session, err := store.New(req, "_session")
	assert.Nil(t, err, "store new session")
	assert.True(t, session.IsNew)
	assert.Len(t, session.Values, 0)
}

func TestEtcdStore_Get(t *testing.T) {
	// req without cookie
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	assert.Nil(t, err, "http new request")

	session, err := store.Get(req, "_session")
	assert.Nil(t, err, "store new session")
	assert.True(t, session.IsNew)
	assert.Len(t, session.Values, 0)
}

func TestEtcdStore_Save(t *testing.T) {
	// req without session header
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	assert.Nil(t, err, "http new request")

	rsp := httptest.NewRecorder()

	session, err := store.New(req, "_session")
	assert.Nil(t, err)
	assert.True(t, session.IsNew)

	session.Values["foo"] = "bar"

	err = session.Save(req, rsp)
	assert.Nil(t, err)

	cookies := rsp.Header().Values("Set-Cookie")
	assert.Len(t, cookies, 1, "cookie header's length")

	// req with session header
	req2, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	assert.Nil(t, err, "http new request")
	req2.Header.Add("Cookie", cookies[0])
	session2, err := store.New(req2, "_session")
	assert.Nil(t, err)
	assert.False(t, session2.IsNew)
	assert.Equal(t, session2.Values["foo"], "bar")

	// We set maxage = -1 to delete the session
	session2.Options.MaxAge = -1
	err = session2.Save(req, rsp)
	assert.Nil(t, err)
}

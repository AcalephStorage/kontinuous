package kv

import (
	"errors"
	"strings"
	"testing"

	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	etcd "github.com/coreos/etcd/client"
)

// A Mock of KV KeysAPI interface etd.KeysAPI
type MockKeysAPI struct {
	// for simulating error
	mockError error
	// dummy data store
	data map[string]string
}

func (m *MockKeysAPI) Set(ctx context.Context, key, value string, opts *etcd.SetOptions) (*etcd.Response, error) {
	m.data[key] = value
	return &etcd.Response{}, m.mockError
}

func (m *MockKeysAPI) Get(ctx context.Context, key string, opts *etcd.GetOptions) (*etcd.Response, error) {
	switch {
	case opts.Sort && opts.Quorum && opts.Recursive:
		var nodes etcd.Nodes
		for k, v := range m.data {
			if strings.HasPrefix(k, key) {
				nodes = append(nodes, &etcd.Node{
					Key:           k,
					Value:         v,
					ModifiedIndex: 1,
				})
			}
		}
		return &etcd.Response{
			Node: &etcd.Node{
				Nodes: nodes,
			},
		}, m.mockError
	}

	val, ok := m.data[key]
	if !ok {
		return nil, errors.New("key not found")
	}
	return &etcd.Response{
		Node: &etcd.Node{
			Value: val,
		},
	}, m.mockError
}

func TestPutSuccess(t *testing.T) {
	store := &MockKeysAPI{data: map[string]string{}}
	kvClient := &etcdClient{store}

	err := kvClient.Put("hello", "world")
	if err != nil {
		t.Error("successful put operation should return without error")
	}

	err = kvClient.PutInt("hi", 5)
	if err != nil {
		t.Error("successful put operation should return without error")
	}
}

func TestPutFail(t *testing.T) {
	store := &MockKeysAPI{
		mockError: errors.New("unable to put data"),
		data:      map[string]string{},
	}
	kvClient := &etcdClient{store}

	err := kvClient.Put("hello", "world")
	if err == nil {
		t.Error("failing put operation should return an error")
	}

	err = kvClient.PutInt("hi", 5)
	if err == nil {
		t.Error("failing put operation should return an error")
	}
}

func TestGetSuccess(t *testing.T) {
	store := &MockKeysAPI{data: map[string]string{}}
	kvClient := &etcdClient{store}
	kvClient.Put("hello", "world")
	val, err := kvClient.Get("hello")
	if err != nil {
		t.Error("unexpected error", err)
	}
	if val != "world" {
		t.Errorf("retrieved data is %s, should be %s.", val, "world")
	}
}

func TestGetFail(t *testing.T) {
	store := &MockKeysAPI{data: map[string]string{}}
	kvClient := &etcdClient{store}
	_, err := kvClient.Get("hello")
	if err == nil {
		t.Error("failing get operation, should return an error")
	}
}

func testGetIntSuccess(t *testing.T) {
	store := &MockKeysAPI{data: map[string]string{}}
	kvClient := &etcdClient{store}
	kvClient.PutInt("hi", 5)
	val, err := kvClient.GetInt("hi")
	if err != nil {
		t.Error("unexpected error", err)
	}
	if val != 5 {
		t.Errorf("retrieved data is %d, should be %d.", val, 5)
	}
}

func TestGetIntFail(t *testing.T) {
	store := &MockKeysAPI{data: map[string]string{}}
	kvClient := &etcdClient{store}
	_, err := kvClient.GetInt("hello")
	if err == nil {
		t.Error("failing GetInt operation, should return an error")
	}
}

func TestPutDirSuccess(t *testing.T) {
	store := &MockKeysAPI{data: map[string]string{}}
	kvClient := &etcdClient{store}

	err := kvClient.PutDir("foo")
	if err != nil {
		t.Error("successful put operation should return without error")
	}

	err = kvClient.PutIntDir("bar", 5)
	if err != nil {
		t.Error("successful put operation should return without error")
	}
}

func TestPutDirFail(t *testing.T) {
	store := &MockKeysAPI{
		mockError: errors.New("unable to put a directory"),
		data:      map[string]string{},
	}
	kvClient := &etcdClient{store}

	err := kvClient.PutDir("foo")
	if err == nil {
		t.Error("failing put operation should return an error")
	}

	err = kvClient.PutIntDir("bar", 5)
	if err == nil {
		t.Error("failing put operation should return an error")
	}
}

func TestGetDirSuccess(t *testing.T) {
	store := &MockKeysAPI{data: map[string]string{}}
	kvClient := &etcdClient{store}
	kvClient.PutDir("hello")
	keys := map[string]string{
		"hello/foo":   "foo",
		"hello/bar":   "bar",
		"hello/world": "world",
	}

	for key, val := range keys {
		kvClient.Put(key, val)
	}

	data, err := kvClient.GetDir("hello")
	if err != nil {
		t.Error("unexpected error", err)
	}

	ctr := 0
	for _, d := range data {
		if keys[d.Key] != "" {
			ctr++
		}
	}
	if ctr != 3 {
		t.Errorf("failed to retrieve expected data, should return 3 items not %d", ctr)
	}
}

// unimplemented mock methods

func (m *MockKeysAPI) Create(ctx context.Context, key, value string) (*etcd.Response, error) {
	return nil, nil
}

func (m *MockKeysAPI) Delete(ctx context.Context, key string, opts *etcd.DeleteOptions) (*etcd.Response, error) {
	return nil, nil
}

func (m *MockKeysAPI) CreateInOrder(ctx context.Context, dir, value string, opts *etcd.CreateInOrderOptions) (*etcd.Response, error) {
	return nil, nil
}

func (m *MockKeysAPI) Watcher(key string, opts *etcd.WatcherOptions) etcd.Watcher {
	return nil
}

func (m *MockKeysAPI) Update(ctx context.Context, key, value string) (*etcd.Response, error) {
	return nil, nil
}

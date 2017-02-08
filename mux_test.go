package mux

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func foo(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("foo"))
}

func bar(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("foo"))
}

func baz(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("foo"))
}

func TestHandler(t *testing.T) {
	mux := NewMux()
	mux.Entry(GET, "/foo", foo)
	mux.Entry(GET, "/bar", bar)
	mux.Entry(GET, "/baz", baz)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// foo
	res, err := http.Get(ts.URL + "/foo")
	if err != nil {
		t.Error("unexpected")
		return
	}

	if res.StatusCode != 200 {
		t.Error("Status code error")
		return
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		t.Error(err)
		return
	}

	if bytes.Compare(body, []byte("foo")) < 0 {
		t.Error("Unexpected Body")
		return
	}

	// bar
	res, err = http.Get(ts.URL + "/bar")
	if err != nil {
		t.Error("unexpected")
		return
	}

	if res.StatusCode != 200 {
		t.Error("Status code error")
		return
	}

	body, err = ioutil.ReadAll(res.Body)

	if err != nil {
		t.Error(err)
		return
	}

	if bytes.Compare(body, []byte("bar")) < 0 {
		t.Error("Unexpected Body")
		return
	}

	// baz
	res, err = http.Get(ts.URL + "/baz")
	if err != nil {
		t.Error("unexpected")
		return
	}

	if res.StatusCode != 200 {
		t.Error("Status code error")
		return
	}

	body, err = ioutil.ReadAll(res.Body)

	if err != nil {
		t.Error(err)
		return
	}

	if bytes.Compare(body, []byte("baz")) < 0 {
		t.Error("Unexpected Body")
		return
	}

	// not found
	res, err = http.Get(ts.URL + "/hoge")
	if err != nil {
		t.Error("unexpected")
		return
	}

	if res.StatusCode != 404 {
		t.Error("Status code error")
		return
	}

	body, err = ioutil.ReadAll(res.Body)

	if err != nil {
		t.Error(err)
		return
	}

	if bytes.Compare(body, []byte("404 page not found")) < 0 {
		t.Errorf("Unexpected Body: %s", body)
		return
	}
}

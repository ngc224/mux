package mux

import (
	"context"
	"net/http"
)

const (
	GET         = "GET"
	POST        = "POST"
	PUT         = "PUT"
	DELETE      = "DELETE"
	HEAD        = "HEAD"
	OPTIONS     = "OPTIONS"
	PATCH       = "PATCH"
	ctxRouteKey = "mux"
)

var (
	charColon    = ':'
	charWildcard = '*'
	charSlash    = '/'
	byteColon    = byte(charColon)
	byteWildcard = byte(charWildcard)
	byteSlash    = byte(charSlash)
)

type (
	Mux struct {
		tree     []*node
		static   map[string]http.HandlerFunc
		NotFound http.HandlerFunc
	}

	node struct {
		child       map[byte]*node
		handlerFunc http.HandlerFunc //End is handlerFunc != nil
		param       string
	}

	rContext struct {
		params params
	}

	params []param

	param struct {
		key   string
		value string
	}
)

func NewMux() *Mux {
	return &Mux{
		static:   make(map[string]http.HandlerFunc),
		NotFound: http.NotFound,
	}
}

func (ps *params) Set(key, value string) {
	*ps = append(*ps, param{
		key:   key,
		value: value,
	})
}

func (ps params) Get(key string) string {
	for _, v := range ps {
		if v.key == key {
			return v.value
		}
	}

	return ""
}

func URLParam(r *http.Request, key string) string {
	if ctx := r.Context().Value(ctxRouteKey); ctx != nil {
		if ctx, ok := ctx.(*rContext); ok {
			return ctx.params.Get(key)
		}
	}

	return ""
}

func isStaticPattern(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == byteColon || pattern[i] == byteWildcard {
			return false
		}
	}

	return true
}

func (m *Mux) Get(pattern string, handlerFunc http.HandlerFunc) {
	m.Entry(GET, pattern, handlerFunc)
}

func (m *Mux) Post(pattern string, handlerFunc http.HandlerFunc) {
	m.Entry(POST, pattern, handlerFunc)
}

func (m *Mux) Put(pattern string, handlerFunc http.HandlerFunc) {
	m.Entry(PUT, pattern, handlerFunc)
}

func (m *Mux) Delete(pattern string, handlerFunc http.HandlerFunc) {
	m.Entry(DELETE, pattern, handlerFunc)
}

func (m *Mux) Head(pattern string, handlerFunc http.HandlerFunc) {
	m.Entry(HEAD, pattern, handlerFunc)
}

func (m *Mux) Options(pattern string, handlerFunc http.HandlerFunc) {
	m.Entry(OPTIONS, pattern, handlerFunc)
}

func (m *Mux) Patch(pattern string, handlerFunc http.HandlerFunc) {
	m.Entry(PATCH, pattern, handlerFunc)
}

func (m *Mux) Entry(method, pattern string, handlerFunc http.HandlerFunc) {
	if pattern[0] != byteSlash {
		panic("There is no leading slash")
	}

	s := method + pattern

	if isStaticPattern(pattern) {
		m.static[s] = handlerFunc
		return
	}

	var treeIndex int

	for i := 0; i < len(s); i++ {
		edge := s[i]

		if treeIndex == len(m.tree) {
			m.tree = append(m.tree, &node{
				child: make(map[byte]*node),
			})
		}

		tree := m.tree[treeIndex]

		if _, ok := tree.child[edge]; ok {
			treeIndex += 1
			continue
		}

		n := &node{}

		if edge == byteColon {
			p := []byte{}

			for ; i < len(s); i++ {
				if s[i] == byteSlash {
					i -= 1
					break
				}
				if s[i] != byteColon {
					p = append(p, s[i])
				}
			}

			n.param = string(p)
		}

		if edge == byteWildcard {
			for ; i < len(s); i++ {
				if s[i] == byteSlash {
					i -= 1
					break
				}
			}
		}

		if i >= len(s)-1 {
			n.handlerFunc = handlerFunc
		}

		treeIndex += 1
		tree.child[edge] = n
	}
}

func (m *Mux) lookup(r *http.Request) (http.HandlerFunc, *rContext) {
	s := r.Method + r.URL.Path

	if fn, ok := m.static[s]; ok {
		return fn, nil
	}

	if len(m.tree) == 0 {
		return nil, nil
	}

	ctx := &rContext{}

	var treeIndex int

	for i := 0; i < len(s); i++ {
		edge := s[i]

		tree := m.tree[treeIndex]
		treeIndex += 1

		if _, ok := tree.child[edge]; ok {
			if tree.child[edge].handlerFunc != nil {
				return tree.child[edge].handlerFunc, ctx
			}

			continue
		}

		if n, ok := tree.child[byteColon]; ok {
			p := []byte{}

			for ; i < len(s); i++ {
				if s[i] == byteSlash {
					i -= 1
					break
				}

				p = append(p, s[i])
			}

			ctx.params.Set(n.param, string(p))

			if n.handlerFunc != nil {
				return n.handlerFunc, ctx
			}

			continue
		}

		if n, ok := tree.child[byteWildcard]; ok {
			for ; i < len(s); i++ {
				if s[i] == byteSlash {
					i -= 1
					break
				}
			}

			if n.handlerFunc != nil {
				return n.handlerFunc, ctx
			}

			continue
		}

		break
	}

	return nil, nil
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if fn, ctx := m.lookup(r); fn != nil {
		if ctx != nil {
			fn.ServeHTTP(w, r.WithContext(context.WithValue(
				r.Context(), ctxRouteKey, ctx),
			))
			return
		}

		fn.ServeHTTP(w, r)
		return
	}

	m.NotFound.ServeHTTP(w, r)
}

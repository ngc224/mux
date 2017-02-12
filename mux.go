package mux

import (
	"context"
	"net/http"
	"strings"
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
	charWildCard = '*'
	charSlash    = '/'
	byteColon    = byte(charColon)
	byteWildCard = byte(charWildCard)
	byteSlash    = byte(charSlash)
)

type (
	Mux struct {
		node     *node
		NotFound http.HandlerFunc
	}

	node struct {
		static map[routeStatic]http.HandlerFunc
		param  map[routeParam][]routeParamValue
	}

	routeStatic struct {
		method  string
		pattern string
	}

	routeParam struct {
		method   string
		dirIndex int
	}

	routeParamValue struct {
		pattern     string
		handlerFunc http.HandlerFunc
		dirs        []string
		dirIndex    int
	}

	routeContext struct {
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
		node: &node{
			param:  make(map[routeParam][]routeParamValue),
			static: make(map[routeStatic]http.HandlerFunc),
		},
		NotFound: http.NotFound,
	}
}

func newRouteStatic(method, pattern string) routeStatic {
	return routeStatic{
		method:  method,
		pattern: pattern,
	}
}

func newRouteParam(method string, index int) routeParam {
	return routeParam{
		method:   method,
		dirIndex: index,
	}
}

func newRouteContext() *routeContext {
	return &routeContext{}
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

func isParamPattern(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == byteColon || pattern[i] == byteWildCard {
			return true
		}
	}

	return false
}

func URLParam(r *http.Request, key string) string {
	if ctx := r.Context().Value(ctxRouteKey); ctx != nil {
		if ctx, ok := ctx.(*routeContext); ok {
			return ctx.params.Get(key)
		}
	}

	return ""
}

func (m *Mux) Entry(method, pattern string, handlerFunc http.HandlerFunc) {
	if pattern[0] != byteSlash {
		panic("There is no leading slash")
	}

	if isParamPattern(pattern) {
		dirs, dirIndex := dirResolve(pattern)
		rt := newRouteParam(method, dirIndex)
		m.node.param[rt] = append(m.node.param[rt], routeParamValue{
			pattern:     pattern,
			handlerFunc: handlerFunc,
			dirs:        dirs,
			dirIndex:    dirIndex,
		})

		return
	}

	m.node.static[newRouteStatic(method, pattern)] = handlerFunc
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

func dirResolve(dir string) ([]string, int) {
	dirs := strings.FieldsFunc(dir, func(r rune) bool {
		return r == charSlash
	})

	if len(dirs) > 0 {
		return dirs, len(dirs) - 1
	}

	return dirs, 0
}

func (n *node) lookup(r *http.Request) (http.HandlerFunc, *routeContext) {
	if fn, ok := n.static[newRouteStatic(r.Method, r.URL.Path)]; ok {
		return fn, nil
	}

	rDirs, rDirIndex := dirResolve(r.URL.Path)
	ctx := newRouteContext()

	for _, v := range n.param[newRouteParam(r.Method, rDirIndex)] {
		for i, vv := range v.dirs {
			if vv[0] == byteWildCard {
				if i == v.dirIndex {
					return v.handlerFunc, ctx
				}

				continue
			}

			if vv[0] == byteColon {
				ctx.params.Set(vv[1:], rDirs[i])

				if i == v.dirIndex {
					return v.handlerFunc, ctx
				}

				continue
			}

			if vv == rDirs[i] {
				if i == v.dirIndex {
					return v.handlerFunc, ctx
				}

				continue
			}

			break
		}
	}

	return nil, nil
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if fn, ctx := m.node.lookup(r); fn != nil {
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

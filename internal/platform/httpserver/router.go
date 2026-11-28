package httpserver

import (
	"net/http"
	"slices"
)

// Router registers routes on one ServeMux under a path prefix, wrapping each in the router's middleware.
type Router struct {
	mux    *http.ServeMux
	prefix string
	mws    []Middleware
}

// NewRouter returns a Router on a new ServeMux, with no prefix and no middleware.
func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

// Group returns a Router on the same mux whose prefix is r's prefix followed by prefix, and whose middleware
// is r's followed by middlewares.
func (r *Router) Group(prefix string, middlewares ...Middleware) *Router {
	return &Router{mux: r.mux, prefix: r.prefix + prefix, mws: append(slices.Clone(r.mws), middlewares...)}
}

// Handle registers handler as "<method> <prefix><path>", wrapped in r's middleware with the first outermost. A
// request it serves has its span (see Instrument) named "<method> <prefix><path>", and its span and metrics
// carry http.route "<prefix><path>". It panics when the pattern conflicts with one already registered, as
// ServeMux.Handle does.
func (r *Router) Handle(method, path string, handler http.Handler) {
	route := r.prefix + path
	r.mux.Handle(method+" "+route, labelRoute(method, route, Chain(handler, r.mws...)))
}

// HandleFunc registers handler as Handle does.
func (r *Router) HandleFunc(method, path string, handler http.HandlerFunc) {
	r.Handle(method, path, handler)
}

// Handler serves every route registered through r or its groups. A path no route matches gets a 404
// not_found problem; a method the path has no route for gets a 405 method_not_allowed problem with the
// Allow header. The mux's redirects to cleaned paths pass through.
func (r *Router) Handler() http.Handler {
	return withRouteProblems(r.mux)
}

func withRouteProblems(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(&routeProblems{ResponseWriter: w, r: r}, r)
	})
}

type routeProblems struct {
	http.ResponseWriter
	r        *http.Request
	replaced bool
}

func (p *routeProblems) WriteHeader(status int) {
	switch status {
	case http.StatusNotFound:
		p.replace(Problem{
			Status: status,
			Code:   "not_found",
			Detail: "no endpoint matches the request path",
		})
	case http.StatusMethodNotAllowed:
		p.replace(Problem{
			Status: status,
			Code:   "method_not_allowed",
			Detail: "the endpoint does not accept this method; the Allow header lists those it does",
		})
	default:
		p.ResponseWriter.WriteHeader(status)
	}
}

func (p *routeProblems) replace(problem Problem) {
	p.replaced = true
	WriteProblem(p.ResponseWriter, p.r, problem)
}

func (p *routeProblems) Write(data []byte) (int, error) {
	if p.replaced {
		return len(data), nil
	}
	return p.ResponseWriter.Write(data)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (p *routeProblems) Unwrap() http.ResponseWriter { return p.ResponseWriter }

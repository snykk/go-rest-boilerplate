package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	V1Handler "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
)

// usersRoute wires the /users/* group — endpoints scoped to a user's
// own profile / data. Auth flows live in route.auth.go.
type usersRoute struct {
	handler        V1Handler.UserHandler
	router         *gin.RouterGroup
	authMiddleware gin.HandlerFunc
}

// NewUsersRoute builds the route module. The auth middleware is
// passed in (rather than constructed here) so the same JWT-validating
// middleware is shared across every protected route group.
func NewUsersRoute(router *gin.RouterGroup, usersUC users.Usecase, authMiddleware gin.HandlerFunc) *usersRoute {
	return &usersRoute{
		handler:        V1Handler.NewUserHandler(usersUC),
		router:         router,
		authMiddleware: authMiddleware,
	}
}

// Routes mounts the /users group and its endpoints.
func (r *usersRoute) Routes() {
	v1 := r.router.Group("/v1")
	users := v1.Group("/users")
	users.Use(r.authMiddleware)
	{
		users.GET("/me", r.handler.GetUserData)
	}
}

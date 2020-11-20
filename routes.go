package user_controller

import (
	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/game"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
)

const (
	authPath   = "auth"
	loginPath  = "login"
	logoutPath = "logout"
)

type Client struct {
	DS    *datastore.Client
	User  user.Client
	Stats stats.Client
	Game  game.Client
}

func NewClient(dsClient *datastore.Client, userClient *datastore.Client) Client {
	return Client{
		DS:    dsClient,
		User:  user.NewClient(dsClient),
		Stats: stats.NewClient(dsClient),
		Game:  game.NewClient(dsClient),
	}
}

func userFrom(c *gin.Context) (*user.User, error) {
	return user.From(c), nil
}

func (client Client) AddRoutes(prefix string, engine *gin.Engine) *gin.Engine {
	// User Group
	g1 := engine.Group(prefix)
	g1.GET("/new",
		// user.RequireLogin(),
		client.NewAction,
	)

	// Create
	g1.POST("",
		// user.RequireLogin(),
		client.Create(prefix),
	)

	// Show User
	g1.GET("show/:uid",
		client.User.Fetch,
		client.Stats.Fetch(userFrom),
		client.Show,
	)

	// Edit User
	g1.GET("edit/:uid",
		// user.RequireLogin(),
		client.User.Fetch,
		client.Stats.Fetch(userFrom),
		client.Edit,
	)

	// Update User
	g1.POST("update/:uid",
		// user.RequireLogin(),
		client.User.Fetch,
		client.Update,
	)

	// User Games
	g1.POST("show/:uid/games/json",
		client.Game.GetFiltered(gtype.All),
		client.Game.JSONIndexAction,
	)

	g1.POST("edit/:uid/games/json",
		// user.RequireLogin(),
		client.Game.GetFiltered(gtype.All),
		client.Game.JSONIndexAction,
	)

	g1.GET("as/:uid",
		user.RequireAdmin,
		client.User.As,
	)

	g1.GET(loginPath, user.Login("/"+prefix+"/"+authPath))

	g1.GET(logoutPath, user.Logout)

	g1.GET(authPath, client.User.Auth("/"+prefix+"/"+authPath))

	// Users group
	g2 := engine.Group(prefix + "s")

	// Index
	g2.GET("",
		user.RequireAdmin,
		client.Index,
	)

	// json data for Index
	g2.POST("/json",
		user.RequireAdmin,
		client.User.FetchAll,
		client.JSON,
	)

	return engine
}

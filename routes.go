package user_controller

import (
	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/sn"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

type Client struct {
	*sn.Client
	User  *user.Client
	Stats *stats.Client
	Game  *game.Client
}

func NewClient(dsClient *datastore.Client, logger *log.Logger, mcache *cache.Cache, router *gin.Engine) *Client {
	logger.Debugf(msgEnter)
	defer logger.Debugf(msgExit)
	userClient := user.NewClient(logger, mcache)
	cl := &Client{
		Client: sn.NewClient(dsClient, logger, mcache, router),
		User:   userClient,
		Stats:  stats.NewClient(userClient, dsClient, logger, mcache),
		Game:   game.NewClient(userClient, dsClient, logger, mcache, router, "game"),
	}
	cl.addRoutes()
	return cl
}

func userFrom(c *gin.Context) (*user.User, error) {
	return user.From(c), nil
}

func (cl *Client) addRoutes() {
	cl.Log.Debugf(msgEnter)
	defer cl.Log.Debugf(msgExit)

	// New
	cl.Router.GET("new", cl.NewAction)

	// Create
	cl.Router.PUT("new", cl.Create)

	// Update
	cl.Router.PUT("update/:uid", cl.Update("uid"))

	// Get
	cl.Router.GET("json/:uid", cl.JSON("uid"))

	// User Games
	cl.Router.POST("show/:uid/games/json",
		cl.Game.GetFiltered(gtype.All),
		cl.Game.JSONIndexAction,
	)

	cl.Router.POST("edit/:uid/games/json",
		// user.RequireLogin(),
		cl.Game.GetFiltered(gtype.All),
		cl.Game.JSONIndexAction,
	)

	cl.Router.GET("as/:uid", cl.User.As)

	cl.Router.GET("current", cl.Current)

	cl.Router.GET("login", user.Login("auth"))

	cl.Router.GET("logout", user.Logout)

	cl.Router.GET("auth", cl.User.Auth("auth"))

	// Index
	cl.Router.GET("index", cl.Index)

	// // json data for Index
	// g2.POST("/json",
	// 	user.RequireAdmin,
	// 	cl.User.FetchAll,
	// 	cl.JSON,
	// )
}

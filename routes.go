package user_controller

import (
	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
)

type Client struct {
	DS    *datastore.Client
	User  user.Client
	Stats stats.Client
	Game  game.Client
}

func NewClient(dsClient *datastore.Client) Client {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)
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

func (client Client) AddRoutes(engine *gin.Engine) *gin.Engine {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	// New
	engine.GET("new", client.NewAction)

	// Create
	engine.PUT("new", client.Create)

	// Update
	engine.PUT("update/:uid", client.Update("uid"))

	// Get
	engine.GET("json/:uid", client.JSON("uid"))

	// User Games
	engine.POST("show/:uid/games/json",
		client.Game.GetFiltered(gtype.All),
		client.Game.JSONIndexAction,
	)

	engine.POST("edit/:uid/games/json",
		// user.RequireLogin(),
		client.Game.GetFiltered(gtype.All),
		client.Game.JSONIndexAction,
	)

	engine.GET("as/:uid",
		user.RequireAdmin,
		client.User.As,
	)

	engine.GET("current", client.Current)

	engine.GET("login", user.Login("auth"))

	engine.GET("logout", user.Logout)

	engine.GET("auth", client.User.Auth("auth"))

	// Index
	engine.GET("index", client.Index)

	// // json data for Index
	// g2.POST("/json",
	// 	user.RequireAdmin,
	// 	client.User.FetchAll,
	// 	client.JSON,
	// )

	return engine
}

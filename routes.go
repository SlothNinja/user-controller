package user_controller

import (
	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/rating"
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
	*datastore.Client
	Game   game.Client
	Rating rating.Client
	Stats  stats.Client
	User   user.Client
}

func NewClient(dsClient *datastore.Client) Client {
	return Client{
		Client: dsClient,
		Game:   game.NewClient(dsClient),
		Rating: rating.NewClient(dsClient),
		Stats:  stats.NewClient(dsClient),
		User:   user.NewClient(dsClient),
	}
}

func (client Client) AddRoutes(prefix string, engine *gin.Engine) *gin.Engine {
	// User Group
	g1 := engine.Group(prefix)
	g1.GET("/new", client.new)

	// Create
	g1.POST("", client.create(prefix))

	// Show User
	g1.GET("show/:uid", client.show)

	// Edit User
	g1.GET("edit/:uid", client.edit)

	// Update User
	g1.POST("update/:uid", client.update)

	// User Ratings
	g1.POST("show/:uid/ratings/json", client.Rating.JSONIndexAction)

	g1.POST("edit/:uid/ratings/json", client.Rating.JSONIndexAction)

	// User Games
	g1.POST("show/:uid/games/json", client.Game.JSONIndexAction)

	g1.POST("edit/:uid/games/json", client.Game.JSONIndexAction)

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
		gtype.SetTypes(),
		client.index,
	)

	// json data for Index
	g2.POST("/json",
		user.RequireAdmin,
		gtype.SetTypes(),
		client.User.FetchAll,
		client.json,
	)

	return engine
}

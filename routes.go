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
}

func NewClient(dsClient *datastore.Client) Client {
	return Client{
		Client: dsClient,
		Game:   game.NewClient(dsClient),
		Rating: rating.NewClient(dsClient),
	}
}

func (client Client) AddRoutes(prefix string, engine *gin.Engine) *gin.Engine {
	// User Group
	g1 := engine.Group(prefix)
	g1.GET("/new",
		gtype.SetTypes(),
		client.new,
	)

	// Create
	g1.POST("",
		client.create(prefix),
	)

	// Show User
	g1.GET("show/:uid",
		user.Fetch,
		stats.Fetch(user.From),
		gtype.SetTypes(),
		client.show,
	)

	// Edit User
	g1.GET("edit/:uid",
		user.Fetch,
		stats.Fetch(user.From),
		gtype.SetTypes(),
		client.edit,
	)

	// Update User
	g1.POST("update/:uid",
		user.Fetch,
		gtype.SetTypes(),
		client.update,
	)

	// User Ratings
	g1.POST("show/:uid/ratings/json",
		user.Fetch,
		client.Rating.JSONIndexAction,
	)

	g1.POST("edit/:uid/ratings/json",
		user.Fetch,
		client.Rating.JSONIndexAction,
	)

	// User Games
	g1.POST("show/:uid/games/json",
		gtype.SetTypes(),
		client.Game.GetFiltered(gtype.All),
		client.Game.JSONIndexAction,
	)

	g1.POST("edit/:uid/games/json",
		gtype.SetTypes(),
		client.Game.GetFiltered(gtype.All),
		client.Game.JSONIndexAction,
	)

	g1.GET("as/:uid",
		user.RequireAdmin,
		user.As,
	)

	g1.GET(loginPath, user.Login("/"+prefix+"/"+authPath))

	g1.GET(logoutPath, user.Logout)

	g1.GET(authPath, user.Auth("/"+prefix+"/"+authPath))

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
		user.FetchAll,
		client.json,
	)

	return engine
}

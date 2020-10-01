package user_controller

import (
	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/rating/v2"
	gtype "github.com/SlothNinja/type"
	stats "github.com/SlothNinja/user-stats/v2"
	"github.com/SlothNinja/user/v2"
	"github.com/gin-gonic/gin"
)

const (
	authPath   = "auth"
	loginPath  = "login"
	logoutPath = "logout"
)

type Client struct {
	DS     *datastore.Client
	User   user.Client
	Stats  stats.Client
	Rating rating.Client
	// Game   game.Client
}

func NewClient(dsClient *datastore.Client) Client {
	return Client{
		DS:     dsClient,
		User:   user.NewClient(dsClient),
		Stats:  stats.NewClient(dsClient),
		Rating: rating.NewClient(dsClient),
		// Game:   game.NewClient(dsClient),
	}
}

func (client Client) AddRoutes(prefix string, engine *gin.Engine) *gin.Engine {
	// User Group
	g1 := engine.Group(prefix)
	g1.GET("/new", client.NewAction)

	// Create
	g1.PUT("/new", client.Create(prefix))

	g1.GET("/json/:uid", client.JSON("uid"))

	// Edit User
	g1.PUT("/update/:uid", client.Update("uid"))

	// // Update User
	// g1.POST("update/:uid",
	// 	// user.RequireLogin(),
	// 	client.User.Fetch,
	// 	gtype.SetTypes(),
	// 	client.Update,
	// )

	// User Ratings
	g1.POST("show/:uid/ratings/json",
		client.User.Fetch,
		client.Rating.JSONIndexAction,
	)

	g1.POST("edit/:uid/ratings/json",
		// user.RequireLogin(),
		client.User.Fetch,
		client.Rating.JSONIndexAction,
	)

	// User Games
	// g1.POST("show/:uid/games/json",
	// 	gtype.SetTypes(),
	// 	client.Game.GetFiltered(gtype.All),
	// 	client.Game.JSONIndexAction,
	// )

	// g1.POST("edit/:uid/games/json",
	// 	// user.RequireLogin(),
	// 	gtype.SetTypes(),
	// 	client.Game.GetFiltered(gtype.All),
	// 	client.Game.JSONIndexAction,
	// )

	g1.GET("as/:uid",
		user.RequireAdmin,
		client.User.As,
	)

	g1.GET("current", client.Current)

	g1.GET(loginPath, user.Login(prefix+"/"+authPath))

	g1.GET(logoutPath, user.Logout)

	g1.GET(authPath, client.User.Auth(prefix+"/"+authPath))

	// Users group
	g2 := engine.Group(prefix + "s")

	// Index
	g2.GET("",
		user.RequireAdmin,
		gtype.SetTypes(),
		client.Index,
	)

	// // json data for Index
	// g2.POST("/json",
	// 	user.RequireAdmin,
	// 	gtype.SetTypes(),
	// 	client.User.FetchAll,
	// 	client.JSON,
	// )

	return engine
}

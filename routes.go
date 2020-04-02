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
	User  user.Client
	Stats stats.Client
}

func NewClient(dsClient *datastore.Client) Client {
	return Client{
		Client: dsClient,
		User:   user.NewClient(dsClient),
		Stats:  stats.NewClient(dsClient),
	}
}

func (client Client) AddRoutes(prefix string, engine *gin.Engine) *gin.Engine {
	// User Group
	g1 := engine.Group(prefix)
	g1.GET("/new",
		// user.RequireLogin(),
		gtype.SetTypes(),
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
		client.Stats.Fetch(user.From),
		gtype.SetTypes(),
		client.Show,
	)

	// Edit User
	g1.GET("edit/:uid",
		// user.RequireLogin(),
		client.User.Fetch,
		client.Stats.Fetch(user.From),
		gtype.SetTypes(),
		client.Edit,
	)

	// Update User
	g1.POST("update/:uid",
		// user.RequireLogin(),
		client.User.Fetch,
		gtype.SetTypes(),
		client.Update,
	)

	// User Ratings
	g1.POST("show/:uid/ratings/json",
		client.User.Fetch,
		rating.JSONIndexAction,
	)

	g1.POST("edit/:uid/ratings/json",
		// user.RequireLogin(),
		client.User.Fetch,
		rating.JSONIndexAction,
	)

	// User Games
	g1.POST("show/:uid/games/json",
		gtype.SetTypes(),
		game.GetFiltered(gtype.All),
		game.JSONIndexAction,
	)

	g1.POST("edit/:uid/games/json",
		// user.RequireLogin(),
		gtype.SetTypes(),
		game.GetFiltered(gtype.All),
		game.JSONIndexAction,
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
		gtype.SetTypes(),
		client.Index,
	)

	// json data for Index
	g2.POST("/json",
		user.RequireAdmin,
		gtype.SetTypes(),
		client.User.FetchAll,
		client.JSON,
	)

	return engine
}

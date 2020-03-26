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

type server struct {
	*datastore.Client
}

func NewClient(dsClient *datastore.Client) server {
	return server{Client: dsClient}
}

func (svr server) AddRoutes(prefix string, engine *gin.Engine) *gin.Engine {
	// User Group
	g1 := engine.Group(prefix)
	g1.GET("/new",
		gtype.SetTypes(),
		svr.new,
	)

	// Create
	g1.POST("",
		svr.create(prefix),
	)

	// Show User
	g1.GET("show/:uid",
		user.Fetch,
		stats.Fetch(user.From),
		gtype.SetTypes(),
		svr.show,
	)

	// Edit User
	g1.GET("edit/:uid",
		user.Fetch,
		stats.Fetch(user.From),
		gtype.SetTypes(),
		svr.edit,
	)

	// Update User
	g1.POST("update/:uid",
		user.Fetch,
		gtype.SetTypes(),
		svr.update,
	)

	// User Ratings
	g1.POST("show/:uid/ratings/json",
		user.Fetch,
		rating.JSONIndexAction,
	)

	g1.POST("edit/:uid/ratings/json",
		user.Fetch,
		rating.JSONIndexAction,
	)

	// User Games
	g1.POST("show/:uid/games/json",
		gtype.SetTypes(),
		game.GetFiltered(gtype.All),
		game.JSONIndexAction,
	)

	g1.POST("edit/:uid/games/json",
		gtype.SetTypes(),
		game.GetFiltered(gtype.All),
		game.JSONIndexAction,
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
		svr.index,
	)

	// json data for Index
	g2.POST("/json",
		user.RequireAdmin,
		gtype.SetTypes(),
		user.FetchAll,
		svr.json,
	)

	return engine
}

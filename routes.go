package user_controller

import (
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

func AddRoutes(prefix string, engine *gin.Engine) {
	// User Group
	g1 := engine.Group(prefix)
	g1.GET("/new",
		// user.RequireLogin(),
		gtype.SetTypes(),
		NewAction,
	)

	// Create
	g1.POST("",
		// user.RequireLogin(),
		Create(prefix),
	)

	// Show User
	g1.GET("show/:uid",
		user.Fetch,
		stats.Fetch(user.From),
		gtype.SetTypes(),
		Show,
	)

	// Edit User
	g1.GET("edit/:uid",
		// user.RequireLogin(),
		user.Fetch,
		stats.Fetch(user.From),
		gtype.SetTypes(),
		Edit,
	)

	// Update User
	g1.POST("update/:uid",
		// user.RequireLogin(),
		user.Fetch,
		gtype.SetTypes(),
		Update,
	)

	// User Ratings
	g1.POST("show/:uid/ratings/json",
		user.Fetch,
		rating.JSONIndexAction,
	)

	g1.POST("edit/:uid/ratings/json",
		// user.RequireLogin(),
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
		// user.RequireLogin(),
		gtype.SetTypes(),
		game.GetFiltered(gtype.All),
		game.JSONIndexAction,
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
		Index,
	)

	// json data for Index
	g2.POST("/json",
		user.RequireAdmin,
		gtype.SetTypes(),
		user.FetchAll,
		JSON,
	)
}

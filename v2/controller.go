package user_controller

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/sn/v2"
	stats "github.com/SlothNinja/user-stats/v2"
	"github.com/SlothNinja/user/v2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	welcomePath = "/welcome"
	userNewPath = "/user/new"
	homePath    = "/"
	msgEnter    = "Entering"
	msgExit     = "Exiting"
	uidParam    = "uid"
)

func (client Client) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "user/index", gin.H{
		"Context": c,
		"CUser":   user.CurrentFrom(c),
	})
}

func (client Client) Show(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	u := user.From(c)
	c.HTML(http.StatusOK, "user/show", gin.H{
		"Context": c,
		"User":    u,
		"CUser":   user.CurrentFrom(c),
		"IsAdmin": user.IsAdmin(c),
		"Stats":   stats.Fetched(c),
	})
}

func (client Client) Edit(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	u := user.From(c)
	c.HTML(http.StatusOK, "user/edit", gin.H{
		"Context": c,
		"User":    u,
		"CUser":   user.CurrentFrom(c),
		"IsAdmin": user.IsAdmin(c),
		"Stats":   stats.Fetched(c),
	})
}

func (client Client) Current(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	cu, err := client.User.Current(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(http.StatusOK, gin.H{"cu": cu})
}

type jUserIndex struct {
	Data            []*jUser `json:"data"`
	Draw            int      `json:"draw"`
	RecordsTotal    int64    `json:"recordsTotal"`
	RecordsFiltered int64    `json:"recordsFiltered"`
}

type omit *struct{}

type jUser struct {
	IntID         int64         `json:"id"`
	StringID      string        `json:"sid"`
	OldID         int64         `json:"oldid"`
	GoogleID      string        `json:"googleid"`
	Name          string        `json:"name"`
	Email         string        `json:"email"`
	Gravatar      template.HTML `json:"gravatar"`
	Joined        time.Time     `json:"joined"`
	Updated       time.Time     `json:"updated"`
	OmitCreatedAt omit          `json:"createdat,omitempty"`
	OmitUpdatedAt omit          `json:"updatedat,omitempty"`
}

func toUserTable(c *gin.Context, us []*user.User) (*jUserIndex, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	table := new(jUserIndex)
	l := len(us)
	table.Data = make([]*jUser, l)

	for i, u := range us {
		table.Data[i] = &jUser{
			IntID:    u.ID(),
			StringID: "",
			OldID:    0,
			Name:     u.Name,
			Email:    u.Email,
			Gravatar: user.Gravatar(u),
			Joined:   u.CreatedAt,
			Updated:  u.UpdatedAt,
		}
	}

	draw, err := strconv.Atoi(c.PostForm("draw"))
	if err != nil {
		return nil, err
	}

	table.Draw = draw
	table.RecordsTotal = user.CountFrom(c)
	table.RecordsFiltered = user.CountFrom(c)
	return table, nil
}

func (client Client) JSON(uidParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf(msgEnter)
		defer log.Debugf(msgExit)

		uid, err := getUID(c, uidParam)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		u, err := client.User.Get(c, uid)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": u})
	}
}

// func (client Client) JSON(c *gin.Context) {
// 	log.Debugf(msgEnter)
// 	defer log.Debugf(msgExit)
//
// 	us := user.UsersFrom(c)
// 	data, err := toUserTable(c, us)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, fmt.Sprintf("%v", err))
// 		return
// 	}
// 	c.JSON(http.StatusOK, data)
// }

func (client Client) NewAction(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	cu := user.CurrentFrom(c)
	if cu != nil {
		sn.JErr(c, fmt.Errorf("user %s present, no need for new one: %w",
			cu.Name, sn.ErrValidation))
		return
	}

	session := sessions.Default(c)
	token, ok := user.SessionTokenFrom(session)
	if !ok {
		sn.JErr(c, fmt.Errorf("missing token."))
		return
	}

	u := user.New(0)
	u.Data = token.Data
	u.Key = token.Key
	u.EmailReminders = true
	u.GravType = "monsterid"

	c.JSON(http.StatusOK, gin.H{"user": u})
}

func (client Client) Create(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cu := user.CurrentFrom(c)
		if cu != nil {
			sn.JErr(c, fmt.Errorf("user %s present, no need for new one: %w",
				cu.Name, sn.ErrValidation))
			return
		}

		u1 := user.New(0)
		err := c.ShouldBind(u1)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		u := user.New(0)
		u.Name = u1.Name
		u.LCName = strings.ToLower(u1.Name)
		u.Email = u1.Email
		u.EmailReminders = true
		u.EmailNotifications = u1.EmailNotifications
		u.GravType = u1.GravType

		uniq, err := client.User.NameIsUnique(c, u.Name)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		if !uniq {
			sn.JErr(c, fmt.Errorf("%q is not a unique user name: %w",
				u.LCName, sn.ErrValidation))
			return
		}

		ks, err := client.DS.AllocateIDs(c, []*datastore.Key{u.Key})
		if err != nil {
			sn.JErr(c, err)
			return
		}

		u.Key = ks[0]

		session := sessions.Default(c)
		token, ok := user.SessionTokenFrom(session)
		if !ok {
			sn.JErr(c, fmt.Errorf("missing token"))
			return
		}

		oaid := user.GenOAuthID(token.Sub)
		oa := user.NewOAuth(oaid)
		oa.ID = u.ID()
		oa.UpdatedAt = time.Now()
		_, err = client.DS.RunInTransaction(c, func(tx *datastore.Transaction) error {
			ks := []*datastore.Key{oa.Key, u.Key}
			es := []interface{}{&oa, u}
			_, err := tx.PutMulti(ks, es)
			return err

		})

		if err != nil {
			sn.JErr(c, err)
			return
		}

		token.Data = u.Data
		token.Loaded = true
		token.Key = u.Key

		err = token.SaveTo(session)
		if err != nil {
			sn.JErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":    u,
			"message": "account created for " + u.Name,
		})
	}
}

func showPath(prefix string, id int64) string {
	return fmt.Sprintf("%s/show/%d", prefix, id)
}

func (client Client) Update(uidParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf(msgEnter)
		defer log.Debugf(msgExit)

		uid, err := getUID(c, uidParam)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		u, err := client.User.Get(c, uid)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		err = client.User.Update(c, u)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		_, err = client.DS.Put(c, u.Key, u)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		session := sessions.Default(c)
		token, _ := user.SessionTokenFrom(session)
		token.Data = u.Data
		token.Loaded = true
		token.Key = u.Key

		err = token.SaveTo(session)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": u})
	}
}

// func GamesIndex(c *gin.Context) {
// 	log.Debugf(msgEnter)
// 	defer log.Debugf(msgExit)
//
// 	if status := game.StatusFrom(c); status != game.NoStatus {
// 		c.HTML(200, "shared/games_index", gin.H{})
// 	} else {
// 		c.HTML(200, "user/games_index", gin.H{})
// 	}
// }

func getUID(c *gin.Context, uidParam string) (int64, error) {
	return strconv.ParseInt(c.Param(uidParam), 10, 64)
}

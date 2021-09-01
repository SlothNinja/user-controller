package user_controller

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	welcomePath = "/welcome"
	userNewPath = "/user/new"
	homePath    = "/"
	msgEnter    = "Entering"
	msgExit     = "Exiting"
)

func isAdmin(u *user.User) bool {
	return u != nil && u.Admin
}

func (client *Client) Index(c *gin.Context) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)
	cu, err := client.User.Current(c)
	if err != nil {
		client.Log.Debugf(err.Error())
	}
	c.HTML(http.StatusOK, "user/index", gin.H{
		"Context": c,
		"CUser":   cu,
	})
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
	table := new(jUserIndex)
	l := len(us)
	table.Data = make([]*jUser, l)

	for i, u := range us {
		table.Data[i] = &jUser{
			IntID:    u.ID(),
			StringID: "",
			OldID:    0,
			GoogleID: u.GoogleID,
			Name:     u.Name,
			Email:    u.Email,
			Gravatar: user.Gravatar(u, "80"),
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

func (client *Client) JSON(uidParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		cu, err := client.User.Current(c)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		uid, err := getUID(c, uidParam)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		if cu.ID() == uid {
			c.JSON(http.StatusOK, gin.H{
				"cu":   cu,
				"user": cu,
			})
			return
		}

		u, err := client.User.Get(c, uid)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"cu":   cu,
			"user": u,
		})
	}
}

func (client *Client) NewAction(c *gin.Context) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	cu, err := client.User.Current(c)
	if cu != nil {
		sn.JErr(c, fmt.Errorf("you already have an account"))
		return
	}

	session := sessions.Default(c)
	u, err := user.NewFrom(session)
	if err != nil {
		client.Log.Errorf(err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
	}

	u.EmailReminders = true
	u.EmailNotifications = true
	u.GravType = "monsterid"
	hash, err := user.EmailHash(u.Email)
	if err != nil {
		client.Log.Warningf("email hash error: %v", err)
		c.Redirect(http.StatusSeeOther, homePath)
	}
	u.EmailHash = hash

	if !isAdmin(cu) {
		cu = u
	}

	c.JSON(http.StatusOK, gin.H{
		"cu":   cu,
		"user": u,
	})
}

func (client *Client) Create(c *gin.Context) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	cu, err := client.User.Current(c)
	if cu != nil {
		sn.JErr(c, fmt.Errorf("you already have an account"))
		return
	}

	session := sessions.Default(c)
	token, ok := user.SessionTokenFrom(session)
	if !ok {
		client.Log.Warningf("missing token")
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	if token.ID != 0 {
		client.Log.Warningf("user present, no need for new one. token: %#v", token)
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	u := user.New(0)
	err = c.ShouldBind(u)
	if err != nil {
		sn.JErr(c, err)
		return
	}

	err = client.User.Update(c, u, u, u)
	if err != nil {
		sn.JErr(c, err)
		return
	}

	ks, err := client.User.AllocateIDs(c, []*datastore.Key{u.Key})
	if err != nil {
		client.Log.Errorf(err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	u.Key = ks[0]
	u.LCName = strings.ToLower(u.Name)

	oaid := user.GenOAuthID(token.Sub)
	oa := user.NewOAuth(oaid)
	oa.ID = u.ID()
	oa.UpdatedAt = time.Now()
	_, err = client.User.RunInTransaction(c, func(tx *datastore.Transaction) error {
		ks := []*datastore.Key{oa.Key, u.Key}
		es := []interface{}{&oa, u}
		_, err := tx.PutMulti(ks, es)
		return err

	})

	if err != nil {
		client.Log.Errorf(err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	if !isAdmin(cu) {
		cu = u
		token.ID = u.Key.ID

		err = token.SaveTo(session)
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}

	}

	c.JSON(http.StatusOK, gin.H{
		"cu":      cu,
		"user":    u,
		"message": "account created for " + u.Name,
	})
}

func (client *Client) Update(uidParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		cu, err := client.User.Current(c)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		uid, err := getUID(c, uidParam)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		u := cu
		if cu.ID() != uid {
			u, err = client.User.Get(c, uid)
			if err != nil {
				sn.JErr(c, err)
				return
			}
		}

		obj := user.New(0)
		err = c.ShouldBind(obj)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		err = client.User.Update(c, cu, u, obj)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		_, err = client.User.Put(c, u.Key, u)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		if !isAdmin(cu) {
			cu = u
			session := sessions.Default(c)
			token, _ := user.SessionTokenFrom(session)
			token.ID = u.Key.ID

			err = token.SaveTo(session)
			if err != nil {
				sn.JErr(c, err)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"cu":   cu,
			"user": u,
		})
	}
}

func GamesIndex(c *gin.Context) {
	if status := game.StatusFrom(c); status != game.NoStatus {
		c.HTML(200, "shared/games_index", gin.H{})
	} else {
		c.HTML(200, "user/games_index", gin.H{})
	}
}

func (client *Client) Current(c *gin.Context) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	cu, err := client.User.Current(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(http.StatusOK, gin.H{"cu": cu})
}

func getUID(c *gin.Context, uidParam string) (int64, error) {
	return strconv.ParseInt(c.Param(uidParam), 10, 64)
}

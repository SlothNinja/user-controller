package user_controller

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/sn/v2"
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

func (client Client) Index(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)
	cu, err := user.CurrentFrom(c)
	if err != nil {
		log.Debugf(err.Error())
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
			GoogleID: u.GoogleID,
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

func (client Client) NewAction(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)
	cu, err := user.CurrentFrom(c)
	if err != nil {
		log.Debugf(err.Error())
	}

	if cu != nil && cu.ID() != 0 {
		log.Warningf("user present, no need for new one.\n key: %#v\n user %#v\n", cu.Key, cu)
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	cu.EmailReminders = true
	cu.GravType = "monsterid"

	c.JSON(http.StatusOK, gin.H{"user": cu})
}

func (client Client) Create(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	session := sessions.Default(c)
	token, ok := user.SessionTokenFrom(session)
	if !ok {
		log.Warningf("missing token")
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	if token.Key.ID != 0 {
		log.Warningf("user present, no need for new one. token: %#v", token)
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	u := user.New(0)
	u.Data = token.Data

	err := client.User.Update(c, u)
	if err != nil {
		sn.JErr(c, err)
		return
	}

	ks, err := client.DS.AllocateIDs(c, []*datastore.Key{u.Key})
	if err != nil {
		log.Errorf(err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	u.Key = ks[0]
	u.LCName = strings.ToLower(u.Name)
	log.Debugf("u.Key: %#v", u.Key)

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
		log.Errorf(err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	token.Key = u.Key
	token.Data = u.Data

	err = token.SaveTo(session)
	if err != nil {
		log.Errorf(err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    u,
		"message": "account created for " + u.Name,
	})
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
		token.Key = u.Key

		err = token.SaveTo(session)
		if err != nil {
			sn.JErr(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": u})
	}
}

func GamesIndex(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	if status := game.StatusFrom(c); status != game.NoStatus {
		c.HTML(200, "shared/games_index", gin.H{})
	} else {
		c.HTML(200, "user/games_index", gin.H{})
	}
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

func getUID(c *gin.Context, uidParam string) (int64, error) {
	return strconv.ParseInt(c.Param(uidParam), 10, 64)
}

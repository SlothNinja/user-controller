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
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	name "github.com/SlothNinja/user-name"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	welcomePath = "/welcome"
	userNewPath = "/user/new"
	homePath    = "/"
)

func (client Client) index(c *gin.Context) {
	c.HTML(http.StatusOK, "user/index", gin.H{
		"Context":   c,
		"VersionID": sn.VersionID(),
		"CUser":     user.CurrentFrom(c),
	})
}

func (client Client) show(c *gin.Context) {
	u, err := client.User.GetUser(c)
	if err != nil {
		log.Errorf(err.Error())
		restful.AddErrorf(c, err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	s, err := client.Stats.ByUser(c, u)
	if err != nil {
		log.Errorf(err.Error())
		restful.AddErrorf(c, err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	c.HTML(http.StatusOK, "user/show", gin.H{
		"Context":   c,
		"VersionID": sn.VersionID(),
		"User":      u,
		"CUser":     user.CurrentFrom(c),
		"IsAdmin":   s,
		"Stats":     stats.Fetched(c),
	})
}

func (client Client) edit(c *gin.Context) {
	u, err := client.User.GetUser(c)
	if err != nil {
		log.Errorf(err.Error())
		restful.AddErrorf(c, err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	s, err := client.Stats.ByUser(c, u)
	if err != nil {
		log.Errorf(err.Error())
		restful.AddErrorf(c, err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	c.HTML(http.StatusOK, "user/edit", gin.H{
		"Context":   c,
		"VersionID": sn.VersionID(),
		"User":      u,
		"CUser":     user.CurrentFrom(c),
		"IsAdmin":   user.IsAdmin(c),
		"Stats":     s,
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

func toUserTable(c *gin.Context, us []interface{}) (table *jUserIndex, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	table = new(jUserIndex)
	l := len(us)
	table.Data = make([]*jUser, l)

	var (
		u  *user.User
		nu *user.NUser
		ok bool
	)

	for i, uinf := range us {
		if u, ok = uinf.(*user.User); ok {
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
		} else if nu, ok = uinf.(*user.NUser); ok {
			table.Data[i] = &jUser{
				IntID:    0,
				StringID: nu.ID(),
				OldID:    nu.OldID,
				GoogleID: nu.GoogleID,
				Name:     nu.Name,
				Email:    nu.Email,
				Gravatar: user.NGravatar(nu),
				Joined:   nu.CreatedAt,
				Updated:  nu.UpdatedAt,
			}
		} else {
			err = fmt.Errorf("not user")
			return
		}
	}

	if draw, err := strconv.Atoi(c.PostForm("draw")); err != nil {
		return nil, err
	} else {
		table.Draw = draw
	}
	table.RecordsTotal = user.CountFrom(c)
	table.RecordsFiltered = user.CountFrom(c)
	return
}

func (client Client) json(c *gin.Context) {
	us := user.UsersFrom(c)
	if data, err := toUserTable(c, us); err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("%v", err))
	} else {
		c.JSON(http.StatusOK, data)
	}
}

func (client Client) new(c *gin.Context) {
	cu := user.CurrentFrom(c)
	if cu != nil {
		log.Warningf("user %#v present, no need for new one", cu)
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	session := sessions.Default(c)
	token, ok := user.SessionTokenFrom(session)
	if !ok {
		log.Warningf("missing token")
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	u := user.New(c, token.ID())
	u.Data = token.User.Data

	c.HTML(http.StatusOK, "user/new", gin.H{
		"Context": c,
		"User":    u,
	})
}

func (client Client) create(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cu := user.CurrentFrom(c)
		if cu != nil {
			log.Warningf("user %#v present, no need for new one", cu)
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}

		session := sessions.Default(c)
		token, ok := user.SessionTokenFrom(session)
		if !ok {
			log.Warningf("missing token")
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}

		// // Fell through 'switch' thus err == user.ErrNotFound
		u := user.New(c, 0)
		u.Name = strings.Split(c.PostForm("user-name"), "@")[0]
		u.LCName = strings.ToLower(u.Name)
		u.Email = token.Email

		uniq, err := client.User.NameIsUnique(c, u.Name)
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}

		if !uniq {
			err = fmt.Errorf("%q is not a unique user name.", u.LCName)
			restful.AddErrorf(c, err.Error())
			log.Warningf(err.Error())
			c.Redirect(http.StatusSeeOther, userNewPath)
			return
		}

		ks, err := client.AllocateIDs(c, []*datastore.Key{u.Key})
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}

		u.Key = ks[0]

		oaid := user.GenOAuthID(token.Sub)
		oa := user.NewOAuth(oaid)
		oa.ID = u.ID()
		oa.UpdatedAt = time.Now()
		_, err = client.RunInTransaction(c, func(tx *datastore.Transaction) error {
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
		token.User = u
		token.Loaded = true
		err = token.SaveTo(session)
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
		c.Redirect(http.StatusSeeOther, showPath(prefix, u.ID()))
	}
}

func showPath(prefix string, id int64) string {
	return fmt.Sprintf("%s/show/%d", prefix, id)
}

func (client Client) update(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	// Get Resource
	uid, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil {
		log.Errorf(err.Error())
		c.Abort()
		return
	}

	u := user.New(c, uid)
	err = client.Get(c, u.Key, u)
	if err != nil {
		log.Errorf(err.Error())
		c.Abort()
		return
	}

	oldName := name.New(u.LCName)
	err = client.User.Update(c, u)
	if err != nil {
		log.Errorf(err.Error())
		restful.AddErrorf(c, err.Error())
		route := fmt.Sprintf("/user/show/%s", c.Param("uid"))
		c.Redirect(http.StatusSeeOther, route)
		return
	}
	newName := name.New(u.LCName)
	newName.GoogleID = u.GoogleID

	_, err = client.RunInTransaction(c, func(tx *datastore.Transaction) error {
		nu := user.ToNUser(c, u)
		entities := []interface{}{u, nu, newName, oldName}
		ks := []*datastore.Key{u.Key, nu.Key, newName.Key, oldName.Key}
		_, err := tx.PutMulti(ks, entities)
		if err != nil {
			return err
		}

		return tx.Delete(oldName.Key)
	})

	switch {
	case sn.IsVError(err):
		restful.AddErrorf(c, err.Error())
	case err != nil:
		log.Errorf(err.Error())
	}

	route := fmt.Sprintf("/user/show/%s", c.Param("uid"))
	c.Redirect(http.StatusSeeOther, route)
}

func (client Client) gamesIndex(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	status := game.StatusFrom(c)
	if status != game.NoStatus {
		c.HTML(200, "shared/games_index", gin.H{})
		return
	}
	c.HTML(200, "user/games_index", gin.H{})
}

package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

func GetUsers(username string, r render.Render) {
	var users []User
	db.Order("id desc").Find(&users)

	data := map[string]interface{}{"username": username, "users": users}
	r.HTML(200, "user", data)
}
func EditUsers(req *http.Request, user User, r render.Render) {
	req.ParseForm()
	user.IsAdmin = req.PostForm.Get("isadmin") == "on"
	if user.Name == "" || user.Password == "" {
		sendFailMsg(r, "保存失败，用户名或密码不能为空.", "")
		return
	}
	if user.Avatar != "" {
		if !strings.HasSuffix(user.Avatar, ".jpg") &&
			!strings.HasSuffix(user.Avatar, ".jpeg") &&
			!strings.HasSuffix(user.Avatar, ".gif") &&
			!strings.HasSuffix(user.Avatar, ".png") {
			sendFailMsg(r, "保存失败，头像只支持gif,jpg,png格式.", "")
			return
		}
	}
	user.CreatedAt = time.Now()
	err := db.Save(&user).Error
	if user.Id > 0 {
		sendSuccessMsg(r, "")
	} else {
		sendFailMsg(r, "保存失败."+err.Error(), "")
	}
}
func DeleteUser(params martini.Params, r render.Render) {
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "id参数错误",
		})
		return
	}

	err = db.Delete(&User{Id: id}).Error
	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "删除出错." + err.Error(),
		})
		return
	}
	r.JSON(200, ActionMessage{
		Success: true,
		Message: "成功",
	})
}
func ToggleSetAdmin(params martini.Params, r render.Render) {
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "id参数错误",
		})
		return
	}

	action := params["action"]
	var user User
	if action == "enable" {
		db.First(&user, id)
		user.IsAdmin = true
		err = db.Debug().Save(user).Error
	} else {
		db.First(&user, id)
		user.IsAdmin = false
		err = db.Save(user).Error
	}

	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "保存出错." + err.Error(),
		})
		return
	}
	r.JSON(200, ActionMessage{
		Success: true,
		Message: "成功",
	})
}
func UserSetting(username string, r render.Render) {
	var user User
	db.First(&user, User{Name: username})

	data := map[string]interface{}{"username": username, "user": user}
	r.HTML(200, "setting", data)
}

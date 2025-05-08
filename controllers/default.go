package controllers

import (
	beego "github.com/beego/beego/v2/server/web"
)

type MainController struct {
	beego.Controller
}

func (c *MainController) Get() {
	c.Data["Website"] = "beego"
	c.Data["Email"] = "lb1144466883@163com"
	c.TplName = "index.html"
}

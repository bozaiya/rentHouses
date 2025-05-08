package controllers

import (
	"encoding/base64"
	"errors"
	"github.com/beego/beego/v2/adapter/orm"
	"github.com/beego/beego/v2/adapter/utils"
	orm2 "github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"log"
	"regexp"
	"rentHouses/models"
	"strconv"
	"time"
)

type AdministerController struct {
	beego.Controller
}

func GetAdministerUser(c *beego.Controller) string {
	userName := c.GetSession("userName")
	if userName == nil {
		c.Data["userName"] = ""
	} else {
		c.Data["userName"] = userName.(string)
		return userName.(string)
	}
	return ""
}

// 展示管理员注册页面
func (c *AdministerController) ShowAdministerRegister() {
	c.TplName = "administer_register.html"
}

// 处理管理员注册数据
func (c *AdministerController) HandleAdministerRegister() {
	// 1、获取数据
	userName := c.GetString("user_name")
	password := c.GetString("pwd")
	cpwd := c.GetString("cpwd")
	email := c.GetString("email")
	// 2、校验数据
	if userName == "" || password == "" || cpwd == "" || email == "" {
		log.Println("数据不完整,请重新注册")
		return
	}
	if cpwd != password {
		log.Println("两次输入密码不一致")
		c.Data["error"] = "两次输入密码不一致,请重新输入"
		c.TplName = "administer_register.html"
		return
	}
	reg, _ := regexp.Compile("^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$")
	res := reg.FindString(email)
	if res == "" {
		c.Data["error"] = "邮箱格式不正确，请重新输入"
		c.TplName = "administer_register.html"
		return
	}
	// 3、处理数据
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	user.PassWord = password
	user.Email = email
	user.Power = 1

	_, err := o.Insert(&user)
	if err != nil {
		c.Data["error"] = "注册失败,请更换数据注册"
		c.TplName = "administer_register.html"
		return
	}

	// 发送邮件
	emailConfig := `{"username":"1144466883@qq.com","password":"lcbdzhpjdctmgfjg","host":"smtp.qq.com","port":587}`
	emailConn := utils.NewEMail(emailConfig)
	emailConn.From = "1144466883@qq.com"
	emailConn.To = []string{email}
	emailConn.Subject = "省心租管理员注册"
	// 发送请求，即 /active
	emailConn.Text = "【省心租】激活管理员用户，请在浏览器输入：192.168.117.132:8080/active?id=" + strconv.Itoa(user.Id) + " 进行激活,谢谢您的使用"
	err = emailConn.Send()
	if err != nil {
		log.Println("发送失败", err)
		c.TplName = "administer_register.html"
		return
	}

	// 4、返回视图
	c.Redirect("/administerLogin", 302)
}

// 管理员激活业务
func (c *AdministerController) HandleAdministerActive() {
	id, err := c.GetInt("id")
	if err != nil {
		c.Data["error"] = "要激活的用户不存在"
		c.TplName = "administer_register.html"
		return
	}

	o := orm.NewOrm()
	user := models.User{}
	user.Id = id
	err = o.Read(&user)
	if err != nil {
		c.Data["error"] = "要激活的用户不存在"
		c.TplName = "administer_register.html"
		return
	}

	user.Active = true
	_, err = o.Update(&user)
	// 返回视图
	c.Redirect("/administerLogin", 302)
}

// 展示管理员登录页面
func (c *AdministerController) ShowAdministerLogin() {
	// 记住用户名获取用户名操作
	userName := c.Ctx.GetCookie("userName")
	temp, _ := base64.StdEncoding.DecodeString(userName)
	if string(temp) == "" {
		c.Data["userName"] = ""
		c.Data["checked"] = ""
	} else {
		c.Data["userName"] = string(temp)
		c.Data["checked"] = "checked"
	}
	c.TplName = "administer_login.html"
}

// 管理员登陆
func (c *AdministerController) HandleAdministerLogin() {
	// 1、获取数据
	userName := c.GetString("username")
	password := c.GetString("pwd")
	// 2、数据校验
	if userName == "" || password == "" {
		c.Data["errmsg"] = "登陆数据不能为空"
		c.TplName = "administer_login.html"
		return
	}
	// 3、执行操作
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	err := o.Read(&user, "Name")

	if user.Power != 1 {
		c.Data["errmsg"] = "该用户不存在"
		c.TplName = "administer_login.html"
	}
	if err != nil {
		c.Data["errmsg"] = "用户名或密码错误"
		c.TplName = "administer_login.html"
		return
	}
	if user.PassWord != password {
		c.Data["errmsg"] = "用户名或密码错误"
		c.TplName = "administer_login.html"
		return
	}
	if user.Active == false {
		c.Data["errmsg"] = "未激活，请激活后登陆"
		c.TplName = "administer_login.html"
		return
	}

	user.Time = time.Now().Format("2006-01-02 15:04:05")
	log.Println(user.Time)
	_, err = o.Update(&user)
	if err != nil {
		log.Println("更新最后一次登陆时间失败")
		return
	}

	// 记住用户名操作
	remember := c.GetString("remember")
	if remember == "on" {
		temp := base64.StdEncoding.EncodeToString([]byte(userName))
		c.Ctx.SetCookie("userName", temp, 24*3600*7)
	} else {
		c.Ctx.SetCookie("userName", userName, -1)
	}

	// 4、跳转到操作界面
	err = c.SetSession("userName", userName)
	if err != nil {
		return
	}
	c.Redirect("/administer/showUserModule", 302)
}

// 管理员退出
func (c *AdministerController) HandleAdministerLogout() {
	err := c.DelSession("userName")
	if err != nil {
		return
	}
	c.Redirect("/administerLogin", 302)
}

// 展示用户模块
func (c *AdministerController) ShowUserModule() {
	userName := GetAdministerUser(&c.Controller)
	c.Data["userName"] = userName

	/* 1、用户模块 */
	o := orm.NewOrm()
	var users []models.User

	_, err := o.QueryTable("User").Filter("Power", 0).OrderBy("-Time").All(&users)
	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("没有普通用户")
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询出错")
		c.Redirect("/administer/showUserModule", 302)
		return
	}

	c.Data["users"] = users
	c.Layout = "administerLayout.html"
	c.TplName = "administer_user.html"
}

// 展示评论模块
func (c *AdministerController) ShowCommentModule() {
	userName := GetAdministerUser(&c.Controller)
	c.Data["userName"] = userName

	/* 2、评论模块 */
	var orderGoods []models.OrderGoods
	o := orm.NewOrm()
	_, err := o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").RelatedSel("OrderInfo__User").OrderBy("-OrderInfo__Time").Limit(30, 0).All(&orderGoods)
	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("没有订单数据")
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询订单出错：", err)
		return
	}

	c.Data["orderGoods"] = orderGoods
	c.Layout = "administerLayout.html"
	c.TplName = "administer_comment.html"
}

// 展示住房模块
func (c *AdministerController) ShowHouseModule() {
	userName := GetAdministerUser(&c.Controller)
	c.Data["userName"] = userName

	/* 3、住房管理模块 */
	var goodsSKU []models.GoodsSKU
	o := orm.NewOrm()
	_, err := o.QueryTable("GoodsSKU").RelatedSel("User").OrderBy("-Time").Limit(30, 0).All(&goodsSKU)
	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("没有住房数据")
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询住房出错：", err)
		return
	}

	c.Data["goodsSKU"] = goodsSKU
	c.Layout = "administerLayout.html"
	c.TplName = "administer_house.html"
}

// 展示订单模块
func (c *AdministerController) ShowOrderModule() {
	userName := GetAdministerUser(&c.Controller)
	c.Data["userName"] = userName

	/* 4、订单管理模块 */
	// 获取订单表数据
	o := orm.NewOrm()
	var orderInfos []models.OrderInfo
	_, err := o.QueryTable("OrderInfo").RelatedSel("Address").OrderBy("-Time").Limit(30, 0).All(&orderInfos)
	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("没有订单数据")
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询订单出错：", err)
		c.Redirect("/administer/showOrderModule", 302)
		return
	}

	goodsBuffer := make([]map[string]interface{}, len(orderInfos))

	for index, orderInfo := range orderInfos {
		var orderGoods []models.OrderGoods
		_, err = o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").RelatedSel("OrderInfo__Address").Filter("OrderInfo__Id", orderInfo.Id).All(&orderGoods)
		if err != nil {
			log.Println("订单商品表查询失败")
			c.Redirect("/administer/showOrderModule", 302)
			return
		}
		temp := make(map[string]interface{})
		temp["orderInfo"] = orderInfo
		temp["orderGoods"] = orderGoods

		goodsBuffer[index] = temp
	}

	c.Data["goodsBuffer"] = goodsBuffer

	c.Layout = "administerLayout.html"
	c.TplName = "administer_order.html"

}

// 展示数据统计模块
func (c *AdministerController) ShowStatsModule() {
	userName := GetAdministerUser(&c.Controller)
	c.Data["userName"] = userName

	/* 5、数据统计模块 */
	o := orm.NewOrm()

	// 1、用户数目统计
	// 用户总数
	userCount, err := o.QueryTable("User").Filter("Power", 0).Count()
	if err != nil {
		log.Println("用户总数统计失败", err)
		return
	}
	c.Data["userCount"] = userCount

	// 已激活用户数量
	aUserCount, err := o.QueryTable("User").Filter("Power", 0).Filter("Active", 1).Count()
	if err != nil {
		log.Println("激活用户数目统计失败", err)
		return
	}
	c.Data["aUserCount"] = aUserCount
	// 未激活用户数量
	c.Data["nUserCount"] = userCount - aUserCount

	// 2、住房数目统计
	// 住房总数
	goodsCount, err := o.QueryTable("GoodsSKU").Count()
	if err != nil {
		log.Println("住房总数统计失败", err)
		return
	}
	c.Data["goodsCount"] = goodsCount

	// 待确认订单数目
	nConfirmCount, err := o.QueryTable("OrderInfo").Filter("ConfirmStatus", 0).Count()
	if err != nil {
		log.Println("待确认订单数统计失败", err)
		return
	}
	c.Data["nConfirmCount"] = nConfirmCount

	// 待支付订单数目
	nPayCount, err := o.QueryTable("OrderInfo").Filter("ConfirmStatus", 1).Filter("OrderStatus", 0).Count()
	if err != nil {
		log.Println("待支付订单数统计失败", err)
		return
	}
	c.Data["nPayCount"] = nPayCount

	// 3、总成交量和总成交额
	var orderInfos []models.OrderInfo
	payCount, _ := o.QueryTable("OrderInfo").Filter("ConfirmStatus", 1).Filter("OrderStatus", 1).All(&orderInfos)

	// 总成交量
	c.Data["payCount"] = payCount
	// 总成交额
	payPrice := 0
	for _, orderInfo := range orderInfos {
		payPrice += orderInfo.TotalPrice
	}
	c.Data["payPrice"] = payPrice
	// 服务费总额
	c.Data["servePrice"] = int(float64(payPrice) / 11)

	now := time.Now().Local()
	// 4、计算年度总成交额
	// 年度开始时间（当年1月1日）
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	// 年度结束时间（下一年1月1日前一纳秒）
	yearEnd := yearStart.AddDate(1, 0, 0).Add(-time.Nanosecond)
	var yearOrders []models.OrderInfo
	_, _ = o.QueryTable("OrderInfo").Filter("ConfirmStatus", 1).Filter("OrderStatus", 1).Filter("Time__gte", yearStart).Filter("Time__lte", yearEnd).All(&yearOrders)

	yearPrice := 0
	for _, yearOrder := range yearOrders {
		yearPrice += yearOrder.TotalPrice
	}
	c.Data["yearPrice"] = yearPrice

	// 5、计算该季度成交额
	year := now.Year()
	// 根据当前月份计算季度
	var startMonth time.Month
	switch now.Month() {
	case time.January, time.February, time.March:
		startMonth = time.January
	case time.April, time.May, time.June:
		startMonth = time.April
	case time.July, time.August, time.September:
		startMonth = time.July
	default:
		startMonth = time.October
	}

	// 构造季度时间范围
	quarterStart := time.Date(year, startMonth, 1, 0, 0, 0, 0, now.Location())
	quarterEnd := quarterStart.AddDate(0, 3, 0).Add(-time.Nanosecond)
	var quarterOrders []models.OrderInfo
	_, _ = o.QueryTable("OrderInfo").Filter("ConfirmStatus", 1).Filter("OrderStatus", 1).Filter("Time__gte", quarterStart).Filter("Time__lte", quarterEnd).All(&quarterOrders)
	quarterPrice := 0
	for _, quarterOrder := range quarterOrders {
		quarterPrice += quarterOrder.TotalPrice
	}
	c.Data["quarterPrice"] = quarterPrice

	// 6、计算月成交额
	// 计算当月时间范围
	MonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	MonthEnd := MonthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
	var monthOrders []models.OrderInfo
	_, _ = o.QueryTable("OrderInfo").Filter("ConfirmStatus", 1).Filter("OrderStatus", 1).Filter("Time__gte", MonthStart).Filter("Time__lte", MonthEnd).All(&monthOrders)
	monthPrice := 0
	for _, monthOrder := range monthOrders {
		monthPrice += monthOrder.TotalPrice
	}
	c.Data["monthPrice"] = monthPrice

	// 7、计算当日成交额
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24*time.Hour - time.Nanosecond)
	var dayOrders []models.OrderInfo
	_, _ = o.QueryTable("OrderInfo").Filter("ConfirmStatus", 1).Filter("OrderStatus", 1).Filter("Time__gte", dayStart).Filter("Time__lte", dayEnd).All(&dayOrders)
	dayPrice := 0
	for _, dayOrder := range dayOrders {
		dayPrice += dayOrder.TotalPrice
	}
	c.Data["dayPrice"] = dayPrice
	
	// 返回视图
	c.Layout = "administerLayout.html"
	c.TplName = "administer_stats.html"
}

// 处理用户搜索
func (c *AdministerController) HandleUserSearch() {
	// 1、获取数据
	name := c.GetString("name")
	o := orm.NewOrm()
	// 2、校验数据
	var users []models.User
	if name == " " {
		_, err := o.QueryTable("User").Filter("Power", 0).OrderBy("-Time").All(&users)
		if errors.Is(err, orm2.ErrNoRows) {
			log.Println("没有普通用户")
		} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
			log.Println("查询出错")
			c.Redirect("/administer/showUserModule", 302)
			return
		}
		c.Data["users"] = users
		return
	}

	_, err := o.QueryTable("User").Filter("Name__icontains", name).Filter("Power", 0).OrderBy("-Time").All(&users)
	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("对应用户不存在")
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询出错")
		c.Redirect("/administer/showUserModule", 302)
		return
	}

	c.Data["users"] = users
	c.Layout = "administerLayout.html"
	c.TplName = "administer_user.html"
}

// 处理用户删除
func (c *AdministerController) HandleUserDelete() {
	// 获取数据
	userName := c.GetString("userName")

	resp := make(map[string]interface{})
	defer c.ServeJSON()

	// 校验数据
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	err := o.Read(&user, "Name")
	if err != nil {
		log.Println("查找失败")
		resp["code"] = 1
		resp["msg"] = "删除失败"
		c.Data["json"] = resp
		return
	}

	_, err = o.Delete(&user)
	if err != nil {
		log.Println("删除失败")
		resp["code"] = 2
		resp["msg"] = "删除失败"
		c.Data["json"] = resp
		return
	}

	resp["code"] = 0
	resp["msg"] = "删除成功"
	c.Data["json"] = resp
}

// 处理评论搜索
func (c *AdministerController) HandleCommentSearch() {
	comment := c.GetString("comment")

	o := orm.NewOrm()
	var goods []models.OrderGoods

	if comment == " " {
		_, err := o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").RelatedSel("OrderInfo__User").OrderBy("CommentTime").All(&goods)
		if errors.Is(err, orm2.ErrNoRows) {
			log.Println("没有评论")
		} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
			log.Println("查询出错")
			c.Redirect("/administer/showCommentModule", 302)
			return
		}
		c.Data["goods"] = goods
		return
	}

	_, err := o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").RelatedSel("OrderInfo__User").Filter("Comment__icontains", comment).OrderBy("CommentTime").All(&goods)
	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("没有对应评论")
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询出错")
		c.Redirect("/administer/showCommentModule", 302)
		return
	}

	c.Data["goods"] = goods
	c.Layout = "administerLayout.html"
	c.TplName = "administer_comment.html"
}

// 处理评论删除
func (c *AdministerController) HandleCommentDelete() {
	goodsName := c.GetString("goodsName")
	userName := c.GetString("userName")

	resp := make(map[string]interface{})
	defer c.ServeJSON()

	o := orm.NewOrm()
	var orderGoods models.OrderGoods
	err := o.QueryTable("OrderGoods").RelatedSel("GoodsSKU", "OrderInfo").RelatedSel("OrderInfo__User").Filter("OrderInfo__User__Name", userName).Filter("GoodsSKU__Name", goodsName).One(&orderGoods)
	if err != nil {
		log.Println("评论查询错误")
		resp["code"] = 1
		resp["msg"] = "删除失败"
		c.Data["json"] = resp
		return
	}

	orderGoods.Comment = ""
	_, err = o.Update(&orderGoods)
	if err != nil {
		log.Println("评论删除失败")
		resp["code"] = 2
		resp["msg"] = "删除失败"
		c.Data["json"] = resp
		return
	}

	resp["code"] = 0
	resp["msg"] = "删除成功"
	c.Data["json"] = resp
}

// 处理住房搜索
func (c *AdministerController) HandleHouseSearch() {
	goodsName := c.GetString("goodsName")

	o := orm.NewOrm()
	var goodsSKU []models.GoodsSKU

	if goodsName == " " {
		_, err := o.QueryTable("GoodsSKU").All(&goodsSKU)
		if errors.Is(err, orm2.ErrNoRows) {
			log.Println("没有住房")
		} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
			log.Println("查询出错")
			c.Redirect("/administer/showHouseModule", 302)
			return
		}
	}

	_, err := o.QueryTable("GoodsSKU").RelatedSel("User", "GoodsType").Filter("Name__icontains", goodsName).OrderBy("-Time").All(&goodsSKU)
	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("没有对应住房")
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询出错")
		c.Redirect("/administer/showHouseModule", 302)
		return
	}

	c.Data["goodsSKU"] = goodsSKU
	c.Layout = "administerLayout.html"
	c.TplName = "administer_house.html"
}

// 处理住房下架
func (c *AdministerController) HandleHouseDelete() {
	Id, _ := c.GetInt("Id")
	resp := make(map[string]interface{})
	defer c.ServeJSON()

	o := orm.NewOrm()
	var goodsSKU models.GoodsSKU
	goodsSKU.Id = Id

	err := o.Read(&goodsSKU)
	if err != nil {
		log.Println("住房核对失败")
		resp["code"] = 1
		resp["msg"] = "下架失败"
		c.Data["json"] = resp
		return
	}

	_, err = o.Delete(&goodsSKU)
	if err != nil {
		log.Println("删除失败")
		resp["code"] = 2
		resp["msg"] = "下架失败"
		c.Data["json"] = resp
		return
	}

	resp["code"] = 0
	resp["msg"] = "下架成功"
	c.Data["json"] = resp
}

// 处理订单搜索
func (c *AdministerController) HandleOrderSearch() {
	orderName := c.GetString("orderName")

	o := orm.NewOrm()
	var orderGoods []models.OrderGoods
	if orderName == " " {
		_, err := o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").OrderBy("-OrderInfo__Time").All(&orderGoods)
		if errors.Is(err, orm2.ErrNoRows) {
			log.Println("没有订单")
		} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
			log.Println("查询出错")
			c.Redirect("/administer/showOrderModule", 302)
			return
		}
	}

	_, err := o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").Filter("GoodsSKU__Name", orderName).OrderBy("-OrderInfo__Time").All(&orderGoods)
	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("没有对应订单")
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询出错")
		c.Redirect("/administer/showOrderModule", 302)
		return
	}

	c.Data["orderGoods"] = orderGoods
	c.Layout = "administerLayout.html"
	c.TplName = "administer_order.html"
}

// 处理订单取消
func (c *AdministerController) HandleOrderDelete() {
	// 获取订单Id
	Id, _ := c.GetInt("Id")
	resp := make(map[string]interface{})
	defer c.ServeJSON()

	o := orm.NewOrm()
	var orderInfo models.OrderInfo
	orderInfo.Id = Id

	//查询是否存在
	err := o.QueryTable("OrderInfo").Filter("OrderId", Id).One(&orderInfo)
	if err != nil {
		log.Println("订单不存在")
		resp["code"] = 1
		resp["msg"] = "取消失败"
		c.Data["json"] = resp
		return
	}

	// 开启事物
	err = o.Begin()
	if err != nil {
		log.Println("开启事物失败")
		return
	}
	defer o.Commit()

	/* 把库存加回去 */
	// 查询所有关联项
	var orderGoods []models.OrderGoods
	_, err = o.QueryTable("OrderGoods").Filter("OrderInfo__OrderId", Id).All(&orderGoods)
	if err != nil {
		log.Println("库存加回查询失败")
		return
	}

	for i := range orderGoods {
		// 获取关联的 GoodsSKU
		orderGoods[i].GoodsSKU = &models.GoodsSKU{Id: orderGoods[i].GoodsSKU.Id}
		if err = o.Read(&orderGoods[i], "GoodsSKU"); err != nil {
			log.Println("加载商品信息失败")
			return
		}
	}

	// 恢复库存
	for _, orderGood := range orderGoods {
		// 加锁读取最新库存
		err = o.QueryTable("GoodsSKU").Filter("Id", orderGood.GoodsSKU.Id).ForUpdate().One(orderGood.GoodsSKU)
		if err != nil {
			log.Println("锁定商品库存失败:", err)
			return
		}

		// 更新库存
		orderGood.GoodsSKU.Stock += orderGood.Count
		if _, err = o.Update(orderGood.GoodsSKU, "Stock"); err != nil {
			log.Println("更新库存失败:", err)
			return
		}
	}

	// 删除关联的OrderGoods数据
	_, err = o.QueryTable("OrderGoods").Filter("OrderInfo__OrderId", Id).Delete()
	if err != nil {
		log.Println("删除订单商品失败")
		return
	}

	// 删除主订单数据
	_, err = o.Delete(&orderInfo)
	if err != nil {
		log.Println("删除主订单数据失败")
		return
	}

	resp["code"] = 0
	resp["msg"] = "取消成功"
	c.Data["json"] = resp
}

// 处理订单确认
func (c *AdministerController) HandleOrderConfirm() {
	Id, _ := c.GetInt("Id")
	resp := make(map[string]interface{})
	defer c.ServeJSON()

	o := orm.NewOrm()
	var orderInfo models.OrderInfo
	orderInfo.Id = Id

	err := o.Read(&orderInfo)
	if err != nil {
		log.Println("订单核对失败")
		resp["code"] = 1
		resp["msg"] = "取消失败"
		c.Data["json"] = resp
		return
	}

	orderInfo.ConfirmStatus = 1
	_, err = o.Update(&orderInfo)
	if err != nil {
		log.Println("更新确认状态失败")
		resp["code"] = 2
		resp["msg"] = "确认失败"
		c.Data["json"] = resp
	}

	resp["code"] = 0
	resp["msg"] = "确认成功"
	c.Data["json"] = resp
}

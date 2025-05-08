package controllers

import (
	"encoding/base64"
	"errors"
	"github.com/beego/beego/v2/adapter/orm"
	"github.com/beego/beego/v2/adapter/utils"
	orm2 "github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/gomodule/redigo/redis"
	"github.com/keonjeo/fdfs_client"
	"log"
	"math"
	"path"
	"regexp"
	"rentHouses/models"
	"strconv"
	"time"
)

type UserController struct {
	beego.Controller
}

// 展示用户注册页面
func (c *UserController) ShowRegister() {
	c.TplName = "register.html"
}

// 处理用户注册数据
func (c *UserController) HandleRegister() {
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
		c.TplName = "register.html"
		return
	}
	reg, _ := regexp.Compile("^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$")
	res := reg.FindString(email)
	if res == "" {
		c.Data["error"] = "邮箱格式不正确，请重新输入"
		c.TplName = "register.html"
		return
	}
	// 3、处理数据
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	user.PassWord = password
	user.Email = email

	_, err := o.Insert(&user)
	if err != nil {
		c.Data["error"] = "注册失败,请更换数据注册"
		c.TplName = "register.html"
		return
	}

	// 发送邮件
	emailConfig := `{"username":"1144466883@qq.com","password":"lcbdzhpjdctmgfjg","host":"smtp.qq.com","port":587}`
	emailConn := utils.NewEMail(emailConfig)
	emailConn.From = "1144466883@qq.com"
	emailConn.To = []string{email}
	emailConn.Subject = "省心租用户注册"
	// 发送请求，即 /active
	emailConn.Text = "【省心租】用户激活，请在浏览器输入：192.168.117.132:8080/active?id=" + strconv.Itoa(user.Id) + " 激活用户,谢谢您的使用"
	err = emailConn.Send()
	if err != nil {
		log.Println("发送失败", err)
		c.TplName = "register.html"
		return
	}

	// 4、返回视图
	c.Redirect("/login", 302)
}

// 激活业务
func (c *UserController) HandleActive() {
	id, err := c.GetInt("id")
	if err != nil {
		c.Data["error"] = "要激活的用户不存在"
		c.TplName = "register.html"
		return
	}

	o := orm.NewOrm()
	user := models.User{}
	user.Id = id
	err = o.Read(&user)
	if err != nil {
		c.Data["error"] = "要激活的用户不存在"
		c.TplName = "register.html"
		return
	}

	user.Active = true
	_, err = o.Update(&user)
	// 返回视图
	c.Redirect("/login", 302)
}

// 展示用户登录页面
func (c *UserController) ShowLogin() {
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
	c.TplName = "login.html"
}

// 处理用户登陆数据
func (c *UserController) HandleLogin() {
	// 1、获取数据
	userName := c.GetString("username")
	password := c.GetString("pwd")
	// 2、数据校验
	if userName == "" || password == "" {
		c.Data["errmsg"] = "登陆数据不能为空"
		c.TplName = "login.html"
		return
	}
	// 3、执行操作
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	err := o.Read(&user, "Name")
	if user.Power != 0 {
		c.Data["errmsg"] = "该用户不存在"
		c.TplName = "login.html"
		return
	}
	if err != nil {
		c.Data["errmsg"] = "用户名或密码错误"
		c.TplName = "login.html"
		return
	}
	if user.PassWord != password {
		c.Data["errmsg"] = "用户名或密码错误"
		c.TplName = "login.html"
		return
	}
	if user.Active == false {
		c.Data["errmsg"] = "未激活，请激活后登陆"
		c.TplName = "login.html"
		return
	}

	user.Time = time.Now().Format("2006-01-02 15:04:05")
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

	// 4、跳转到首页
	err = c.SetSession("userName", userName)
	if err != nil {
		return
	}
	c.Redirect("/", 302)
}

// 用户退出登陆
func (c *UserController) HandleLogout() {
	err := c.DelSession("userName")
	if err != nil {
		return
	}
	c.Redirect("/login", 302)
}

// 展示用户个人信息页面
func (c *UserController) ShowUserCenterInfo() {
	userName := GetUser(&c.Controller)
	c.Data["userName"] = userName

	// 若无历史记录，那么就展示无历史记录
	showLabel := 1

	// 查询地址表内容
	o := orm.NewOrm()
	// 高级表查询、表关联
	var addr models.Address
	err := o.QueryTable("Address").RelatedSel("User").Filter("User__Name", userName).Filter("IsDefault", true).One(&addr)
	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("用户地址未填写")
		c.Data["addr"] = ""
	} else if err != nil {
		log.Println("表查询失败")
		c.Redirect("/", 302)
		return
	} else {
		c.Data["addr"] = addr
	}

	//获取历史浏览记录
	conn, err := redis.Dial("tcp", "192.168.117.132:6379")
	defer conn.Close()
	if err != nil {
		log.Println("redis连接错误")
	}

	//获取用户ID
	var user models.User
	user.Name = userName
	err = o.Read(&user, "Name")
	if err != nil {
		log.Println("查询用户失败")
		return
	}
	// 查询5条历史记录
	reply, err := conn.Do("lrange", "history_"+strconv.Itoa(user.Id), 0, 4)
	if err != nil {
		log.Println("查询历史记录失败", err)
	}

	goodsIDs, _ := redis.Ints(reply, err)
	var goodsSKUs []models.GoodsSKU

	for _, value := range goodsIDs {
		var goosSKU models.GoodsSKU
		goosSKU.Id = value
		err = o.Read(&goosSKU)
		if errors.Is(err, orm2.ErrNoRows) {
			log.Println("没有对应历史记录")
			showLabel = 0
		} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
			log.Println("历史记录查询失败")
			c.Redirect("/", 302)
			return
		} else {
			goodsSKUs = append(goodsSKUs, goosSKU)
		}
	}

	c.Data["showLabel"] = showLabel
	c.Data["goodsSKUs"] = goodsSKUs

	// 查询地址表内容，高级查询表关联
	c.Layout = "user_center_layout.html"
	c.TplName = "user_center_info.html"
}

// 展示用户订单信息页面
func (c *UserController) ShowUserCenterOrder() {
	userName := GetUser(&c.Controller)
	o := orm.NewOrm()
	var user models.User

	user.Name = userName
	err := o.Read(&user, "Name")
	if err != nil {
		log.Println("读取数据错误")
		return
	}

	/* 分页实现 */
	// 1、获取总的数据数量
	var Infos []models.OrderInfo
	count, _ := o.QueryTable("OrderInfo").RelatedSel("User").Filter("User__Id", user.Id).All(&Infos)

	pageSize := 4
	// 向上取整分隔有多少个页面
	pageCount := math.Ceil(float64(count) / float64(pageSize))
	// 2、获取当前pageIndex
	pageIndex, err := c.GetInt("pageIndex")
	// 如果没获取到pageIndex则说明是刚打开的情况，设置默认值为1
	if err != nil {
		pageIndex = 1
	}
	// 3、分页函数
	pages := splitPage(int(pageCount), pageIndex)
	// 4、返回分页的结果
	c.Data["pages"] = pages
	c.Data["pageIndex"] = pageIndex

	// 每一页的开始数据索引
	start := (pageIndex - 1) * pageSize

	// 获取上一页的页码
	prePage := pageIndex - 1
	if prePage <= 1 {
		prePage = 1
	}
	c.Data["prePage"] = prePage

	// 获取下一页页码
	nextPage := pageIndex + 1
	if nextPage >= int(pageCount) {
		nextPage = int(pageCount)
	}
	c.Data["nextPage"] = nextPage

	// 获取订单表数据(按照一定顺序)
	var orderInfos []models.OrderInfo
	_, _ = o.QueryTable("OrderInfo").RelatedSel("User").Filter("User__Id", user.Id).Limit(pageSize, start).All(&orderInfos)

	goodsBuffer := make([]map[string]interface{}, len(orderInfos))

	for index, orderInfo := range orderInfos {
		var orderGoods []models.OrderGoods
		o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").Filter("OrderInfo__Id", orderInfo.Id).All(&orderGoods)

		temp := make(map[string]interface{})
		temp["orderInfo"] = orderInfo
		temp["orderGoods"] = orderGoods

		goodsBuffer[index] = temp
	}

	c.Data["goodsBuffer"] = goodsBuffer

	// 若为空，则不用显示最底下的元，已支付，未支付的框，优化页面
	showLabel := 1
	if len(orderInfos) == 0 {
		showLabel = 0
	}
	c.Data["showLabel"] = showLabel

	// 返回视图
	c.Layout = "user_center_layout.html"
	c.TplName = "user_center_order.html"
}

// 展示我的发布页面
func (c *UserController) ShowUserCenterMyPublish() {
	userName := GetUser(&c.Controller)
	c.Data["userName"] = userName

	// 查询我发布的订单
	var goodsSKUs []models.GoodsSKU
	o := orm.NewOrm()
	count, err := o.QueryTable("GoodsSKU").RelatedSel("User", "GoodsType").Filter("User__Name", userName).OrderBy("-Time").All(&goodsSKUs)
	if err != nil {
		log.Println("查询失败", err)
		return
	}

	// 租出和未支付的房屋
	var allRentGoods []models.OrderGoods
	// 未租出的房屋
	var allRestGoods []models.GoodsSKU

	for _, goodsSKU := range goodsSKUs {
		//查询与该用户相同名字的订单商品，查到就表明租出去，未查到就未租出
		var orderGoods models.OrderGoods
		err = o.QueryTable("OrderGoods").RelatedSel("GoodsSKU", "OrderInfo").Filter("GoodsSKU__Id", goodsSKU.Id).RelatedSel("GoodsSKU__User", "OrderInfo__Address").Filter("GoodsSKU__User__Name", userName).One(&orderGoods)
		if !errors.Is(err, orm2.ErrNoRows) && err != nil {
			log.Println("查询卖出订单失败:", err)
			return
		} else if err == nil {
			// 该商品租出去了，插入到租出去的rentGoods表
			allRentGoods = append(allRentGoods, orderGoods)
		} else {
			// 未查找到，该商品未卖出去,插入未租出去的restGoods表
			allRestGoods = append(allRestGoods, goodsSKU)
		}
	}

	c.Data["allRentGoods"] = allRentGoods
	c.Data["allRestGoods"] = allRestGoods

	/* 实现分页 */

	// 1、获取租出去的总个数count
	pageSize := 5
	// 向上取整分隔有多少个页面
	pageCount := math.Ceil(float64(count) / float64(pageSize))
	// 2、获取当前pageIndex
	pageIndex, err := c.GetInt("pageIndex")
	// 如果没获取到pageIndex则说明是刚打开的情况，设置默认值为1
	if err != nil {
		pageIndex = 1
	}
	// 3、分页函数
	pages := splitPage(int(pageCount), pageIndex)
	// 4、返回分页的结果
	c.Data["pages"] = pages
	c.Data["pageIndex"] = pageIndex

	// 获取上一页的页码
	prePage := pageIndex - 1
	if prePage <= 1 {
		prePage = 1
	}
	c.Data["prePage"] = prePage

	// 获取下一页页码
	nextPage := pageIndex + 1
	if nextPage >= int(pageCount) {
		nextPage = int(pageCount)
	}
	c.Data["nextPage"] = nextPage

	// 截取租出和未支付的房屋
	var rentGoods []models.OrderGoods
	// 截取未租出的房屋
	var restGoods []models.GoodsSKU

	if len(allRentGoods) < pageSize {
		// 如果租出去和未支付的总个数小于pageSize，则第一页返回租出去的和pageSize减去租出去的个数的未租出商品，后续都是未租出去的
		if pageIndex == 1 {
			// 第一页
			for i := 0; i < len(rentGoods); i++ {
				// 处理租出去和未支付总个数为零的情况，防止赋值失败导致panic
				if len(allRentGoods) != 0 {
					rentGoods = append(rentGoods, allRentGoods[i])
				}
			}
			c.Data["rentGoods"] = allRentGoods

			for j := 0; j < pageSize-len(rentGoods)-1; j++ {
				// 处理未租出个数为零的情况，防止赋值失败导致panic
				if len(allRestGoods) != 0 {
					restGoods = append(restGoods, allRestGoods[j])
				}
			}
			c.Data["restGoods"] = restGoods
		} else if (pageIndex-1)*pageSize+pageSize-len(rentGoods)-1 > len(allRestGoods) {
			// 处理最后一页，避免索引超出
			c.Data["rentGoods"] = rentGoods
			for i := (pageIndex-2)*pageSize + pageSize - len(rentGoods) - 1; i < len(allRestGoods); i++ {
				// 过滤掉未租出个数为零导致panic
				if len(allRestGoods) != 0 {
					restGoods = append(restGoods, allRestGoods[i])
				}
			}
			c.Data["restGoods"] = restGoods
		} else {
			// 其他页
			c.Data["rentGoods"] = rentGoods
			for i := (pageIndex-2)*pageSize + pageSize - len(rentGoods) - 1; i < (pageIndex-1)*pageSize+pageSize-len(rentGoods)-1; i++ {
				// 过滤掉未租出个数为零导致panic
				if len(allRestGoods) != 0 {
					restGoods = append(restGoods, allRestGoods[i])
				}
			}
			c.Data["restGoods"] = restGoods
		}
	} else if len(allRentGoods) == pageSize {
		// 如果租出去和未支付的总个数等于pageSize,则第一页返回租出去和未支付的，后续的页数都是未租出去的
		if pageIndex == 1 {
			// 第一页
			c.Data["rentGoods"] = allRentGoods
		} else if (pageIndex-1)*pageSize > len(allRestGoods) {
			// 处理最后一页，避免超出索引最大值
			c.Data["rentGoods"] = rentGoods
			for i := (pageIndex - 2) * pageSize; i < len(allRestGoods); i++ {
				if len(allRestGoods) != 0 {
					// 过滤掉未租出个数为零导致panic
					if len(allRestGoods) != 0 {
						restGoods = append(restGoods, allRestGoods[i])
					}
				}
			}
			c.Data["restGoods"] = restGoods
		} else {
			// 其他页
			c.Data["rentGoods"] = rentGoods
			for i := (pageIndex - 2) * pageSize; i < (pageIndex-1)*pageSize; i++ {
				// 过滤掉未租个数为零的情况导致赋值失败的panic
				if len(allRestGoods) != 0 {
					restGoods = append(restGoods, allRestGoods[i])
				}
			}
			c.Data["restGoods"] = restGoods
		}
	} else {
		// 如果租出去和未支付的总个数大于pageSize,则到分界处，租出去的个数除以pageSize向上取整的页面=分界页面，分界页面显示pageSize以及pageSize减去余数的未租出房屋个数，后续全为未租出去的
		rentCount := len(allRentGoods) % pageSize // 计算出余数

		if pageIndex < len(allRentGoods)/pageSize+1 {
			// 分界处之前
			for i := (pageIndex - 1) * pageSize; i < pageIndex*pageSize; i++ {
				rentGoods = append(rentGoods, allRentGoods[i])
			}
			c.Data["rentGoods"] = rentGoods
			c.Data["restGoods"] = restGoods
		} else if pageIndex == len(allRentGoods)/pageSize+1 {
			// 在分界处
			for i := len(allRentGoods) - rentCount; i < len(allRentGoods); i++ {
				rentGoods = append(rentGoods, allRentGoods[i])
			}
			c.Data["rentGoods"] = rentGoods
			for j := 0; j < pageSize-rentCount; j++ {
				// 过滤掉未租出个数为零导致panic
				if len(allRestGoods) != 0 {
					restGoods = append(restGoods, allRestGoods[j])
				}
			}
			c.Data["restGoods"] = restGoods
		} else if pageSize-rentCount+(pageIndex-len(allRentGoods)/pageSize-1)*pageSize > len(allRestGoods) {
			// 最后一页过滤溢出处理
			c.Data["rentGoods"] = rentGoods
			for i := pageSize - rentCount + (pageIndex-len(allRentGoods)/pageSize-2)*pageSize; i < len(allRestGoods); i++ {
				// 过滤掉未租出个数为零导致panic
				if len(allRestGoods) != 0 {
					restGoods = append(restGoods, allRestGoods[i])
				}
			}
			c.Data["restGoods"] = restGoods
			log.Println("最后一页")
		} else {
			// 分界处之后
			c.Data["rentGoods"] = rentGoods
			for i := pageSize - rentCount + (pageIndex-len(allRentGoods)/pageSize-2)*pageSize; i < pageSize-rentCount+(pageIndex-len(allRentGoods)/pageSize-1)*pageSize; i++ {
				// 过滤掉未租出个数为零导致panic
				if len(allRestGoods) != 0 {
					restGoods = append(restGoods, allRestGoods[i])
				}
			}
			c.Data["restGoods"] = restGoods
			log.Println("分界处之后")
		}
	}

	// 判断展示分页按钮
	showLabel := 1
	if count < int64(pageSize) {
		showLabel = 0
	}
	c.Data["showLabel"] = showLabel

	// 返回视图
	c.Layout = "user_center_layout.html"
	c.TplName = "user_center_myPublish.html"
}

// 展示住房信息更新页面
func (c *UserController) ShowGoodsUpdate() {
	userName := GetUser(&c.Controller)
	c.Data["userName"] = userName

	skuId, err := c.GetInt("skuId")
	if err != nil {
		log.Println("获取住房Id失败")
	}

	var goodsSKU models.GoodsSKU
	o := orm.NewOrm()
	err = o.QueryTable("GoodsSKU").RelatedSel("Goods", "GoodsType", "User").Filter("Id", skuId).One(&goodsSKU)
	if err != nil {
		log.Println("获取住房失败")
		return
	}
	c.Data["goodsSKU"] = goodsSKU

	c.Layout = "user_center_layout.html"
	c.TplName = "user_center_updatePublish.html"
}

// 处理住房信息更新
func (c *UserController) HandleGoodsUpdate() {
	resp := make(map[string]interface{})
	defer c.ServeJSON()
	// 1、获取数据
	skuId, err := c.GetInt("skuId")
	if err != nil {
		log.Println("获取住房Id失败")
		resp["code"] = 1
		resp["msg"] = "住房信息获取失败"
		c.Data["json"] = resp
	}

	goodsTypeId, err := c.GetInt("goodsTypeId")
	if err != nil {
		log.Println("获取类型Id失败")
		resp["code"] = 2
		resp["msg"] = "住房类型获取失败"
		c.Data["json"] = resp
	}
	goodsName := c.GetString("goodsName")
	goodsPrice, _ := c.GetInt("goodsPrice")
	goodsAddr := c.GetString("goodsAddr")
	goodsPhone := c.GetString("goodsPhone")
	goodsDesc := c.GetString("goodsDesc")
	goodsDetail := c.GetString("goodsDetail")

	o := orm.NewOrm()

	// 获取上传文件
	file, head, err := c.GetFile("goodsImage")
	defer file.Close()
	if err != nil {
		resp["code"] = 3
		resp["msg"] = "获取文件失败,请稍候再试"
		c.Data["json"] = resp
		log.Println("获取文件失败")
		return
	}

	// 2、校验数据
	// 判断文件格式
	ext := path.Ext(head.Filename)
	if ext != ".jpg" && ext != ".png" && ext != ".jpeg" && ext != ".gif" && ext != ".bmp" && ext != ".webp" {
		log.Println("图片格式不正确")
		return
	}

	// 判断文件大小
	if head.Size > 5000000 {
		log.Println("文件太大，不允许上传")
		resp["code"] = 4
		resp["msg"] = "文件过大"
		c.Data["json"] = resp
		return
	}

	// 3、处理数据
	// (1)获取一个[]byte
	fileBuffer := make([]byte, head.Size)
	// (2)把文件数据读入到fileBuffer中
	_, err = file.Read(fileBuffer)
	if err != nil {
		log.Println("读入文件数据失败")
		resp["code"] = 5
		resp["msg"] = "文件读入失败"
		c.Data["json"] = resp
		return
	}

	// (3)获取client对象
	client, _ := fdfs_client.NewFdfsClient("/etc/fdfs/client.conf")

	// (4)上传,该切片截取是由于ext为.jpg等形式，从1开始截取就去掉.了
	fdfsResponse, err := client.UploadByBuffer(fileBuffer, ext[1:])
	if err != nil {
		resp["code"] = 6
		resp["msg"] = "上传文件失败,请稍候再试"
		c.Data["json"] = resp
		return
	}
	// 截取最后一段路径即为上传成功的图片名
	imageName := fdfsResponse.RemoteFileId

	var goodsSKU models.GoodsSKU
	err = o.QueryTable("GoodsSKU").RelatedSel("Goods", "GoodsType").Filter("Id", skuId).One(&goodsSKU)
	if err != nil {
		log.Println("查询SKU表失败")
		resp["code"] = 7
		resp["msg"] = "信息查询失败"
		c.Data["json"] = resp
		return
	}

	// (5)删除原来的图片
	err = client.DeleteFile(goodsSKU.Image)
	if err != nil {
		log.Println("删除失败")
		return
	}

	goodsSKU.Id = skuId
	goodsSKU.Name = goodsName
	goodsSKU.Price = goodsPrice
	goodsSKU.Image = imageName
	goodsSKU.Addr = goodsAddr
	goodsSKU.Phone = goodsPhone
	goodsSKU.Desc = goodsDesc
	goodsSKU.GoodsType.Id = goodsTypeId
	goodsSKU.Goods.Detail = goodsDetail

	_, err = o.Update(&goodsSKU)
	if err != nil {
		log.Println("信息更新失败")
		resp["code"] = 8
		resp["msg"] = "住房信息获取失败"
		c.Data["json"] = resp
		return
	}

	resp["code"] = 9
	resp["msg"] = "更新成功"
	log.Println("更新成功")
	c.Data["json"] = resp
}

// 处理住房下架
func (c *UserController) HandleGoodsDelete() {
	// 1、获取数据
	skuId, _ := c.GetInt("skuId")
	var goodsSKU models.GoodsSKU
	goodsSKU.Id = skuId
	o := orm.NewOrm()

	// 2、校验数据
	err := o.Read(&goodsSKU)
	if err != nil {
		log.Println("查询对应商品错误")
		c.Redirect("/user/userCenterMyPublish", 302)
		return
	}

	// 3、处理数据
	_, err = o.Delete(&goodsSKU)
	if err != nil {
		log.Println("下架商品失败")
		c.Redirect("/user/userCenterMyPublish", 302)
		return
	}

	// 获取client对象
	client, _ := fdfs_client.NewFdfsClient("/etc/fdfs/client.conf")

	// 删除FastDFS里面存储的图片
	err = client.DeleteFile(goodsSKU.Image)
	if err != nil {
		log.Println("删除失败")
		return
	}

	// 4、返回视图
	c.Redirect("/user/userCenterMyPublish", 302)
}

// 展示发布房源页面
func (c *UserController) ShowUserCenterPublish() {
	userName := GetUser(&c.Controller)
	c.Data["userName"] = userName

	// 返回视图
	c.Layout = "user_center_layout.html"
	c.TplName = "user_center_publish.html"
}

// 处理发布房源信息
func (c *UserController) HandleUserCenterPublish() {
	userName := GetUser(&c.Controller)
	o := orm.NewOrm()
	var user models.User

	user.Name = userName
	err := o.Read(&user, "Name")
	if err != nil {
		log.Println("读取数据错误")
		return
	}

	// 1、获取数据
	goodsTypeId, _ := c.GetInt("goodsTypeId")
	goodsName := c.GetString("goodsName")
	goodsPrice, _ := c.GetInt("goodsPrice")
	goodsAddr := c.GetString("goodsAddr")
	goodsPhone := c.GetString("goodsPhone")
	goodsDesc := c.GetString("goodsDesc")
	goodsDetail := c.GetString("goodsDetail")

	resp := make(map[string]interface{})
	defer c.ServeJSON()

	count, _ := o.QueryTable("GoodsSKU").RelatedSel("User").Filter("User__Name", userName).Filter("Name", goodsName).Count()
	if count != 0 {
		resp["code"] = 0
		resp["msg"] = "该住房已上传，该操作不被允许"
		c.Data["json"] = resp
		return
	}

	// 获取上传文件
	file, head, err := c.GetFile("goodsImage")
	defer file.Close()
	if err != nil {
		resp["code"] = 1
		resp["msg"] = "获取文件失败,请稍候再试"
		c.Data["json"] = resp
		return
	}

	// 2、校验数据
	// 判断文件格式
	ext := path.Ext(head.Filename)
	if ext != ".jpg" && ext != ".png" && ext != ".jpeg" && ext != ".gif" && ext != ".bmp" && ext != ".webp" {
		log.Println("图片格式不正确")
		return
	}

	// 判断文件大小
	if head.Size > 5000000 {
		log.Println("文件太大，不允许上传")
		resp["code"] = 2
		resp["msg"] = "文件过大"
		c.Data["json"] = resp
		return
	}

	// 3、处理数据
	// (1)获取一个[]byte
	fileBuffer := make([]byte, head.Size)
	// (2)把文件数据读入到fileBuffer中
	_, err = file.Read(fileBuffer)
	if err != nil {
		log.Println("读入文件数据失败")
		resp["code"] = 3
		resp["msg"] = "文件读入失败"
		c.Data["json"] = resp
		return
	}

	// (3)获取client对象
	client, _ := fdfs_client.NewFdfsClient("/etc/fdfs/client.conf")
	// (4)上传,该切片截取是由于ext为.jpg等形式，从1开始截取就去掉.了
	fdfsResponse, err := client.UploadByBuffer(fileBuffer, ext[1:])
	if err != nil {
		resp["code"] = 2
		resp["msg"] = "上传文件失败,请稍候再试"
		c.Data["json"] = resp
		return
	}
	// 截取最后一段路径即为上传成功的图片名
	imageName := fdfsResponse.RemoteFileId

	// 开启事物
	err = o.Begin()
	if err != nil {
		log.Println("开启失败")
		return
	}
	var goodsType models.GoodsType
	goodsType.Id = goodsTypeId
	err = o.Read(&goodsType, "Id")
	if err != nil {
		log.Println("读取类型表错误")
		err = o.Rollback()
		if err != nil {
			log.Println("类型表回滚失败")
		}
		resp["code"] = 3
		resp["msg"] = "未选择类型"
		c.Data["json"] = resp
		return
	}

	var goods models.Goods
	goods.Name = goodsName
	err = o.Read(&goods, "Name")

	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("goods表该行不存在,创建新行")
		goods.Detail = goodsDetail
		_, err = o.Insert(&goods)
		if err != nil {
			err = o.Rollback()
			if err != nil {
				log.Println("插入回滚失败")
			}
			c.Redirect("/user/userCenterUpload", 302)
			return
		}
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询错误")
		err = o.Rollback()
		if err != nil {
			log.Println("查询回滚失败")
		}
		return
	}

	var goodsSKU models.GoodsSKU
	goodsSKU.User = &user
	goodsSKU.Goods = &goods
	goodsSKU.GoodsType = &goodsType
	goodsSKU.Name = goodsName
	goodsSKU.Desc = goodsDesc
	goodsSKU.Price = goodsPrice
	goodsSKU.Unite = "RMB"
	goodsSKU.Image = imageName
	goodsSKU.Addr = goodsAddr
	goodsSKU.Phone = goodsPhone
	goodsSKU.Stock = 1
	goodsSKU.Status = 1
	goodsSKU.Time = time.Now().Format("2006-01-02 15:04:05")

	_, err = o.Insert(&goodsSKU)
	if err != nil {
		log.Println("插入失败")
		err = o.Rollback()
		if err != nil {
			log.Println("插入回滚失败")
		}
		c.Redirect("/user/userCenterUpload", 302)
		return
	}

	o.Commit()
	// 4、返回视图
	resp["code"] = 5
	resp["msg"] = "发布成功"
	c.Data["json"] = resp
}

// 展示用户地址信息页面
func (c *UserController) ShowUserCenterSite() {
	userName := GetUser(&c.Controller)
	c.Data["userName"] = userName
	//获取地址信息
	o := orm.NewOrm()
	var addr models.Address
	err := o.QueryTable("Address").RelatedSel("User").Filter("User__Name", userName).Filter("IsDefault", true).One(&addr)
	c.Data["addr"] = addr

	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("无地址记录")
		c.Data["addr"] = ""
	} else if err != nil {
		log.Println("查询失败")
	}

	//传递给视图
	c.Layout = "user_center_layout.html"
	c.TplName = "user_center_site.html"
}

// 更新用户中心地址数据
func (c *UserController) HandleUserCenterSite() {
	userName := c.GetSession("userName")
	var user models.User
	user.Name = userName.(string)

	o := orm.NewOrm()
	err := o.Read(&user, "Name")
	if err != nil {
		log.Println("没有该用户", err)
		return
	}

	// 1、获取数据
	receiver := c.GetString("receiver")
	addr := c.GetString("addr")
	zipCode := c.GetString("zipCode")
	phone := c.GetString("phone")

	// 2、校验数据
	if receiver == "" || addr == "" || zipCode == "" || phone == "" {
		log.Println("数据不完整")
		c.Redirect("/user/userCenterSite", 302)
		return
	}

	// 3、处理数据
	var address models.Address
	err = o.QueryTable("Address").RelatedSel("User").Filter("User__Name", userName).One(&address)

	address.User = &user
	address.Receiver = receiver
	address.IsDefault = true
	address.Addr = addr
	address.Phone = phone
	address.Zipcode = zipCode

	if errors.Is(err, orm2.ErrNoRows) {
		log.Println("未查询到地址,创建地址")
		_, err = o.Insert(&address)
		if err != nil {
			log.Println("创建地址失败")
			c.Redirect("/user/userCenterSite", 302)
			return
		}
	} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("查询地址失败")
		c.Redirect("/user/userCenterSite", 302)
		return
	} else {
		// 查询到了更新地址
		_, err = o.Update(&address)
		if err != nil {
			log.Println("更新失败", err)
			return
		}
	}

	// 4、视图返回
	c.Redirect("/user/userCenterSite", 302)
}

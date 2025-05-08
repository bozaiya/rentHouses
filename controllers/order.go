package controllers

import (
	"fmt"
	"github.com/beego/beego/v2/adapter/orm"
	orm2 "github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/gomodule/redigo/redis"
	"github.com/smartwalle/alipay/v3"
	"log"
	"math"
	"rentHouses/models"
	"strconv"
	"strings"
	"time"
)

type OrderController struct {
	beego.Controller
}

// 订单页面展示
func (c *OrderController) ShowOrder() {
	// 获取数据
	skuids := c.GetStrings("skuid")

	log.Println(skuids)

	// 校验数据
	if len(skuids) == 0 {
		log.Println("购物车请求数据错误")
		c.Redirect("/user/cart", 302)
		return
	}

	// 处理数据
	o := orm.NewOrm()
	conn, _ := redis.Dial("tcp", "192.168.117.132:6379")
	defer conn.Close()

	// 获取用户数据
	var user models.User
	userName := c.GetSession("userName")
	user.Name = userName.(string)
	err := o.Read(&user, "Name")
	if err != nil {
		log.Println("读取用户失败")
		return
	}

	goodsBuffer := make([]map[string]interface{}, len(skuids))

	totalCount := 0
	totalPrice := 0

	for index, skuid := range skuids {

		temp := make(map[string]interface{})
		// 将获取到的string转换成int的id
		id, _ := strconv.Atoi(skuid)
		// 查询商品数据
		var goodsSKU models.GoodsSKU
		goodsSKU.Id = id
		//从redis读取到了商品数据SKU赋给GoodsSKU
		err = o.Read(&goodsSKU)
		if err != nil {
			log.Println("读取商品数据失败")
			return
		}
		temp["goodsSKU"] = goodsSKU

		//获取商品数量(在redis中)
		count, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), id))
		temp["count"] = count
		// 计算小计
		amount := goodsSKU.Price * count
		temp["amount"] = amount
		// 计算总金额和总件数
		totalCount += count
		totalPrice += amount
		//临时容器数据给这个buffer
		goodsBuffer[index] = temp
	}
	// 传商品数据(包含数量和SKU)给前端
	c.Data["goodsBuffer"] = goodsBuffer

	// 查询地址信息
	var addrs []models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__Id", user.Id).All(&addrs)
	addrCount := len(addrs)
	// 传地址信息给前端
	c.Data["addrCount"] = addrCount
	c.Data["addrs"] = addrs

	// 传递总金额和总件数
	c.Data["totalCount"] = totalCount
	c.Data["totalPrice"] = totalPrice
	transferPrice := int(math.Round(float64(totalPrice) * 0.1))
	c.Data["transferPrice"] = transferPrice
	c.Data["reallyPrice"] = totalPrice + transferPrice

	// 传递所有商品的id
	c.Data["skuids"] = skuids

	/* 分页处理 */
	/* 一页可以展示5个数据
	pageSize := 5
	count := len(goodsBuffer)
	pageCount := float64(count) / float64(pageSize)
	pageIndex, _ := c.GetInt("pageIndex")
	if pageCount <= 3 {
		pages := make([]int, int64(math.Ceil(pageCount)))
		for i := 0; i < len(pages); i++ {
			pages[i] = i + 1
			c.Data["pages"] = pages
		}
	} else if pageCount > 3 {
		pages := []int{1, 2, int(pageCount - 1), int(pageCount)}
		c.Data["pages"] = pages
	}
	// 每页展示
	start := (pageIndex - 1) * pageSize
	goods := make([]map[string]interface{}, pageSize)
	for i := 0; i < pageSize; i++ {
		goods[i] = goodsBuffer[start+i]
	}
	c.Data["goods"] = goods

	//获取上页页码
	if pageIndex <= 1 {
		pageIndex = 1
		c.Data["prePage"] = pageIndex
	} else {
		pageIndex--
		c.Data["prePage"] = pageIndex
	}

	//获取下页页码
	if pageIndex >= int(math.Ceil(pageCount)) {
		pageIndex = int(math.Ceil(pageCount))
		c.Data["nextPage"] = pageIndex
	} else {
		pageIndex++
		c.Data["nextPage"] = pageIndex
	}
	*/

	// 返回视图
	GetUser(&c.Controller)
	c.TplName = "place_order.html"
}

// 提交订单
func (c *OrderController) HandleAddOrder() {
	// 获取数据
	addrId, _ := c.GetInt("addrId")
	payId, _ := c.GetInt("payId")
	skuid := c.GetString("skuids")
	ids := skuid[1 : len(skuid)-1]
	skuids := strings.Split(ids, " ")

	log.Println("地址：", addrId, "支付码:", payId, "skuid:", skuid, "ids:", ids, "skuids", skuids)

	totalCount, _ := c.GetInt("totalCount")
	transferPrice, _ := c.GetInt("transferPrice")
	reallyPrice, _ := c.GetInt("reallyPrice")

	// 校验数据
	resp := make(map[string]interface{})
	defer c.ServeJSON()

	if payId == 0 || len(skuids) == 0 {
		resp["code"] = 1
		resp["msg"] = "数据库连接错误"
		c.Data["json"] = resp
		return
	}

	if addrId == 0 {
		log.Println("未填写地址")
		resp["code"] = 2
		resp["msg"] = "未填写地址，请填写地址后操作"
		c.Data["json"] = resp
		return
	}

	// 处理数据
	// 向订单表中插入数据
	o := orm.NewOrm()
	// 开启事物
	err := o.Begin()
	if err != nil {
		log.Println("开启失败")
		return
	}

	var user models.User
	userName := c.GetSession("userName")
	if userName == nil {
		log.Println("用户名为空")
		return
	}
	user.Name = userName.(string)
	err = o.Read(&user, "Name")
	if err != nil {
		log.Println("读取用户失败")
		return
	}

	var order models.OrderInfo
	order.OrderId = time.Now().Format("20060102150405") + strconv.Itoa(user.Id)
	order.User = &user
	order.OrderStatus = 0
	order.PayMethod = payId
	order.TotalCount = totalCount
	order.TotalPrice = reallyPrice
	order.TransitPrice = transferPrice
	// 查询地址
	var addr models.Address
	addr.Id = addrId
	err = o.Read(&addr)
	if err != nil {
		log.Println("读取地址失败")
		return
	}
	// 执行插入操作
	order.Address = &addr
	_, err = o.Insert(&order)
	if err != nil {
		log.Println("插入失败", err)
		return
	}

	// 向订单商品表中插入数据
	conn, _ := redis.Dial("tcp", "192.168.117.132:6379")
	for _, skuid := range skuids {
		id, _ := strconv.Atoi(skuid)

		var goodsSKU models.GoodsSKU
		goodsSKU.Id = id
		for i := 0; i < 3; i++ {
			err = o.Read(&goodsSKU)
			if err != nil {
				log.Println("读取商品失败")
				return
			}

			var orderGoods models.OrderGoods
			orderGoods.GoodsSKU = &goodsSKU
			orderGoods.OrderInfo = &order

			// 判断库存,从redis购物车里面取商品数量
			count, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), id))

			if count > goodsSKU.Stock {
				resp["code"] = 3
				resp["msg"] = "商品库存不足"
				c.Data["json"] = resp
				// 回滚操作
				err = o.Rollback()
				if err != nil {
					log.Println("回滚失败")
					return
				}
				return
			}

			preCount := goodsSKU.Stock
			time.Sleep(time.Second * 1)

			orderGoods.Count = count
			orderGoods.Price = count * goodsSKU.Price

			_, err = o.Insert(&orderGoods)
			if err != nil {
				return
			}

			goodsSKU.Stock -= count
			goodsSKU.Sales += count
			// 提交事物
			updateCount, _ := o.QueryTable("GoodsSKU").Filter("Id", goodsSKU.Id).Filter("Stock", preCount).Update(orm2.Params{"Stock": goodsSKU.Stock, "Sales": goodsSKU.Sales})
			if updateCount == 0 {
				if i >= 0 {
					i -= 1
					continue
				}
				resp["code"] = 4
				resp["msg"] = "商品库存改变"
				c.Data["json"] = resp
				err = o.Rollback()
				if err != nil {
					log.Println("库存改变、回滚失败")
					return
				}
				return
			} else {
				//提交订单成功了，那么则需要把购物车已勾选的数据给删除
				_, err = conn.Do("hdel", "cart_"+strconv.Itoa(user.Id), goodsSKU.Id)
				if err != nil {
					log.Println("删除失败", err)
					return
				}
				break
			}
		}
	}
	err = o.Commit()
	if err != nil {
		log.Println("提交事物失败")
		return
	}

	// 返回视图
	resp["code"] = 5
	resp["msg"] = "提交成功"
	c.Data["json"] = resp
}

// 处理支付
func (c *OrderController) HandlePay() {
	var privateKey = "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCyfNjuIJSm/b2Q" +
		"TyMNE6AsdJiAPOk9twjNOqFZ3GAp9YejBNwNbxXJfEXzHnvEVdnpYP5aMty+07ah" +
		"YZtLZ2REG1Pi0GmmfjdjYk5UbtMs2kXXfURzhMahqp0dKRxUCzQ5me+LvubNlql7" +
		"C1sg+rcNxEhwyx1+ZRPEa7otnc392WYtfMT78wFmYRzf1IC6tC+YLH1HzufCM2uZ" +
		"Tqw0HoXOJax6qZcoFcIjDOwiUAAmxaaHkRP0m3mQJBXNR1mnKuXaQTWyeCEMpLe4" +
		"P8093bHE050r0uZQZOIwTqflPYH/2+0VeGGT6sfW/cNP2x6+rS0y6ct54stbLb11" +
		"sCYHFiX7AgMBAAECggEAFf0h1Sr9X+9C+HF6E3qB+/K8oOZ1eO1rhQET71f3dDkz" +
		"5buhB89SI07kkorZsOU/YdbZrwS0w5ZIhGwxGg0u7IxjUr4zQy1/JwbC402oms4k" +
		"gY4vWH3t4e0ndaK4GIster/XTjjirUbs3TmXwfcxMF3TQNYSbZ0H3jsZUNUFPiBi" +
		"VSm6J53iOAmI88l8mEMGzwyv+7EtMeuU65s1QzENQjXLjQW+uJzDkZYLoQkymAJM" +
		"19CXCHsHBytRvi0FKhjJN1wrhBBV0tvFAujAmlb5ZJOdlGdnHnVh/EZSNUtO7rAj" +
		"SS3sSRsn4CS/hPmUAiXv2dfVj5uu/PqAFLWwL8vNlQKBgQD0AU9aLNrbCpbk8tMv" +
		"IAMDOQK0ylkp/KeV74WIBkPxAi2XHPzut4qJxcrji1Toj5z67aIPQk3l6Lt8aC19" +
		"LmwoStrpGDgK2+InrPoDIn4Q2ansN3+PCcdJf/31+7EGc+TYREE3Ato4lESgzLfJ" +
		"rjzD66CPQBJINBsEM406MBoFHQKBgQC7Qwf/a8p6L009KcTbR8Snv3rQymic3JQB" +
		"G1+PWRD4tXyiFBPaOYbYFvF4CIZubNPeTZLpy7xXRYcKgthCKf8BtImu7MoR9OjO" +
		"Tmz9btXhDkM8tDlwyJWEo6+bMhLUgdUaN/Tjb3+/UyYiYNv4WD3/Hlyh9JmiZn+x" +
		"2ReJSzVj9wKBgQDCT+C/cQUAdlhgDrf6yUVc5aOwEYwcEaXrkwkFn+evIArqUh1i" +
		"hSuAN9Ewj56YbPWYJnFuMWETe9kCY3wGOlfLZoEaKz1F+IELE4ctw+Qcyxm0kSW1" +
		"5RWdBJ5bq4n4F4bgasp8Ynshn4FfhGe/5k9hvlzodx+X/Fafa+ZFtlSiSQKBgHS7" +
		"K0AgXF5gICDRacJbcY40AYYntqCZq7Uo8B+2oKq4z1FlfJ6bH6CSZMGzZsFtG4FH" +
		"EB6nfudUEwMNX2uXLDxO40jkmG4rIfiA0NYGglLBhk5P9kKE9xdwxeXTiANqT4IB" +
		"galI9vQ4C9yATn732uWucoYOqLqgdFdUAaT2+fgvAoGAZbbZlaZ5Dy6lQ8hAqP4K" +
		"Rz1f5p2EZaSl0pRbFNXLO164lxQ6OX9FXTctIP+a7jhV+FEHGbnBZTeHfxVct9Pl" +
		"IDchr5O5/1lFp3NJtHYNvTmGQUeiE5at2BOs3b5cwwVbON4fhNnDgGf4MMrsvb4s" +
		"LXdAMRb7pUM5BbySaNc1tCo="
	var appId = "2021000147644101"
	var aliPublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAkE/H+0vLgHnfutCrrMg36lVQnYE/ztgekRDc19/u4Lo3Xfj1S1I4sBNgdW4mwxQHg/XdOVqrsRgevnlyvn2ApMMZYgvrw9fM8xhNGMqIK33ayxchAF9cA+5QerK3d8FSfzJt6g4DlkhKuoh4GD1yK0K69s1lfX1o5oG5Aws+htDb9RRHUaCsQwObrW0gf8170L4Y6uXDrU+k7B4VyNGJ7HRdrPXpJDZiN8QHm0zZXcHEdaoh7gEjYcJyIqKS5N7c8oLZucXaJW93iwC0Uf4g2aZ5P2+xSQfDyJkfP5dhPWCEw8SKJLpB6FzzzA2w3m4IeTcNZ7SrPNMxbtt6WEZjrwIDAQAB"

	var client, err = alipay.New(appId, privateKey, false)
	if err != nil {
		fmt.Println(err)
		return
	}

	//普通公钥模式
	err = client.LoadAliPayPublicKey(aliPublicKey)
	if err != nil {
		log.Println("公钥模式错误")
	}

	//获取数据
	orderId := c.GetString("orderId")
	tPrice, _ := c.GetInt("totalPrice")
	totalPrice := strconv.FormatFloat(float64(tPrice), 'f', 2, 64)

	var p = alipay.TradePagePay{}
	p.NotifyURL = "http://xxx"
	p.ReturnURL = "http://192.168.117.132:8080/user/payOk" //支付成功返回的页面
	p.Subject = "省心租交易平台"
	p.OutTradeNo = orderId
	p.TotalAmount = totalPrice
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	var url, err1 = client.TradePagePay(p)
	if err1 != nil {
		fmt.Println(err)
	}

	// 这个 payURL 即是用于打开支付宝支付页面的 URL，可将输出的内容复制，到浏览器中访问该 URL 即可打开支付页面。
	var payURL = url.String()
	fmt.Println(orderId, totalPrice, payURL)
	c.Redirect(payURL, 302)
}

// 支付成功
func (c *OrderController) HandlePayOk() {
	// 获取订单数据
	orderId := c.GetString("out_trade_no")
	// 检验数据
	if orderId == "" {
		log.Println("支付返回数据错误")
		c.Redirect("/user/userCenterOrder", 302)
		return
	}
	// 处理数据
	/* 更新未支付字段变为已支付*/
	o := orm.NewOrm()
	count, _ := o.QueryTable("OrderInfo").Filter("OrderId", orderId).Update(orm2.Params{"OrderStatus": 1})
	if count == 0 {
		log.Println("更新失败")
		c.Redirect("/user/userCenterOrder", 302)
		return
	}
	// 返回视图
	c.Redirect("/user/userCenterOrder", 302)
}

// 取消支付，租客点击取消支付
func (c *OrderController) HandleQuitPay() {
	userName := GetUser(&c.Controller)
	o := orm.NewOrm()
	var user models.User

	user.Name = userName
	err := o.Read(&user, "Name")
	if err != nil {
		log.Println("读取姓名错误")
		return
	}

	// 获取订单表数据
	orderId := c.GetString("orderId")

	var orderInfo models.OrderInfo
	//查询是否存在
	err = o.QueryTable("OrderInfo").Filter("OrderId", orderId).Filter("User__Id", user.Id).One(&orderInfo)
	if err != nil {
		log.Println("订单不存在")
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
	_, err = o.QueryTable("OrderGoods").Filter("OrderInfo__OrderId", orderId).All(&orderGoods)
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
	_, err = o.QueryTable("OrderGoods").Filter("OrderInfo__OrderId", orderId).Delete()
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

	c.Redirect("/user/userCenterOrder", 302)
}

// 取消订单，房东点击取消订单，只针对自己的订单
func (c *OrderController) HandleQuitOrder() {
	// 1、获取数据
	// 获取订单商品表Id
	Id, err := c.GetInt("Id")
	if err != nil {
		log.Println("获取orderId失败")
		c.Redirect("/user/userCenterMyPublish", 302)
		return
	}

	o := orm.NewOrm()
	// 开启事物
	err = o.Begin()
	if err != nil {
		log.Println("开启事物失败")
		return
	}
	defer o.Commit()

	var orderGoods models.OrderGoods
	// 加锁读取最新库存
	err = o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").Filter("Id", Id).ForUpdate().One(&orderGoods)
	if err != nil {
		log.Println("订单商品表查询失败")
		c.Redirect("/user/userCenterMyPublish", 302)
		return
	}

	// 把库存加回去
	orderGoods.GoodsSKU.Stock += orderGoods.Count
	if _, err = o.Update(orderGoods.GoodsSKU, "Stock"); err != nil {
		log.Println("加回库存失败:", err)
		c.Redirect("/user/userCenterMyPublish", 302)
		return
	}

	// 查询该商品关联的订单是否有其他商品，有就不删除订单表，只删除订单商品表，反之都删掉
	var allGoods []models.OrderGoods
	_, err = o.QueryTable("OrderGoods").RelatedSel("OrderInfo").Filter("OrderInfo__Id", orderGoods.OrderInfo.Id).All(&allGoods)
	if err != nil {
		log.Println("订单表查询失败", err)
		c.Redirect("/user/userCenterMyPublish", 302)
		return
	}

	// 删除orderInfo数据和orderGoods数据
	_, err = o.QueryTable("OrderGoods").Filter("Id", Id).Delete()
	if err != nil {
		log.Println("删除订单商品失败")
		return
	}

	if len(allGoods) == 1 {
		_, err = o.Delete(&orderGoods.OrderInfo)
		if err != nil {
			log.Println("删除主订单数据失败")
			return
		}
	}

	c.Redirect("/user/userCenterMyPublish", 302)
}

// 重新上架
func (c *OrderController) HandleReload() {
	Id, err := c.GetInt("skuId")
	if err != nil {
		log.Println("获取skuId失败")
		c.Redirect("/user/userCenterMyPublish", 302)
		return
	}

	resp := make(map[string]interface{})
	defer c.ServeJSON()

	var orderGoods []models.OrderGoods
	o := orm.NewOrm()

	// 查询所有包含该住房的订单
	_, err = o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").Filter("GoodsSKU__Id", Id).All(&orderGoods)
	if err != nil {
		log.Println("查询错误")
		c.Redirect("/user/userCenterMyPublish", 302)
		return
	}

	// 找订单里面的最晚时间，最晚时间超过一个月才能重新上架
	latestTime := orderGoods[0].OrderInfo.Time
	for _, orderGood := range orderGoods {
		if orderGood.OrderInfo.Time.After(latestTime) {
			latestTime = orderGood.OrderInfo.Time
		}
	}
	placeTimePlusMonth := latestTime.AddDate(0, 1, 0)
	nowString := time.Now().Format("2006-01-02 15:04:05 -0700 MST")
	now, err := time.Parse("2006-01-02 15:04:05 -0700 MST", nowString)
	if err != nil {
		log.Println("解析失败", err)
		resp["code"] = 2
		resp["msg"] = "重新上架失败"
		c.Data["json"] = resp
		return
	}

	log.Println(latestTime, now, placeTimePlusMonth)

	if placeTimePlusMonth.After(now) {
		// 时间未到，不能够进行重新上架
		log.Println("时间未到，不能够进行重新上架")
		resp["code"] = 0
		resp["msg"] = "重新上架时间不少于一月"
		c.Data["json"] = resp
		return
	} else if placeTimePlusMonth.Before(now) {
		// 时间到了，可以进行重新上架
		log.Println("可以进行重新上架")
		var goodsSKU models.GoodsSKU
		err = o.Read(&goodsSKU, "Id")
		if err != nil {
			log.Println("读取对应住房失败", err)
			c.Redirect("/user/userCenterMyPublish", 302)
			return
		}
		// 库存增加
		goodsSKU.Stock += 1
		_, err = o.Update(&goodsSKU)
		if err != nil {
			log.Println("更新库存失败")
			resp["code"] = 3
			resp["msg"] = "重新上架失败"
			c.Data["json"] = resp
			return
		}
	}

	resp["code"] = 1
	resp["msg"] = "上架成功"
	c.Data["json"] = resp
}

// 展示评价页面
func (c *OrderController) ShowComment() {
	userName := GetUser(&c.Controller)
	c.Data["userName"] = userName

	// 1、获取订单表的Id
	Id, err := c.GetInt("Id")
	// 2、校验数据
	if err != nil {
		log.Println("获取订单商品表Id失败")
		return
	}
	// 3、处理数据
	var orderGoods models.OrderGoods
	o := orm.NewOrm()
	o.QueryTable("OrderGoods").RelatedSel("GoodsSKU", "OrderInfo").Filter("Id", Id).One(&orderGoods)
	log.Println(orderGoods)
	c.Data["orderGoods"] = orderGoods

	// 4、返回视图
	c.TplName = "comment.html"
}

// 添加评价
func (c *OrderController) HandleAddComment() {
	// 1、获取数据
	userName := c.GetSession("userName")

	Id, _ := c.GetInt("Id")
	comment := c.GetString("comment")
	if comment == "" {
		log.Println("评价不能为空")
		c.Redirect("/user/addComment?Id="+strconv.Itoa(Id), 302)
		return
	}

	// 2、校验数据
	if userName == nil {
		log.Println("用户名为空")
		c.Redirect("/login", 302)
		return
	}

	// 3、处理数据
	o := orm.NewOrm()
	var orderGoods models.OrderGoods

	o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").Filter("Id", Id).One(&orderGoods)
	orderGoods.Comment = comment
	orderGoods.CommentTime = time.Now().Format("2006-01-02 15:04:05")
	_, err := o.Update(&orderGoods)
	if err != nil {
		log.Println("添加评论失败", err)
		return
	}

	c.Redirect("/user/addComment?Id="+strconv.Itoa(Id), 302)
}

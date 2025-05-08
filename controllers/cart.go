package controllers

import (
	"github.com/beego/beego/v2/adapter/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/gomodule/redigo/redis"
	"log"
	"rentHouses/models"
	"strconv"
)

type CartController struct {
	beego.Controller
}

// 加入购物车功能处理
func (c *CartController) HandleAddCart() {
	//获取数据
	skuid, err1 := c.GetInt("skuid")
	count, err2 := c.GetInt("count")

	//1、设立json容器
	resp := make(map[string]interface{})
	defer c.ServeJSON()
	if err1 != nil || err2 != nil {
		resp["code"] = 1
		resp["msg"] = "传递数据不正确"
		c.Data["json"] = resp
		return
	}
	userName := c.GetSession("userName")
	if userName == nil {
		resp["code"] = 2
		resp["msg"] = "当前用户未登录"
		c.Data["json"] = resp
		return
	}

	//校验数据
	var user models.User
	o := orm.NewOrm()
	user.Name = userName.(string)
	err := o.Read(&user, "Name")
	if err != nil {
		log.Println("读取Name失败")
		return
	}
	//处理数据
	conn, err := redis.Dial("tcp", "192.168.117.132:6379")
	defer conn.Close()
	if err != nil {
		log.Println("redis数据链接错误")
		return
	}

	//先获取原来的数量，给原来的数量加起来
	result, err := conn.Do("hget", "cart_"+strconv.Itoa(user.Id), skuid)
	if err != nil {
		resp["code"] = 3
		resp["msg"] = "redis查询失败"
		c.Data["json"] = resp
		return
	}
	// 处理结果不存在的情况
	var preCount int
	if result == nil {
		preCount = 0 // 商品不存在，数量设为0
	} else {
		preCount, err = redis.Int(result, nil)
		if err != nil {
			resp["code"] = 3
			resp["msg"] = "redis数据解析失败"
			c.Data["json"] = resp
			return
		}
	}
	_, err = conn.Do("hset", "cart_"+strconv.Itoa(user.Id), skuid, count+preCount)
	if err != nil {
		resp["code"] = 3
		resp["msg"] = "redis更改失败"
		c.Data["json"] = resp
		return
	}

	reply, err := conn.Do("hlen", "cart_"+strconv.Itoa(user.Id))
	if err != nil {
		resp["code"] = 4
		resp["msg"] = "获取购物车数量失败"
		c.Data["json"] = resp
		return
	}
	cartCount, _ := redis.Int(reply, err)

	resp["code"] = 5
	resp["msg"] = "ok"

	//返回购物车商品数量个数
	resp["cartCount"] = cartCount

	//3、返回json数据,ServeJSON
	c.Data["json"] = resp
}

// 获取购物车数量函数封装
func GetCartCount(c *beego.Controller) int {
	//从redis获取购物车数量
	userName := c.GetSession("userName")
	if userName == nil {
		return 0
	}
	o := orm.NewOrm()
	var user models.User
	user.Name = userName.(string)
	err := o.Read(&user, "Name")
	if err != nil {
		log.Println("购物车数量、查询用户失败")
		return 0
	}

	conn, err := redis.Dial("tcp", "192.168.117.132:6379")
	defer conn.Close()

	rep, err := conn.Do("hlen", "cart_"+strconv.Itoa(user.Id))
	cartCount, _ := redis.Int(rep, err)

	return cartCount
}

// 显示购物车页面
func (c *CartController) ShowCart() {
	// 用户信息
	userName := GetUser(&c.Controller)

	// 从redis中获取购物车数据
	conn, err := redis.Dial("tcp", "192.168.117.132:6379")
	if err != nil {
		log.Println("连接redis失败")
		return
	}
	defer conn.Close()

	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	err = o.Read(&user, "Name")
	if err != nil {
		log.Println("用户信息查询失败")
		return
	}

	// 用于将Redis的哈希(Hash)类型数据转换为Go中的map[string]int类型
	goodsMap, _ := redis.IntMap(conn.Do("hgetall", "cart_"+strconv.Itoa(user.Id)))

	cartGoods := make([]map[string]interface{}, len(goodsMap))
	i := 0
	totalCount := 0
	totalPrice := 0
	transferPrice := 0
	for index, value := range goodsMap {
		skuid, _ := strconv.Atoi(index)
		var goodsSKU models.GoodsSKU
		goodsSKU.Id = skuid
		err = o.Read(&goodsSKU)
		if err != nil {
			log.Println("读取商品SKU失败")
			return
		}

		temp := make(map[string]interface{})
		temp["goodsSKU"] = goodsSKU
		temp["count"] = value

		totalCount += value
		transferPrice = int(float64(goodsSKU.Price*value) * 0.1)

		totalPrice += goodsSKU.Price*value + transferPrice

		// 小计
		temp["addPrice"] = goodsSKU.Price*value + transferPrice
		// 服务费
		temp["transferPrice"] = transferPrice

		cartGoods[i] = temp
		i++
	}

	// 获取购物车个数
	cartCount := GetCartCount(&c.Controller)
	c.Data["cartCount"] = cartCount

	c.Data["totalCount"] = totalCount
	c.Data["totalPrice"] = totalPrice
	c.Data["cartGoods"] = cartGoods
	c.TplName = "cart.html"
}

// 处理购物车更新
func (c *CartController) HandleCartUpdate() {
	// 获取JSON数据
	skuid, err1 := c.GetInt("skuid")
	count, err2 := c.GetInt("count")
	userName := c.GetSession("userName")
	resp := make(map[string]interface{})
	defer c.ServeJSON()

	// 校验数据
	if err1 != nil || err2 != nil {
		resp["code"] = 1
		resp["msg"] = "请求数据不正确"
		c.Data["json"] = resp
		return
	}

	if userName == nil {
		resp["code"] = 3
		resp["msg"] = "用户未登录"
		c.Data["json"] = resp
		return
	}

	o := orm.NewOrm()
	var user models.User
	user.Name = userName.(string)
	o.Read(&user, "Name")

	// 处理数据
	conn, err := redis.Dial("tcp", "192.168.117.132:6379")
	defer conn.Close()
	if err != nil {
		resp["code"] = 2
		resp["msg"] = "redis数据库连接失败"
		c.Data["json"] = resp
		return
	}
	conn.Do("hset", "cart_"+strconv.Itoa(user.Id), skuid, count)

	resp["code"] = 5
	resp["msg"] = "OK"
	c.Data["json"] = resp
}

// 处理购物车删除
func (c *CartController) HandleCartDelete() {
	// 获取JSON数据
	skuid, err := c.GetInt("skuid")
	userName := c.GetSession("userName")
	// 校验JSON数据
	resp := make(map[string]interface{})
	defer c.ServeJSON()
	if err != nil {
		resp["code"] = 1
		resp["msg"] = "无效商品ID"
		c.Data["json"] = resp
		return
	}
	if userName == nil {
		resp["code"] = 3
		resp["msg"] = "用户未登录"
		c.Data["json"] = resp
		return
	}
	// 处理JSON数据删除
	o := orm.NewOrm()
	var user models.User
	user.Name = userName.(string)
	err = o.Read(&user, "Name")
	if err != nil {
		return
	}

	conn, err := redis.Dial("tcp", "192.168.117.132:6379")
	defer conn.Close()
	if err != nil {
		resp["code"] = 2
		resp["msg"] = "redis数据连接失败"
		c.Data["json"] = resp
		return
	}
	_, err = conn.Do("hdel", "cart_"+strconv.Itoa(user.Id), skuid)
	if err != nil {
		resp["code"] = 3
		resp["error"] = "删除商品失败"
		c.Data["json"] = resp
		return
	}

	// 删除成功返回的数据
	resp["code"] = 5
	resp["msg"] = "OK"
	// 返回购物车数量
	cartCount := GetCartCount(&c.Controller)
	resp["cartCount"] = cartCount

	c.Data["json"] = resp
}

package controllers

import (
	"errors"
	"github.com/beego/beego/v2/adapter/orm"
	orm2 "github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/gomodule/redigo/redis"
	"log"
	"math"
	"rentHouses/models"
	"strconv"
)

type GoodsController struct {
	beego.Controller
}

// 判断
func GetUser(c *beego.Controller) string {
	userName := c.GetSession("userName")
	if userName == nil {
		c.Data["userName"] = ""
	} else {
		c.Data["userName"] = userName.(string)
		return userName.(string)
	}
	return ""
}

// 展示首页
func (c *GoodsController) ShowIndex() {
	GetUser(&c.Controller)
	o := orm.NewOrm()

	//1、获取类型数据
	var goodsTypes []models.GoodsType
	_, err := o.QueryTable("GoodsType").All(&goodsTypes)
	if err != nil {
		log.Println("获取类型数据失败")
		return
	}
	c.Data["goodsTypes"] = goodsTypes

	//2、获取轮播图数据
	var indexGoodsBanner []models.IndexGoodsBanner
	_, err = o.QueryTable("IndexGoodsBanner").OrderBy("Index").All(&indexGoodsBanner)
	if err != nil {
		log.Println("获取轮播图数据失败")
		return
	}
	c.Data["indexGoodsBanner"] = indexGoodsBanner

	//3、获取促销商品数据
	var promotionGoods []models.IndexPromotionBanner
	_, err = o.QueryTable("IndexPromotionBanner").All(&promotionGoods)
	if err != nil {
		return
	}
	c.Data["promotionGoods"] = promotionGoods

	//4、首页展示商品数据
	goods := make([]map[string]interface{}, len(goodsTypes))

	//向切片interface中插入类型数据
	for index, goodsType := range goodsTypes {
		temp := make(map[string]interface{})
		temp["type"] = goodsType
		goods[index] = temp
	}

	for _, value := range goods {
		var textGoods []models.IndexTypeGoodsBanner
		var imgGoods []models.IndexTypeGoodsBanner
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU", "GoodsType").OrderBy("Index").Filter("GoodsType", value["type"]).Filter("DisplayType", 0).Filter("GoodsSKU__Stock", 1).All(&textGoods)
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU", "GoodsType").OrderBy("Index").Filter("GoodsType", value["type"]).Filter("DisplayType", 1).Filter("GoodsSKU__Stock", 1).All(&imgGoods)

		value["textGoods"] = textGoods
		value["imgGoods"] = imgGoods
	}
	c.Data["goods"] = goods

	// 获取购物车数量
	cartCount := GetCartCount(&c.Controller)
	c.Data["cartCount"] = cartCount

	// 返回视图
	c.TplName = "index.html"
}

// 视图布局函数
func ShowLayout(c *beego.Controller) {
	//查询类型
	o := orm.NewOrm()
	var types []models.GoodsType
	_, err := o.QueryTable("GoodsType").All(&types)
	if err != nil {
		log.Println("查询类型错误")
		return
	}
	c.Data["types"] = types
	//获取用户信息
	GetUser(c)
	c.Layout = "goodsLayout.html"
}

// 展示商品详情页
func (c *GoodsController) ShowDetail() {
	// 1、获取数据
	id, err := c.GetInt("id")
	// 2、校验数据
	if err != nil {
		log.Println("获取商品详情id数据失败")
		c.Redirect("/", 302)
		return
	}
	// 3、处理数据
	o := orm.NewOrm()
	//获取商品详情
	var goodsSKU models.GoodsSKU
	goodsSKU.Id = id
	err = o.QueryTable("GoodsSKU").RelatedSel("GoodsType", "Goods").Filter("Id", id).One(&goodsSKU)
	if err != nil {
		return
	}
	c.Data["goodsSKU"] = goodsSKU

	//获取同类型时间靠前的两条房屋数据
	var goodsNew []models.GoodsSKU
	_, err = o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType", goodsSKU.GoodsType).OrderBy("Time").Limit(2, 0).All(&goodsNew)
	if err != nil {
		log.Println("查询新房错误")
		return
	}
	c.Data["goodsNew"] = goodsNew

	// 添加历史浏览记录:首先检测用户是否登录
	userName := c.GetSession("userName")
	if userName != nil {
		//查询用户信息
		o := orm.NewOrm()
		var user models.User
		user.Name = userName.(string)
		err := o.Read(&user, "Name")
		if err != nil {
			log.Println("查询用户信息失败")
			return
		}
		//添加历史记录,用redis存储
		conn, err := redis.Dial("tcp", "192.168.117.132:6379")
		defer conn.Close()
		if err != nil {
			log.Println("redis链接错误")
			return
		}
		//先把以前的对应用户的记录清空，再插入，实现只保留一次的记录
		_, err = conn.Do("lrem", "history_"+strconv.Itoa(user.Id), 0, id)
		if err != nil {
			log.Println("清空用户历史记录失败", err)
			return
		}
		_, err = conn.Do("lpush", "history_"+strconv.Itoa(user.Id), id)
		if err != nil {
			log.Println("插入用户历史记录失败")
			return
		}
	}

	/* 展示评价模块 */
	var orderGoods []models.OrderGoods
	_, err = o.QueryTable("OrderGoods").RelatedSel("GoodsSKU", "OrderInfo").RelatedSel("OrderInfo__User").Filter("GoodsSKU__Id", goodsSKU.Id).OrderBy("-CommentTime").Limit(7, 0).All(&orderGoods)
	if err != nil {
		log.Println("订单评价查询失败", err)
	}
	c.Data["orderGoods"] = orderGoods

	// 4、视图返回
	cartCount := GetCartCount(&c.Controller)
	c.Data["cartCount"] = cartCount

	ShowLayout(&c.Controller)
	c.TplName = "detail.html"
}

// 分页函数
func splitPage(pageCount int, pageIndex int) []int {
	var pages []int
	// 如果信息总页数小于等于5页
	if pageCount <= 5 {
		// 全部写入切片返回,即全部页数索引都展示
		pages = make([]int, pageCount)
		for i := 0; i < pageCount; i++ {
			pages[i] = i + 1
		}
	} else if pageIndex <= 3 {
		// 如果当前页面超过5页且已经点击到了前3页,就只显示前5页
		pages = []int{1, 2, 3, 4, 5}
	} else if pageIndex > pageCount-3 {
		// 如果当前页面超过了5页且已经点击到了后3页，则只显示后5页
		pages = []int{pageCount - 4, pageCount - 3, pageCount - 2, pageCount - 1, pageCount}
	} else {
		// 反之页面超过5页且在中间某个部分时，则显示当前索引页的前后两个页面
		pages = []int{pageIndex - 2, pageIndex - 1, pageIndex, pageIndex + 1, pageIndex + 2}
	}
	return pages
}

// 展示商品列表页
func (c *GoodsController) ShowGoodsList() {
	typeId, err := c.GetInt("typeId")
	if err != nil {
		log.Println("获取商品列表id失败")
		c.Redirect("/", 302)
		return
	}

	//获取新品数据
	o := orm.NewOrm()
	var goodsNew []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).OrderBy("Time").Limit(2, 0).All(&goodsNew)
	c.Data["goodsNew"] = goodsNew

	//获取相应类型数据
	var goodsSKU []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).All(&goodsSKU)
	c.Data["goodsSKU"] = goodsSKU

	//获取总的类型
	var goodsType models.GoodsType
	goodsType.Id = typeId
	err = o.Read(&goodsType, "Id")
	if err != nil {
		c.Redirect("/", 302)
		return
	}
	c.Data["goodsType"] = goodsType

	//实现分页
	//1、计算页码
	count, err := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).Filter("Stock", 1).Count()
	if err != nil {
		log.Println("查询类型个数失败")
	}
	pageSize := 10
	pageCount := math.Ceil(float64(count) / float64(pageSize))

	pageIndex, err := c.GetInt("pageIndex")
	if err != nil {
		pageIndex = 1
	}

	pages := splitPage(int(pageCount), pageIndex)
	c.Data["pages"] = pages
	c.Data["typeId"] = typeId
	c.Data["pageIndex"] = pageIndex

	start := (pageIndex - 1) * pageSize

	//获取上页页码
	prePage := pageIndex - 1
	if prePage <= 1 {
		pageIndex = 1
	}
	c.Data["prePage"] = prePage

	//获取下页页码
	nextPage := pageIndex + 1
	if nextPage > int(pageCount) {
		nextPage = int(pageCount)
	}
	c.Data["nextPage"] = nextPage

	//按照顺序获取商品
	sort := c.GetString("sort")
	if sort == "" {
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).Filter("Stock", 1).Limit(pageSize, start).All(&goodsSKU)
		c.Data["sort"] = ""
		c.Data["goodsSKU"] = goodsSKU
	} else if sort == "price" {
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).Filter("Stock", 1).OrderBy("Price").Limit(pageSize, start).All(&goodsSKU)
		c.Data["sort"] = "price"
		c.Data["goodsSKU"] = goodsSKU
	} else {
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).Filter("Stock", 1).OrderBy("Sales").Limit(pageSize, start).All(&goodsSKU)
		c.Data["sort"] = "sale"
		c.Data["goodsSKU"] = goodsSKU
	}

	cartCount := GetCartCount(&c.Controller)
	c.Data["cartCount"] = cartCount

	// 返回视图
	ShowLayout(&c.Controller)
	c.TplName = "list.html"
}

// 处理地址搜索
func (c *GoodsController) HandleGoodsSearch() {
	//获取数据
	goodsAddr := c.GetString("goodsAddr")
	c.Data["goodsAddr"] = goodsAddr

	o := orm.NewOrm()
	var goodsSKU []models.GoodsSKU
	//校验数据
	if goodsAddr == "" {
		_, err := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").All(&goodsSKU)
		if errors.Is(err, orm2.ErrNoRows) {
			c.Data["goodsSKU"] = ""
		} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
			log.Println("查询失败")
			c.Redirect("/", 302)
			return
		}
	}

	sort := c.GetString("sort")
	if sort == "" {
		_, err := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("Addr__icontains", goodsAddr).All(&goodsSKU)
		if errors.Is(err, orm2.ErrNoRows) {
			log.Println("对应名称查询为空")
		} else if !errors.Is(err, orm2.ErrNoRows) && err != nil {
			log.Println("默认分类失败")
			return
		}

		c.Data["sort"] = ""
		c.Data["goodsSKU"] = goodsSKU
	} else if sort == "price" {
		_, err := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("Addr__icontains", goodsAddr).OrderBy("Price").All(&goodsSKU)
		if err != nil {
			log.Println("价格分类失败")
			return
		}
		c.Data["sort"] = "price"
		c.Data["goodsSKU"] = goodsSKU
	} else {
		_, err := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("Addr__icontains", goodsAddr).OrderBy("Sales").All(&goodsSKU)
		if err != nil {
			log.Println("销量分类失败")
			return
		}
		c.Data["sort"] = "sale"
		c.Data["goodsSKU"] = goodsSKU
	}

	//获取新品数据
	var goodsNew []models.GoodsSKU
	_, err := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").OrderBy("Time").Limit(2, 0).All(&goodsNew)
	if !errors.Is(err, orm2.ErrNoRows) && err != nil {
		log.Println("新品查询失败")
		c.Redirect("/", 302)
		return
	}
	c.Data["goodsNew"] = goodsNew

	// 获取购物车数量
	cartCount := GetCartCount(&c.Controller)
	c.Data["cartCount"] = cartCount

	// 返回视图
	ShowLayout(&c.Controller)
	c.TplName = "search.html"
}

package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"log"
	"rentHouses/controllers"
)

func init() {
	// beego.Router("/", &controllers.MainController{})
	// 用户端过滤
	beego.InsertFilter("/user/*", beego.BeforeRouter, filter)
	// 管理员端过滤
	beego.InsertFilter("/administer/*", beego.BeforeRouter, administerFilter)

	// 管理员-注册
	beego.Router("/administerRegister", &controllers.AdministerController{}, "get:ShowAdministerRegister;post:HandleAdministerRegister")
	// 管理员-登陆
	beego.Router("/administerLogin", &controllers.AdministerController{}, "get:ShowAdministerLogin;post:HandleAdministerLogin")
	// 管理员-退出
	beego.Router("/administer/logout", &controllers.AdministerController{}, "get:HandleAdministerLogout")
	// 管理员-用户模块展示
	beego.Router("/administer/showUserModule", &controllers.AdministerController{}, "get:ShowUserModule")
	// 管理员-评论模块展示
	beego.Router("/administer/showCommentModule", &controllers.AdministerController{}, "get:ShowCommentModule")
	// 管理员-住房模块展示
	beego.Router("/administer/showHouseModule", &controllers.AdministerController{}, "get:ShowHouseModule")
	// 管理员-订单模块展示
	beego.Router("/administer/showOrderModule", &controllers.AdministerController{}, "get:ShowOrderModule")
	// 管理员-数据统计模块展示
	beego.Router("/administer/showStatsModule", &controllers.AdministerController{}, "get:ShowStatsModule")
	// 管理员-用户搜索与删除
	beego.Router("/administer/userOperation", &controllers.AdministerController{}, "get:HandleUserSearch;post:HandleUserDelete")
	// 管理员-评论搜索与删除
	beego.Router("/administer/commentOperation", &controllers.AdministerController{}, "get:HandleCommentSearch;post:HandleCommentDelete")
	// 管理员-住房搜索与下架
	beego.Router("/administer/houseOperation", &controllers.AdministerController{}, "get:HandleHouseSearch;post:HandleHouseDelete")
	// 管理员-订单搜索与取消
	beego.Router("/administer/orderOperation", &controllers.AdministerController{}, "get:HandleOrderSearch;post:HandleOrderDelete")
	// 管理员-订单确认
	beego.Router("/administer/orderConfirm", &controllers.AdministerController{}, "post:HandleOrderConfirm")

	// 用户-注册
	beego.Router("/register", &controllers.UserController{}, "get:ShowRegister;post:HandleRegister")
	// 用户-激活
	beego.Router("/active", &controllers.UserController{}, "get:HandleActive")
	// 用户-登陆
	beego.Router("/login", &controllers.UserController{}, "get:ShowLogin;post:HandleLogin")
	// 主页面显示
	beego.Router("/", &controllers.GoodsController{}, "get:ShowIndex")
	// 用户-退出
	beego.Router("/user/logout", &controllers.UserController{}, "get:HandleLogout")
	// 展示用户中心
	beego.Router("/user/userCenterInfo", &controllers.UserController{}, "get:ShowUserCenterInfo")
	// 展示用户订单页
	beego.Router("/user/userCenterOrder", &controllers.UserController{}, "get:ShowUserCenterOrder")
	// 展示我的发布页
	beego.Router("/user/userCenterMyPublish", &controllers.UserController{}, "get:ShowUserCenterMyPublish")
	// 处理商品下架
	beego.Router("/user/goodsDelete", &controllers.UserController{}, "get:HandleGoodsDelete")
	// 住房信息更新
	beego.Router("/user/goodsUpdate", &controllers.UserController{}, "get:ShowGoodsUpdate;post:HandleGoodsUpdate")
	// 展示发布房源页
	beego.Router("/user/userCenterPublish", &controllers.UserController{}, "get:ShowUserCenterPublish;post:HandleUserCenterPublish")
	// 展示用户地址页
	beego.Router("/user/userCenterSite", &controllers.UserController{}, "get:ShowUserCenterSite;post:HandleUserCenterSite")
	// 展示列表页
	beego.Router("/goodsList", &controllers.GoodsController{}, "get:ShowGoodsList")
	// 商品搜索
	beego.Router("/goodsSearch", &controllers.GoodsController{}, "get:HandleGoodsSearch")
	// 展示商品详情页
	beego.Router("/user/goodsDetail", &controllers.GoodsController{}, "get:ShowDetail")
	// 添加购物车
	beego.Router("/user/addCart", &controllers.CartController{}, "post:HandleAddCart")
	// 显示购物车页面
	beego.Router("/user/cart", &controllers.CartController{}, "get:ShowCart")
	// 购物车更新
	beego.Router("/user/cartUpdate", &controllers.CartController{}, "post:HandleCartUpdate")
	// 购物车删除
	beego.Router("/user/cartDelete", &controllers.CartController{}, "post:HandleCartDelete")
	// 订单页面展示
	beego.Router("/user/showOrder", &controllers.OrderController{}, "post:ShowOrder")
	// 添加订单
	beego.Router("/user/addOrder", &controllers.OrderController{}, "post:HandleAddOrder")
	// 支付
	beego.Router("/user/pay", &controllers.OrderController{}, "get:HandlePay")
	// 取消支付
	beego.Router("/user/quitPay", &controllers.OrderController{}, "get:HandleQuitPay")
	// 取消订单
	beego.Router("/user/quitOrder", &controllers.OrderController{}, "get:HandleQuitOrder")
	// 重新上架
	beego.Router("/user/reload", &controllers.OrderController{}, "post:HandleReload")
	// 支付成功跳转
	beego.Router("/user/payOk", &controllers.OrderController{}, "get:HandlePayOk")
	// 评价页面
	beego.Router("/user/addComment", &controllers.OrderController{}, "get:ShowComment;post:HandleAddComment")
}

// 添加过滤器
func filter(ctx *context.Context) {
	// 判断user.go的所有操作之前是否登陆了用户，即Session有无数据
	userName := ctx.Input.Session("userName")
	if userName == nil {
		log.Println("用户未登录")
		ctx.Redirect(302, "/login")
		return
	}
}

// 添加过滤器
func administerFilter(ctx *context.Context) {
	// 判断user.go的所有操作之前是否登陆了用户，即Session有无数据
	userName := ctx.Input.Session("userName")
	if userName == nil {
		log.Println("未登录")
		ctx.Redirect(302, "/administerLogin")
		return
	}
}

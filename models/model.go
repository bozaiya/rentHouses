package models

import (
	"github.com/beego/beego/v2/adapter/orm"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
)

type User struct { //用户表
	Id        int
	Name      string       `orm:"size(20);unique"` //用户名
	PassWord  string       `orm:"size(20)"`        //登陆密码
	Email     string       `orm:"size(50)"`        //邮箱
	Active    bool         `orm:"default(false)"`  //是否激活 0 表示未激活  1表示激活
	Power     int          `orm:"default(0)"`      //权限设置 0表示普通用户 1表示管理员用户
	Time      string       //最后一次登陆时间
	Address   []*Address   `orm:"reverse(many)"`
	OrderInfo []*OrderInfo `orm:"reverse(many)"`

	// 关联发布商品
	GoodsSKU []*GoodsSKU `orm:"reverse(many)"`
}

type Address struct { //地址表
	Id        int
	Receiver  string       `orm:"size(20)"`       //联系人
	Addr      string       `orm:"size(50)"`       //联系地址
	Zipcode   string       `orm:"size(20)"`       //邮编
	Phone     string       `orm:"size(20)"`       //联系方式
	IsDefault bool         `orm:"default(false)"` //是否默认 0 为非默认  1为默认
	User      *User        `orm:"rel(fk)"`        //用户ID
	OrderInfo []*OrderInfo `orm:"reverse(many)"`
}

type Goods struct { //商品SPU表
	Id       int
	Name     string      `orm:"size(20)"`  //商品名称
	Detail   string      `orm:"size(200)"` //详细描述
	GoodsSKU []*GoodsSKU `orm:"reverse(many)"`
}

type GoodsType struct { //商品类型表
	Id                   int
	Name                 string                  //种类名称
	Logo                 string                  //logo
	Image                string                  //图片
	GoodsSKU             []*GoodsSKU             `orm:"reverse(many)"`
	IndexTypeGoodsBanner []*IndexTypeGoodsBanner `orm:"reverse(many)"`
}

type GoodsSKU struct { //商品SKU表
	Id                   int
	Goods                *Goods                  `orm:"rel(fk)"` //商品SPU
	GoodsType            *GoodsType              `orm:"rel(fk)"` //商品所属种类
	Name                 string                  //商品名称
	Desc                 string                  //商品简介
	Price                int                     //商品价格
	Unite                string                  `orm:"default(RMB)"` //商品单位
	Image                string                  //商品图片
	Addr                 string                  `orm:"size(50)"`   //商品地址
	Phone                string                  `orm:"size(20)"`   //联系方式
	Stock                int                     `orm:"default(1)"` //商品库存
	Sales                int                     `orm:"default(0)"` //商品销量
	Status               int                     `orm:"default(1)"` //商品状态
	Time                 string                  //上架时间
	GoodsImage           []*GoodsImage           `orm:"reverse(many)"`
	IndexGoodsBanner     []*IndexGoodsBanner     `orm:"reverse(many)"`
	IndexTypeGoodsBanner []*IndexTypeGoodsBanner `orm:"reverse(many)"`
	OrderGoods           []*OrderGoods           `orm:"reverse(many)"`

	User *User `orm:"rel(fk)"` //关联上传该商品的user
}

type GoodsImage struct { //商品图片表
	Id       int
	Image    string    //商品图片
	GoodsSKU *GoodsSKU `orm:"rel(fk)"` //商品SKU
}
type IndexGoodsBanner struct { //首页轮播商品展示表
	Id       int
	GoodsSKU *GoodsSKU `orm:"rel(fk)"` //商品sku
	Image    string    //商品图片
	Index    int       `orm:"default(0)"` //展示顺序
}

type IndexTypeGoodsBanner struct { //首页分类商品展示表
	Id          int
	GoodsType   *GoodsType `orm:"rel(fk)"`    //商品类型
	GoodsSKU    *GoodsSKU  `orm:"rel(fk)"`    //商品sku
	DisplayType int        `orm:"default(1)"` //展示类型 0代表文字，1代表图片
	Index       int        `orm:"default(0)"` //展示顺序
}

type IndexPromotionBanner struct { //首页促销商品展示表
	Id    int
	Name  string `orm:"size(20)"` //活动名称
	Url   string `orm:"size(50)"` //活动链接
	Image string //活动图片
	Index int    `orm:"default(0)"` //展示顺序
}

type OrderInfo struct { //订单表（包含多个订单商品表）
	Id            int
	OrderId       string        `orm:"unique"`
	User          *User         `orm:"rel(fk)"` //用户
	Address       *Address      `orm:"rel(fk)"` //地址
	PayMethod     int           //付款方式
	TotalCount    int           `orm:"default(1)"` //商品数量
	TotalPrice    int           //支付总价
	TransitPrice  int           //代理费
	OrderStatus   int           `orm:"default(0)"`   //订单状态
	TradeNo       string        `orm:"default('')"`  //支付编号
	Time          time.Time     `orm:"auto_now_add"` //下单时间
	ConfirmStatus int           `orm:"default(0)"`   //是否确认
	OrderGoods    []*OrderGoods `orm:"reverse(many)"`
}

type OrderGoods struct { //订单商品表
	Id        int
	OrderInfo *OrderInfo `orm:"rel(fk)"`    //订单
	GoodsSKU  *GoodsSKU  `orm:"rel(fk)"`    //商品
	Count     int        `orm:"default(1)"` //商品数量
	Price     int        //商品价格
	Comment   string     //评价内容

	CommentTime string //评价时间
}

func init() {
	// set default database
	err := orm.RegisterDataBase("default", "mysql", "root:123456@tcp(127.0.0.1:3306)/rentHouses?charset=utf8")
	if err != nil {
		log.Println("没连上")
		return
	}

	// register model
	orm.RegisterModel(new(User), new(Address), new(OrderGoods), new(OrderInfo), new(IndexPromotionBanner), new(IndexTypeGoodsBanner), new(IndexGoodsBanner), new(GoodsImage), new(GoodsSKU), new(GoodsType), new(Goods))

	// create table
	err = orm.RunSyncdb("default", false, true)
	if err != nil {
		log.Println("创建表失败")
		return
	}
}

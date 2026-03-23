package api

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var lokasiWita = time.FixedZone("WITA", 8*3600)

type Product struct {
	Id_Jaja   int    `gorm:"primaryKey;column:id_jaja" json:"Id_Jaja"`
	Nama_Jaja string `gorm:"column:nama_jaja" json:"Nama_Jaja"`
	Harga     int    `gorm:"column:harga" json:"Harga"`
}

func (Product) TableName() string { return "products" }

type Transaction struct {
	ID               int       `gorm:"primaryKey;column:id" json:"id"`
	NamaPembeli      string    `gorm:"column:nama_pembeli" json:"nama_pembeli"`
	TotalBelanja     int       `gorm:"column:total_belanja" json:"total_belanja"`
	TanggalTransaksi time.Time `gorm:"column:tanggal_transaksi" json:"tanggal_transaksi"`
}

func (Transaction) TableName() string { return "transactions" }

type TransactionDetail struct {
	ID            int `gorm:"primaryKey;column:id" json:"id"`
	TransactionID int `gorm:"column:transaction_id" json:"transaction_id"`
	ProductID     int `gorm:"column:product_id" json:"product_id"`
	Kuantitas     int `gorm:"column:kuantitas" json:"kuantitas"`
}

func (TransactionDetail) TableName() string { return "transaction_details" }

type RequestPesanan struct {
	NamaPembeli string `json:"nama_pembeli"`
	TotalHarga  int    `json:"total_harga"`
	Keranjang   []struct {
		ID  int `json:"id"`
		Qty int `json:"qty"`
	} `json:"keranjang"`
}

type DetailResponse struct {
	NamaJaja  string `json:"nama_jaja"`
	Harga     int    `json:"harga"`
	Kuantitas int    `json:"kuantitas"`
	Subtotal  int    `json:"subtotal"`
}

var app *gin.Engine

func init() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "root:4125Des.@tcp(127.0.0.1:3306)/db_kasir_jaja?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		if err != nil {
			c.JSON(500, gin.H{"error": "DB Error"})
			c.Abort()
			return
		}
		c.Set("db", db)
		c.Next()
	})

	if err == nil {
		db.AutoMigrate(&Product{}, &Transaction{}, &TransactionDetail{})
	}

	r.GET("/api/products", func(c *gin.Context) {
		database := c.MustGet("db").(*gorm.DB)
		var products []Product
		database.Find(&products)
		c.JSON(200, gin.H{"data": products})
	})

	r.POST("/api/products", func(c *gin.Context) {
		database := c.MustGet("db").(*gorm.DB)
		var newProduct Product
		if err := c.ShouldBindJSON(&newProduct); err == nil {
			database.Create(&newProduct)
			c.JSON(200, gin.H{"pesan": "Sukses"})
		}
	})

	r.DELETE("/api/products/:id", func(c *gin.Context) {
		database := c.MustGet("db").(*gorm.DB)
		id := c.Param("id")
		database.Delete(&Product{}, "id_jaja = ?", id)
		c.JSON(200, gin.H{"pesan": "Menu berhasil dihapus!"})
	})

	r.POST("/api/transactions", func(c *gin.Context) {
		database := c.MustGet("db").(*gorm.DB)
		var req RequestPesanan
		if err := c.ShouldBindJSON(&req); err != nil {
			return
		}

		waktuSekarang := time.Now().In(lokasiWita)
		trx := Transaction{NamaPembeli: req.NamaPembeli, TotalBelanja: req.TotalHarga, TanggalTransaksi: waktuSekarang}
		database.Create(&trx)

		for _, item := range req.Keranjang {
			database.Create(&TransactionDetail{TransactionID: trx.ID, ProductID: item.ID, Kuantitas: item.Qty})
		}
		c.JSON(200, gin.H{"pesan": "Sukses!", "id_nota": trx.ID, "tanggal": trx.TanggalTransaksi.Format("02 Jan 2006, 15:04")})
	})

	r.GET("/api/transactions", func(c *gin.Context) {
		database := c.MustGet("db").(*gorm.DB)
		var trxs []Transaction
		database.Order("tanggal_transaksi desc").Find(&trxs)
		c.JSON(200, gin.H{"data": trxs})
	})

	r.GET("/api/transactions/:id/details", func(c *gin.Context) {
		database := c.MustGet("db").(*gorm.DB)
		id := c.Param("id")
		var details []DetailResponse
		database.Table("transaction_details").
			Select("products.nama_jaja, products.harga, transaction_details.kuantitas, (products.harga * transaction_details.kuantitas) as subtotal").
			Joins("JOIN products ON products.id_jaja = transaction_details.product_id").
			Where("transaction_details.transaction_id = ?", id).
			Scan(&details)
		c.JSON(200, gin.H{"data": details})
	})

	// FITUR HAPUS TRANSAKSI BERSERTA DETAILNYA
	r.DELETE("/api/transactions/:id", func(c *gin.Context) {
		database := c.MustGet("db").(*gorm.DB)
		id := c.Param("id")
		database.Where("transaction_id = ?", id).Delete(&TransactionDetail{})
		database.Delete(&Transaction{}, id)
		c.JSON(200, gin.H{"pesan": "Nota berhasil dihapus!"})
	})

	app = r
}

func Handler(w http.ResponseWriter, r *http.Request) {
	app.ServeHTTP(w, r)
}

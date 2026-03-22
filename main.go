package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

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
	TanggalTransaksi time.Time `gorm:"column:tanggal_transaksi;autoCreateTime" json:"tanggal_transaksi"`
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

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {

		dsn = "root:4125Des.@tcp(127.0.0.1:3306)/db_kasir_jaja?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal connect: ", err)
	}

	db.AutoMigrate(&Product{}, &Transaction{}, &TransactionDetail{})

	r := gin.Default()

	// 1. Ambil Menu (GET)
	r.GET("/api/products", func(c *gin.Context) {
		var products []Product
		db.Find(&products)
		c.JSON(200, gin.H{"data": products})
	})

	// 2. Tambah Menu (POST)
	r.POST("/api/products", func(c *gin.Context) {
		var newProduct Product
		if err := c.ShouldBindJSON(&newProduct); err != nil {
			return
		}
		db.Create(&newProduct)
		c.JSON(200, gin.H{"pesan": "Menu baru berhasil ditambahkan!"})
	})

	// 3. FITUR BARU: Hapus Menu (DELETE)
	r.DELETE("/api/products/:id", func(c *gin.Context) {
		id := c.Param("id")
		// Menyuruh MySQL menghapus produk berdasarkan ID
		db.Delete(&Product{}, "id_jaja = ?", id)
		c.JSON(200, gin.H{"pesan": "Menu berhasil dihapus!"})
	})

	// 4. Simpan Transaksi (POST)
	r.POST("/api/transactions", func(c *gin.Context) {
		var req RequestPesanan
		if err := c.ShouldBindJSON(&req); err != nil {
			return
		}

		trx := Transaction{NamaPembeli: req.NamaPembeli, TotalBelanja: req.TotalHarga}
		db.Create(&trx)

		for _, item := range req.Keranjang {
			db.Create(&TransactionDetail{TransactionID: trx.ID, ProductID: item.ID, Kuantitas: item.Qty})
		}
		c.JSON(200, gin.H{"pesan": "Sukses!", "id_nota": trx.ID, "tanggal": trx.TanggalTransaksi.Format("02 Jan 2006, 15:04")})
	})

	// 5. Ambil Rekapan (GET)
	r.GET("/api/transactions", func(c *gin.Context) {
		var trxs []Transaction
		db.Order("tanggal_transaksi desc").Find(&trxs)
		c.JSON(200, gin.H{"data": trxs})
	})

	// 6. Ambil Detail Belanjaan Berdasarkan ID Nota (GET)
	r.GET("/api/transactions/:id/details", func(c *gin.Context) {
		id := c.Param("id")
		var details []DetailResponse
		db.Table("transaction_details").
			Select("products.nama_jaja, products.harga, transaction_details.kuantitas, (products.harga * transaction_details.kuantitas) as subtotal").
			Joins("JOIN products ON products.id_jaja = transaction_details.product_id").
			Where("transaction_details.transaction_id = ?", id).
			Scan(&details)
		c.JSON(200, gin.H{"data": details})
	})

	r.StaticFile("/", "./index.html")

	// BACA PORT DARI SERVER CLOUD
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server jalan di port " + port)
	r.Run(":" + port)
}

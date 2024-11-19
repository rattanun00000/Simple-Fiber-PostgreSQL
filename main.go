package main

import (
	"database/sql"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

var db *sql.DB

func SetupDatabase() *sql.DB {
	const connectionString = "user=myuser password=mypassword dbname=mydatabase sslmode=disable"
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	return db
}

type Product struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Category string `json:"category"`
}

// validate ตรวจสอบความถูกต้องของข้อมูล Product
func (p *Product) validate() error {
	if p.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Name is required")
	}
	if p.Price <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Price must be greater than 0")
	}
	if p.Category == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Category is required")
	}
	return nil
}

func CreateProduct(c *fiber.Ctx) error {
	p := new(Product)
	if err := c.BodyParser(p); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if err := p.validate(); err != nil {
		return err
	}

	// Insert product into database
	err := db.QueryRow(
		"INSERT INTO products (name, price, category) VALUES ($1, $2, $3) RETURNING id",
		p.Name, p.Price, p.Category,
	).Scan(&p.ID)

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Could not create product")
	}

	return c.Status(fiber.StatusCreated).JSON(p)
}

func GetProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	var p Product

	err := db.QueryRow(
		"SELECT id, name, price, category FROM products WHERE id = $1",
		id,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Category)

	if err != nil {
		if err == sql.ErrNoRows {
			return fiber.NewError(fiber.StatusNotFound, "Product not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Could not retrieve product")
	}

	return c.JSON(p)
}

func UpdateProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	p := new(Product)

	if err := c.BodyParser(p); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if err := p.validate(); err != nil {
		return err
	}

	result, err := db.Exec(
		"UPDATE products SET name = $1, price = $2, category = $3 WHERE id = $4",
		p.Name, p.Price, p.Category, id,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Could not update product")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Could not verify update")
	}
	if rowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Product not found")
	}

	p.ID, _ = strconv.Atoi(id)
	return c.JSON(p)
}

func DeleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")

	result, err := db.Exec("DELETE FROM products WHERE id = $1", id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Could not delete product")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Could not verify deletion")
	}
	if rowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Product not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func main() {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Default error handling
			code := fiber.StatusInternalServerError
			message := "Internal Server Error"

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			}

			return c.Status(code).JSON(fiber.Map{
				"error": message,
			})
		},
	})

	db = SetupDatabase()
	defer db.Close()

	// Set up routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello world")
	})
	app.Post("/products", CreateProduct)
	app.Get("/products/:id", GetProduct)
	app.Put("/products/:id", UpdateProduct)
	app.Delete("/products/:id", DeleteProduct)

	log.Fatal(app.Listen(":3000"))
}

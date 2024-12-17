package handler

import (
	"github.com/gofiber/fiber/v2"
	"main/model"
	"time"
)

var cards = make([]model.Card, 0)

func findId(id string) int {
	var cardIndex = -1
	for i, n := range cards {
		if n.Id == id {
			cardIndex = i
		}
	}
	return cardIndex
}

func deleteFromCards(cardIndex int) {
	cards[cardIndex] = cards[len(cards)-1]
	cards[len(cards)-1] = model.Card{}
	cards = cards[:len(cards)-1]
}

func GetAllCards(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "success", "data": &cards})
}

func GetCard(c *fiber.Ctx) error {
	id := c.Params("id")
	var cardIndex = findId(id)
	var card = cards[cardIndex]

	if card.Content == "" {
		return c.JSON(fiber.Map{"status": "error", "message": "карты с таким Id не найдено", "data": nil})
	}

	return c.JSON(fiber.Map{"status": "success", "data": card})
}

func CreateCard(c *fiber.Ctx) error {
	card := new(model.Card)
	card.CreatedAt = time.Now()

	if err := c.BodyParser(card); err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Couldn't create product", "data": err})
	}

	if card.Content == "" || card.Id == "" {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Couldn't create product"})
	}
	cards = append(cards, *card)
	return c.JSON(fiber.Map{"status": "success", "message": "Created card", "data": card})
}

func UpdateCard(c *fiber.Ctx) error {
	id := c.Params("id")
	card := new(model.Card)
	var cardIndex = findId(id)

	if cardIndex == -1 {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Карты с таким id не существует"})
	}

	if err := c.BodyParser(card); err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Couldn't update product", "data": err})
	}

	if card.Content == "" {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Couldn't update product"})
	}
	card.CreatedAt = time.Now()
	card.Id = id
	deleteFromCards(cardIndex)
	cards = append(cards, *card)

	return c.JSON(fiber.Map{"status": "success", "message": "Updated card", "data": card})

}

func DeleteCard(c *fiber.Ctx) error {
	id := c.Params("id")
	var cardIndex = findId(id)
	if cardIndex == -1 {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Карты с таким id не существует"})
	}
	deleteFromCards(cardIndex)
	return c.JSON(fiber.Map{"status": "success", "message": "Deleted card", "data": cards})
}

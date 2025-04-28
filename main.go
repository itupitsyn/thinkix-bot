package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"os"
	"os/signal"
	"time"
)

type Order struct {
	id       int64
	name     string
	email    string
	whatsapp *string
	telegram *string
}

var ticker = time.NewTicker(30 * time.Minute)
var quit = make(chan struct{})

func loadEnv() {
	err := godotenv.Load(".env.local")
	if err != nil {
		panic("Error loading .env file")
	}
}

func main() {
	loadEnv()

	groupId := os.Getenv("GROUP_ID")
	if groupId == "" {
		panic("Group id is not set")
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected to database")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{}

	b, err := bot.New(os.Getenv("TELEGRAM_APITOKEN"), opts...)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				rows, err := db.Query("SELECT id, name, email, whatsapp, telegram FROM \"Order\" WHERE \"notificationSentAt\" IS NULL;")
				if err != nil {
					fmt.Println("Error getting orders from database")
					fmt.Println(err)
					break
				}

				for rows.Next() {
					var kek Order
					if err := rows.Scan(&kek.id, &kek.name, &kek.email, &kek.whatsapp, &kek.telegram); err != nil {
						fmt.Println(err)
						continue
					}
					text := fmt.Sprintf("Пришла новая заявочка\nИмя: %s\nEmail: %s", kek.name, kek.email)
					if kek.telegram != nil {
						text += fmt.Sprintf("\nTelegram: %s", *kek.telegram)
					}
					if kek.whatsapp != nil {
						text += fmt.Sprintf("\nWhatsApp: %s", *kek.whatsapp)
					}

					if _, err = b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: groupId,
						Text:   text,
					}); err != nil {
						fmt.Println("Error sending message")
						fmt.Println(err)
						continue
					}

					if _, err = db.Exec("UPDATE \"Order\" SET \"notificationSentAt\"=$1 WHERE \"id\"=$2;", time.Now().Format(time.RFC3339), kek.id); err != nil {
						fmt.Println("Error updating columns")
						fmt.Println(err)
					}
				}
				rows.Close()
			case <-quit:
				ticker.Stop()
				fmt.Println("Stop checking updates")
				return
			}
		}
	}()

	b.Start(ctx)
}

// func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
// 	fmt.Println(update.Message.Chat.ID)
// 	// b.SendMessage(ctx, &bot.SendMessageParams{
// 	// 	ChatID: update.Message.Chat.ID,
// 	// 	Text:   update.Message.Text,
// 	// })
// }

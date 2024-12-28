package main

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	API_ID   = api_id
	API_HASH = "api_hash"
	BOT_TOKEN = "bottoken"
	OWNER_ID  = owner_id
)

var (
	mongoURI    = "mongodburi://localhost:8080"
	database    = "autodeletebot"
	usersCol    = "users"
	settingsCol = "settings"
)

func addUser(ctx context.Context, client *mongo.Client, userID int64) {
	usersCollection := client.Database(database).Collection(usersCol)
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"user_id": userID, "joined_at": time.Now()}}
	opts := options.Update().SetUpsert(true)
	_, err := usersCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Printf("Error adding user: %v\n", err)
	}
}

func addGroupSettings(ctx context.Context, client *mongo.Client, chatID int64, timeStr string) {
	settingsCollection := client.Database(database).Collection(settingsCol)
	filter := bson.M{"chat_id": chatID}
	update := bson.M{"$set": bson.M{"chat_id": chatID, "delete_time": timeStr}}
	opts := options.Update().SetUpsert(true)
	_, err := settingsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Printf("Error adding group settings: %v\n", err)
	}
}

func removeGroupSettings(ctx context.Context, client *mongo.Client, chatID int64) {
	settingsCollection := client.Database(database).Collection(settingsCol)
	_, err := settingsCollection.DeleteOne(ctx, bson.M{"chat_id": chatID})
	if err != nil {
		log.Printf("Error removing group settings: %v\n", err)
	}
}

func getGroupSettings(ctx context.Context, client *mongo.Client, chatID int64) string {
	settingsCollection := client.Database(database).Collection(settingsCol)
	var result struct {
		DeleteTime string `bson:"delete_time"`
	}
	err := settingsCollection.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&result)
	if err != nil {
		log.Printf("Error fetching group settings: %v\n", err)
		return ""
	}
	return result.DeleteTime
}

func parseDeleteTime(deleteTimeStr string) (time.Duration, error) {
	unit := deleteTimeStr[len(deleteTimeStr)-1]
	value, err := strconv.Atoi(deleteTimeStr[:len(deleteTimeStr)-1])
	if err != nil {
		return 0, err
	}

	switch unit {
	case 's':
		return time.Duration(value) * time.Second, nil
	case 'm':
		return time.Duration(value) * time.Minute, nil
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	}
	return 0, nil
}

func main() {
	ctx := context.TODO()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB connection error: %v\n", err)
	}
	defer mongoClient.Disconnect(ctx)

	bot, err := tgbotapi.NewBotAPI(BOT_TOKEN)
	if err != nil {
		log.Fatalf("Error initializing bot: %v\n", err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		switch update.Message.Command() {
		case "start":
			go addUser(ctx, mongoClient, update.Message.From.ID)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi! I'm an auto-delete bot!")
			bot.Send(msg)

		case "set_time":
			if update.Message.Chat.IsPrivate() {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Use this command in a group!")
				bot.Send(msg)
				continue
			}

			args := strings.Split(update.Message.Text, " ")
			if len(args) < 2 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Please provide a valid time! E.g., 1s, 1m, 1h, 1d.")
				bot.Send(msg)
				continue
			}

			deleteTime := args[1]
			go addGroupSettings(ctx, mongoClient, update.Message.Chat.ID, deleteTime)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Auto delete time set!")
			bot.Send(msg)

		case "stop_del":
			if update.Message.Chat.IsPrivate() {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Use this command in a group!")
				bot.Send(msg)
				continue
			}

			go removeGroupSettings(ctx, mongoClient, update.Message.Chat.ID)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Auto delete stopped!")
			bot.Send(msg)

		case "stats":
			if update.Message.From.ID != OWNER_ID {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You are not authorized to use this command!")
				bot.Send(msg)
				continue
			}

			usersCount, err := mongoClient.Database(database).Collection(usersCol).CountDocuments(ctx, bson.M{})
			if err != nil {
				log.Printf("Error fetching users count: %v\n", err)
			}

			groupsCount, err := mongoClient.Database(database).Collection(settingsCol).CountDocuments(ctx, bson.M{})
			if err != nil {
				log.Printf("Error fetching groups count: %v\n", err)
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"Stats:\n"+
					"Users: " + strconv.FormatInt(usersCount, 10) + "\n" +
					"Groups: " + strconv.FormatInt(groupsCount, 10))
			bot.Send(msg)
		}

		if !update.Message.Chat.IsPrivate() {
			deleteTimeStr := getGroupSettings(ctx, mongoClient, update.Message.Chat.ID)
			if deleteTimeStr != "" {
				deleteTimeDuration, err := parseDeleteTime(deleteTimeStr)
				if err == nil {
					go func() {
						time.Sleep(deleteTimeDuration)
						deleteMsg := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
						bot.Send(deleteMsg)

						if update.Message.ReplyToMessage != nil {
							deleteBotMessage := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.ReplyToMessage.MessageID)
							bot.Send(deleteBotMessage)
						}
					}()
				}
			}
		}
	}
}

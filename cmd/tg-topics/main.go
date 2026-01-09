package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/gotd/td/tg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tg-topics @channel_username")
		fmt.Println("example: tg-topics @golang_jobs")
		os.Exit(1)
	}

	username := strings.TrimPrefix(os.Args[1], "@")
	ctx := context.Background()

	// read credentials from env
	apiIDStr := os.Getenv("TG_API_ID")
	apiHash := os.Getenv("TG_API_HASH")
	sessionString := os.Getenv("TG_SESSION_STRING")

	if apiIDStr == "" || apiHash == "" || sessionString == "" {
		fmt.Println("error: missing required environment variables")
		fmt.Println("please set: TG_API_ID, TG_API_HASH, TG_SESSION_STRING")
		os.Exit(1)
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		fmt.Printf("error: invalid TG_API_ID: %v\n", err)
		os.Exit(1)
	}

	// create telegram client with string session
	client, err := gotgproto.NewClient(
		apiID,
		apiHash,
		gotgproto.ClientTypePhone(""),
		&gotgproto.ClientOpts{
			Session:          sessionMaker.StringSession(sessionString),
			DisableCopyright: true,
			InMemory:         true, // don't write to disk
		},
	)
	if err != nil {
		fmt.Printf("error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Stop()

	fmt.Printf("fetching topics for @%s...\n\n", username)

	// resolve channel username
	resolved, err := client.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		fmt.Printf("error resolving username: %v\n", err)
		os.Exit(1)
	}

	if len(resolved.Chats) == 0 {
		fmt.Printf("channel @%s not found\n", username)
		os.Exit(1)
	}

	channel, ok := resolved.Chats[0].(*tg.Channel)
	if !ok {
		fmt.Printf("@%s is not a channel\n", username)
		os.Exit(1)
	}

	// get full channel info to check if it's a forum
	fullChannel, err := client.API().ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  channel.ID,
		AccessHash: channel.AccessHash,
	})
	if err != nil {
		fmt.Printf("error getting channel info: %v\n", err)
		os.Exit(1)
	}

	chFull, ok := fullChannel.FullChat.(*tg.ChannelFull)
	if !ok {
		fmt.Printf("unexpected channel type\n")
		os.Exit(1)
	}

	// check if it's a forum (flag 30)
	if !chFull.Flags.Has(30) {
		fmt.Printf("@%s is not a forum (no topics available)\n", username)
		fmt.Println("this tool only works with forum-type supergroups")
		os.Exit(0)
	}

	// get forum topics
	result, err := client.API().MessagesGetForumTopics(ctx, &tg.MessagesGetForumTopicsRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		Limit: 100,
	})
	if err != nil {
		fmt.Printf("error fetching topics: %v\n", err)
		os.Exit(1)
	}

	topics := result

	// display results
	fmt.Printf("forum: %s (@%s)\n", channel.Title, username)
	fmt.Printf("total topics: %d\n\n", len(topics.Topics))

	fmt.Printf("%-8s | %-30s | %-10s | %-10s\n", "id", "title", "messages", "status")
	fmt.Println(strings.Repeat("-", 70))

	for _, t := range topics.Topics {
		topic, ok := t.(*tg.ForumTopic)
		if !ok {
			continue
		}

		status := "open"
		if topic.Closed {
			status = "closed"
		}

		// truncate long titles
		title := topic.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		fmt.Printf("%-8d | %-30s | %-10d | %-10s\n",
			topic.ID,
			title,
			topic.TopMessage,
			status,
		)
	}

	fmt.Println("\nto parse specific topics, use their ids in the scrape request:")
	fmt.Println(`  "topic_ids": [1, 15, 28]`)
}

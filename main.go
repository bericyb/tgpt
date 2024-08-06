package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type Chat struct {
	conversation []string
}

func main() {

	var red = "\x1b[31m"
	var green = "\x1b[32m"
	var blue = "\x1b[35m"

	client := openai.NewClient(os.Getenv("OPENAI_KEY"))
	ctx := context.Background()

	scanner := bufio.NewScanner(os.Stdin)
	messages := []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful ai assistant that is a unix wizard that helps with not only code, but also general terminal wizardry. Answer always with the code only itself first and an explaination second. Don't bother with markdown formatting.",
		}}

	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, syscall.SIGINT)

	fmt.Println(blue, "User:")
	for scanner.Scan() {

		line := scanner.Text()

		if strings.ToLower(line) == "exit" {
			fmt.Println(red, "Exiting and saving chat...")

			data, err := json.Marshal(messages)
			if err != nil {
				fmt.Println("Error saving chat... abandoning chat history, sorry!", err)
			}

			if err := os.MkdirAll("/var/lib/tgpt", 0755); err != nil {
				fmt.Println("Error saving chat.. abandoning chat history, sorry!", err)
			}

			err = os.WriteFile(fmt.Sprintf("/var/lib/tgpt/%s", time.Now().Format("20060102150405")), data, 0644)
			if err != nil {
				fmt.Println("Error writing chat to file... abandoning chat history, sorry!", err)
			}

			fmt.Println(green, "Goodbye!")
			return
		}

		newMessage := openai.ChatCompletionMessage{
			Role:    "user",
			Content: line,
		}

		req := openai.ChatCompletionRequest{
			Model:    openai.GPT4Turbo,
			Messages: append(messages, newMessage),
			Stream:   true,
		}

		stream, err := client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			fmt.Print("Error completing chat request.", err)
			return
		}

		defer stream.Close()

		respMessage := ""

		fmt.Println(green, "GPT:")
	messageLoop:
		for {
			select {
			case signal := <-sigChan:
				if signal == syscall.SIGINT {
					break messageLoop
				}
			default:
			}
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				fmt.Println("Error completing chat request.", err)
				break
			}

			fmt.Printf(response.Choices[0].Delta.Content)
			respMessage = respMessage + response.Choices[0].Delta.Content
		}
		fmt.Print("\n")

		messages = append(messages, openai.ChatCompletionMessage{Role: "assistant", Content: respMessage})

		fmt.Println(blue, "User:")
	}

}

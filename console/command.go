package console

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Command struct{}

func Ask(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(prompt + " ")
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func AskWithDefault(prompt, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(prompt + " ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}

	return input
}

func AskYesOrNo(prompt string, defaultYes bool) bool {
	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}
	for {
		fmt.Printf("%s [%s]: ", prompt, defaultStr)
		reader := bufio.NewReader(os.Stdin)
		fmt.Println(prompt + " ")
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "" {
			return defaultYes
		}

		if input == "y" || input == "yes" {
			return true
		}

		if input == "n" || input == "no" {
			return false
		}
		Info("Please answer yes or no.")
	}

}

func AskWithValidator(prompt string, validate func(string) error) string {
	reader := bufio.NewReader(os.Stdin)
	for {
		Info(prompt + " ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if err := validate(input); err != nil {
			Info("Error:", err)
			continue
		}

		return input
	}

}

func Info(a ...any) {
	fmt.Println(a...)
}

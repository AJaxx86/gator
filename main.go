package main

import (
	"github.com/ajaxx86/gator/internal/config"
	"fmt"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}
	fmt.Println("Config:", cfg)

	cfg.SetUser("jaxx")
	cfg, err = config.Read()
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}
	fmt.Println("Updated config:", cfg)
}

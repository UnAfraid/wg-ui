package main

import (
	"flag"
	"log"
	"os"

	"github.com/UnAfraid/dataloaden/v2/pkg/generator"
)

func main() {
	var (
		name      string
		fileName  string
		keyType   string
		valueType string
	)

	flag.StringVar(&name, "name", "", "the data loader name, example: UserLoader")
	flag.StringVar(&fileName, "fileName", "", "the generated output file name")
	flag.StringVar(&keyType, "keyType", "", "the data loader key type, example: int64")
	flag.StringVar(&valueType, "valueType", "", "the data loader value type, example: *github.com/UnAfraid/dataloaden/example.User")
	flag.Parse()

	if len(name) == 0 || len(keyType) == 0 || len(valueType) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	if err := generator.Generate(name, fileName, keyType, valueType, wd); err != nil {
		log.Printf("failed to generate data loader for %s - %v", name, err.Error())
		os.Exit(1)
	}
}

package main

import (
	"k8s-update-deployment-ecr-tag/webhook/api"
	"log"
)

func main() {
	err := api.StartServer()
	if err != nil {
		log.Fatal(err)
	}
}

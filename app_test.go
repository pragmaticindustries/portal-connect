package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-co-op/gocron"
	"os"
	"testing"
	"time"
)

type Book struct {
	Name   string
	Author string
}

func task1(name string, age int) {
	fmt.Printf("run task1, with arguments: %s, %d\n", name, age)
}

func task2() {
	fmt.Println("run task2, without arguments")
}

func Test_Scheduler(t *testing.T) {
	s := gocron.NewScheduler(time.UTC)

	// delay with 1 second, job function with arguments
	_, err := s.Every(2).Seconds().Do(task1, "prprprus", 23)
	if err != nil {
		fmt.Println("There was an error...")
	}

	_, error2 := s.Every(1).Seconds().Do(task1, "asdf", 24)
	if error2 != nil {
		fmt.Println("There was an error...")
	}

	s.StartBlocking()
}

func Test_connectMQTT(t *testing.T) {
	err := os.Setenv("MQTT_USER", "julian")
	user := os.Getenv("MQTT_USER")

	fmt.Println("User: " + user)

	jsonString := `{"Name":"Test", "Author": "Author"}`

	var book Book

	err = json.Unmarshal([]byte(jsonString), &book)

	if err != nil {
		t.Errorf("connectMQTT()")
	}

	expected := Book{"Test", "Author"}
	if expected != book {
		t.Errorf("connectMQTT()")
	}
}

func Test_Loop(t *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Println("i =", i)
	}
}

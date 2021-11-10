package main

import (
	"fmt"
	"time"

	"github.com/apache/plc4x/plc4go/pkg/plc4go"
	"github.com/apache/plc4x/plc4go/pkg/plc4go/drivers"
	"github.com/apache/plc4x/plc4go/pkg/plc4go/transports"

	"os"

	"github.com/go-co-op/gocron"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func main() {
	// PLC connection string
	var plcConStr = getenv("PLC_ADDRESS", "s7://192.168.167.210/0/0")

	c := connectMQTT()
	defer disconnectMQTT(c)
	readPlc(plcConStr, c)
}

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func connectMQTT() MQTT.Client {
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://mqtt.eclipseprojects.io:1883")
	opts.SetClientID("portal-connect")
	opts.SetDefaultPublishHandler(f)

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	//subscribe to the topic /portal-test/sample and request messages to be delivered
	//at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe("portal-test/sample", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	return c
}

func disconnectMQTT(c MQTT.Client) {
	//unsubscribe from /portal-test/sample
	if token := c.Unsubscribe("portal-test/sample"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	c.Disconnect(250)
}

func publishMQTT(c MQTT.Client, label string, value string) {
	text := fmt.Sprintf("key: %s, value: %s", label, value)
	token := c.Publish("portal-test/sample", 0, false, text)
	token.Wait()
}

func readPlc(plcConStr string, c MQTT.Client) {
	//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	// Follow this: http://plc4x.apache.org/users/getting-started/plc4go.html
	//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

	// Create a new instance of the PlcDriverManager
	driverManager := plc4go.NewPlcDriverManager()

	// Register the Transports
	transports.RegisterTcpTransport(driverManager)
	transports.RegisterUdpTransport(driverManager)

	// Register the Drivers
	drivers.RegisterS7Driver(driverManager)

	// Get a connection to a remote PLC
	connectionRequestChanel := driverManager.GetConnection(plcConStr)

	// Wait for the driver to connect (or not)
	connectionResult := <-connectionRequestChanel

	// Check if something went wrong
	if connectionResult.Err != nil {
		fmt.Printf("Error connecting to PLC: %s", connectionResult.Err.Error())
		return
	}

	// If all was ok, get the connection instance
	connection := connectionResult.Connection

	// Make sure the connection is closed at the end
	defer connection.Close()

	// Try to ping the remote device
	pingResultChannel := connection.Ping()

	// Wait for the Ping operation to finsh
	pingResult := <-pingResultChannel
	if pingResult.Err != nil {
		fmt.Printf("Couldn't ping device: %s", pingResult.Err.Error())
		return
	}

	if !connection.GetMetadata().CanRead() {
		fmt.Printf("This connection doesn't support read operations")
		return
	}

	parameters := map[string]string{
		"motor-current": "%DB444.DBD8:REAL",
		"position":      "%DB444.DBD0:REAL",
		"rand_val":      "%DB444.DBD4:REAL",
	}

	s := gocron.NewScheduler(time.UTC)

	// Shedule the Event
	s.Every(5).Seconds().Do(performRequest, c, connection, parameters, connectionResult)

	// Run the main "event loop"
	s.StartBlocking()
}

func performRequest(c MQTT.Client, connection plc4go.PlcConnection, parameters map[string]string, connectionResult plc4go.PlcConnectionConnectResult) {
	fmt.Printf("Doing a Request now...")
	// Prepare a read-request
	builder := connection.ReadRequestBuilder()

	for key, value := range parameters {
		builder.AddQuery(key, value)
	}

	readRequest, err := builder.Build()
	if err != nil {
		fmt.Printf("error preparing read-request: %s", connectionResult.Err.Error())
		return
	}

	// Execute a read-request
	readResponseChanel := readRequest.Execute()

	// Wait for the response to finish
	readRequestResult := <-readResponseChanel
	if readRequestResult.Err != nil {
		fmt.Printf("error executing read-request: %s", readRequestResult.Err.Error())
		return
	}

	// Do something with the response
	for key := range parameters {
		val := readRequestResult.Response.GetValue(key)
		publishMQTT(c, "motor-current", val.GetString())
	}
}

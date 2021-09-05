package main

import (
	"fmt"

	"github.com/apache/plc4x/plc4go/pkg/plc4go"
	"github.com/apache/plc4x/plc4go/pkg/plc4go/drivers"
	"github.com/apache/plc4x/plc4go/pkg/plc4go/transports"

	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	pingPLC()
	connectMQTT()
}

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

// PLC connection string
var plcConStr = "modbus://192.168.23.30?unit-identifier=1"

func connectMQTT() {
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://mqtt.eclipseprojects.io:1883")
	opts.SetClientID("go-simple")
	opts.SetDefaultPublishHandler(f)

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	//subscribe to the topic /go-mqtt/sample and request messages to be delivered
	//at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe("go-mqtt/sample", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	//Publish 5 messages to /go-mqtt/sample at qos 1 and wait for the receipt
	//from the server after sending each message
	for i := 0; i < 5; i++ {
		text := fmt.Sprintf("this is msg #%d!", i)
		token := c.Publish("go-mqtt/sample", 0, false, text)
		token.Wait()
	}

	time.Sleep(3 * time.Second)

	//unsubscribe from /go-mqtt/sample
	if token := c.Unsubscribe("go-mqtt/sample"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	c.Disconnect(250)
}

func pingPLC() {
	//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	// Follow this: http://plc4x.apache.org/users/getting-started/plc4go.html
	//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

	// Create a new instance of the PlcDriverManager
	driverManager := plc4go.NewPlcDriverManager()

	// Register the Transports
	transports.RegisterTcpTransport(driverManager)
	transports.RegisterUdpTransport(driverManager)

	// Register the Drivers
	drivers.RegisterKnxDriver(driverManager)
	drivers.RegisterModbusDriver(driverManager)

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
}

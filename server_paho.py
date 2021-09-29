
import random
import time
import os
import json

from dotenv import load_dotenv

from paho.mqtt import client as mqtt_client

load_dotenv()

tenantId = os.getenv('HONOTENANT')
deviceId = os.getenv('HONODEVICE')
plcStr = os.getenv('PLCURL')

broker = os.getenv('HONOMQTTIP')
port = 1883
topic = f"command/{tenantId}/{deviceId}"
# generate client ID with pub prefix randomly
client_id = f'python-mqtt-{random.randint(0, 1000)}'
username=f"{os.getenv('HONODEVICE')}@{os.getenv('HONOTENANT')}"
password=os.getenv('HONODEVICEPASSWORD')

def connect_mqtt():
    def on_connect(client, userdata, flags, rc):
        if rc == 0:
            print("Connected to MQTT Broker!")
        else:
            print("Failed to connect, return code %d\n", rc)

    client = mqtt_client.Client(client_id)
    client.username_pw_set(username, password)
    client.on_connect = on_connect
    client.connect(broker, port)
    return client


def publish(client):
    while True:
        time.sleep(1)
        msg = json.dumps({"host": plcStr,"plcField":"%DB444:6.0:REAL"})
        result = client.publish(topic, msg, 1)
        # result: [0, 1]
        status = result[0]
        if status == 0:
            print(f"Send `{msg}` to topic `{topic}`")
        else:
            print(f"Failed to send message to topic {topic}")
            print(status)
            print(result)


def run():
    client = connect_mqtt()
    client.loop_start()
    publish(client)


if __name__ == '__main__':
    run()
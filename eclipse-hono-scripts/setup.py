import json
import requests as requests

import os
from dotenv import load_dotenv
from pathlib import Path

dotenv_path = Path('../.env')
load_dotenv(dotenv_path=dotenv_path)

devicePassword = os.getenv('HONODEVICEPASSWORD')
registryIp = os.getenv('HONOREGISTRYIP')
mqttAdapterIp = os.getenv('HONOMQTTIP') #mqtt Port 1883

tenant = requests.post(f'http://{registryIp}:28080/v1/tenants').json()
tenantId = tenant["id"]

f = open("../.env", "a")

print(f'Registered tenant {tenantId}')

f.write("tenant:")
f.write(tenantId)

# Add Device to Tenant
device = requests.post(f'http://{registryIp}:28080/v1/devices/{tenantId}').json()
deviceId = device["id"]
print(f'Registered device {deviceId}')

f.write(", device:")
f.write(deviceId)
f.write(", password:")
f.write(devicePassword)
f.close()

code = requests.put(f'http://{registryIp}:28080/v1/credentials/{tenantId}/{deviceId}',
                        headers={'content-type': 'application/json'},
                        data=json.dumps(
                            [{"type": "hashed-password", "auth-id": deviceId,
                            "secrets": [{"pwd-plain": devicePassword}]}]))

if code.status_code == 204:
    print("Password is set!")
else:
    print("Unnable to set Password")